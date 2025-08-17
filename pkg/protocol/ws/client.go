package ws

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v3"
	"orly.dev/pkg/encoders/envelopes"
	"orly.dev/pkg/encoders/envelopes/authenvelope"
	"orly.dev/pkg/encoders/envelopes/closedenvelope"
	"orly.dev/pkg/encoders/envelopes/eoseenvelope"
	"orly.dev/pkg/encoders/envelopes/eventenvelope"
	"orly.dev/pkg/encoders/envelopes/noticeenvelope"
	"orly.dev/pkg/encoders/envelopes/okenvelope"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/interfaces/signer"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/normalize"
)

var subscriptionIDCounter atomic.Int64

// Relay represents a connection to a Nostr relay.
type Client struct {
	closeMutex sync.Mutex

	URL           string
	requestHeader http.Header // e.g. for origin header

	Connection    *Connection
	Subscriptions *xsync.MapOf[int64, *Subscription]

	ConnectionError         error
	connectionContext       context.T // will be canceled when the connection closes
	connectionContextCancel context.C

	challenge                     []byte       // NIP-42 challenge, we only keep the last
	noticeHandler                 func(string) // NIP-01 NOTICEs
	customHandler                 func(string) // nonstandard unparseable messages
	okCallbacks                   *xsync.MapOf[string, func(bool, string)]
	writeQueue                    chan writeRequest
	subscriptionChannelCloseQueue chan *Subscription

	// custom things that aren't often used
	//
	AssumeValid bool // this will skip verifying signatures for events received from this relay
}

type writeRequest struct {
	msg    []byte
	answer chan error
}

// NewRelay returns a new relay. It takes a context that, when canceled, will close the relay connection.
func NewRelay(ctx context.T, url string, opts ...RelayOption) *Client {
	ctx, cancel := context.Cause(ctx)
	r := &Client{
		URL:                     string(normalize.URL(url)),
		connectionContext:       ctx,
		connectionContextCancel: cancel,
		Subscriptions:           xsync.NewMapOf[int64, *Subscription](),
		okCallbacks: xsync.NewMapOf[string, func(
			bool, string,
		)](),
		writeQueue:                    make(chan writeRequest),
		subscriptionChannelCloseQueue: make(chan *Subscription),
		requestHeader:                 nil,
	}

	for _, opt := range opts {
		opt.ApplyRelayOption(r)
	}

	return r
}

// RelayConnect returns a relay object connected to url.
//
// The given subscription is only used during the connection phase. Once successfully connected, cancelling ctx has no effect.
//
// The ongoing relay connection uses a background context. To close the connection, call r.Close().
// If you need fine grained long-term connection contexts, use NewRelay() instead.
func RelayConnect(ctx context.T, url string, opts ...RelayOption) (
	*Client, error,
) {
	r := NewRelay(context.Bg(), url, opts...)
	err := r.Connect(ctx)
	return r, err
}

// RelayOption is the type of the argument passed when instantiating relay connections.
type RelayOption interface {
	ApplyRelayOption(*Client)
}

var (
	_ RelayOption = (WithNoticeHandler)(nil)
	_ RelayOption = (WithCustomHandler)(nil)
	_ RelayOption = (WithRequestHeader)(nil)
)

// WithNoticeHandler just takes notices and is expected to do something with them.
// when not given, defaults to logging the notices.
type WithNoticeHandler func(notice string)

func (nh WithNoticeHandler) ApplyRelayOption(r *Client) {
	r.noticeHandler = nh
}

// WithCustomHandler must be a function that handles any relay message that couldn't be
// parsed as a standard envelope.
type WithCustomHandler func(data string)

func (ch WithCustomHandler) ApplyRelayOption(r *Client) {
	r.customHandler = ch
}

// WithRequestHeader sets the HTTP request header of the websocket preflight request.
type WithRequestHeader http.Header

func (ch WithRequestHeader) ApplyRelayOption(r *Client) {
	r.requestHeader = http.Header(ch)
}

// String just returns the relay URL.
func (r *Client) String() string {
	return r.URL
}

// Context retrieves the context that is associated with this relay connection.
// It will be closed when the relay is disconnected.
func (r *Client) Context() context.T { return r.connectionContext }

// IsConnected returns true if the connection to this relay seems to be active.
func (r *Client) IsConnected() bool { return r.connectionContext.Err() == nil }

// Connect tries to establish a websocket connection to r.URL.
// If the context expires before the connection is complete, an error is returned.
// Once successfully connected, context expiration has no effect: call r.Close
// to close the connection.
//
// The given context here is only used during the connection phase. The long-living
// relay connection will be based on the context given to NewRelay().
func (r *Client) Connect(ctx context.T) error {
	return r.ConnectWithTLS(ctx, nil)
}

func subIdToSerial(subId string) int64 {
	n := strings.Index(subId, ":")
	if n < 0 || n > len(subId) {
		return -1
	}
	serialId, _ := strconv.ParseInt(subId[0:n], 10, 64)
	return serialId
}

// ConnectWithTLS is like Connect(), but takes a special tls.Config if you need that.
func (r *Client) ConnectWithTLS(
	ctx context.T, tlsConfig *tls.Config,
) (err error) {
	if r.connectionContext == nil || r.Subscriptions == nil {
		return fmt.Errorf("relay must be initialized with a call to NewRelay()")
	}

	if r.URL == "" {
		return fmt.Errorf("invalid relay URL '%s'", r.URL)
	}

	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.F
		ctx, cancel = context.TimeoutCause(
			ctx, 7*time.Second, errors.New("connection took too long"),
		)
		defer cancel()
	}
	var conn *Connection
	if conn, err = NewConnection(
		ctx, r.URL, r.requestHeader, tlsConfig,
	); chk.E(err) {
		err = fmt.Errorf("error opening websocket to '%s': %w", r.URL, err)
		return
	}
	r.Connection = conn
	// ping every 29 seconds
	ticker := time.NewTicker(29 * time.Second)
	// queue all write operations here so we don't do mutex spaghetti
	go func() {
		for {
			select {
			case <-r.connectionContext.Done():
				ticker.Stop()
				r.Connection = nil
				for _, sub := range r.Subscriptions.Range {
					sub.unsub(
						fmt.Errorf(
							"relay connection closed: %w / %w",
							context.GetCause(r.connectionContext),
							r.ConnectionError,
						),
					)
				}
				return
			case <-ticker.C:
				err := r.Connection.Ping(r.connectionContext)
				if err != nil && !strings.Contains(
					err.Error(), "failed to wait for pong",
				) {
					log.T.C(
						func() string {
							return fmt.Sprintf(
								"{%s} error writing ping: %v; closing websocket",
								r.URL,
								err,
							)
						},
					)
					r.Close() // this should trigger a context cancelation
					return
				}
			case writeRequest := <-r.writeQueue:
				// all write requests will go through this to prevent races
				log.T.C(
					func() string {
						return fmt.Sprintf(
							"{%s} sending %v\n", r.URL,
							string(writeRequest.msg),
						)
					},
				)
				if err := r.Connection.WriteMessage(
					r.connectionContext, writeRequest.msg,
				); err != nil {
					writeRequest.answer <- err
				}
				close(writeRequest.answer)
			}
		}
	}()
	// general message reader loop
	go func() {

		for {
			buf := new(bytes.Buffer)
			if err := conn.ReadMessage(r.connectionContext, buf); err != nil {
				r.ConnectionError = err
				r.close(err)
				break
			}
			var err error
			var t string
			var rem []byte
			if t, rem, err = envelopes.Identify(buf.Bytes()); chk.E(err) {
				continue
			}
			switch t {
			case noticeenvelope.L:
				env := noticeenvelope.NewFrom(rem)
				// see WithNoticeHandler
				if r.noticeHandler != nil {
					r.noticeHandler(string(env.Message))
				} else {
					log.D.F(
						"NOTICE from %s: '%s'\n", r.URL, string(env.Message),
					)
				}
			case authenvelope.L:
				env := authenvelope.NewChallengeWith(rem)
				if env.Challenge == nil {
					continue
				}
				r.challenge = env.Challenge
			case eventenvelope.L:
				// log.I.F("%s", rem)
				var env *eventenvelope.Result
				env = eventenvelope.NewResult()
				if _, err = env.Unmarshal(rem); err != nil {
					continue
				}
				subid := env.Subscription.String()
				sub, ok := r.Subscriptions.Load(subIdToSerial(subid))
				if !ok {
					log.W.F(
						"unknown subscription with id '%s'\n",
						subid,
					)
					continue
				}
				if !sub.Filters.Match(env.Event) {
					log.T.C(
						func() string {
							return fmt.Sprintf(
								"{%s} filter does not match: %v ~ %s\n", r.URL,
								sub.Filters, env.Event.Marshal(nil),
							)
						},
					)
					continue
				}
				if !r.AssumeValid {
					if ok, err = env.Event.Verify(); !ok || chk.E(err) {
						log.T.C(
							func() string {
								return fmt.Sprintf(
									"{%s} bad signature on %s\n", r.URL,
									env.Event.ID,
								)
							},
						)
						continue
					}
				}
				sub.dispatchEvent(env.Event)
			case eoseenvelope.L:
				var env *eoseenvelope.T
				if env, rem, err = eoseenvelope.Parse(rem); chk.E(err) {
					continue
				}
				if len(rem) != 0 {
					log.W.F(
						"{%s} unexpected data after EOSE: %s\n", r.URL,
						string(rem),
					)
				}
				sub, ok := r.Subscriptions.Load(subIdToSerial(env.Subscription.String()))
				if !ok {
					log.W.F(
						"unknown subscription with id '%s'\n",
						env.Subscription.String(),
					)
					continue
				}
				sub.dispatchEose()
			case closedenvelope.L:
				var env *closedenvelope.T
				if env, rem, err = closedenvelope.Parse(rem); chk.E(err) {
					continue
				}
				sub, ok := r.Subscriptions.Load(subIdToSerial(env.Subscription.String()))
				if !ok {
					log.W.F(
						"unknown subscription with id '%s'\n",
						env.Subscription.String(),
					)
					continue
				}
				sub.handleClosed(env.ReasonString())
			case okenvelope.L:
				var env *okenvelope.T
				if env, rem, err = okenvelope.Parse(rem); chk.E(err) {
					continue
				}
				eventIDStr := env.EventID.String()
				if okCallback, exist := r.okCallbacks.Load(eventIDStr); exist {
					okCallback(env.OK, string(env.Reason))
				} else {
					log.T.C(
						func() string {
							return fmt.Sprintf(
								"{%s} got an unexpected OK message for event %s",
								r.URL,
								eventIDStr,
							)
						},
					)
				}
			default:
				log.W.F("unknown envelope type %s\n%s", t, rem)
				continue
			}
		}
	}()
	return
}

// Write queues an arbitrary message to be sent to the relay.
func (r *Client) Write(msg []byte) <-chan error {
	ch := make(chan error)
	select {
	case r.writeQueue <- writeRequest{msg: msg, answer: ch}:
	case <-r.connectionContext.Done():
		go func() { ch <- fmt.Errorf("connection closed") }()
	}
	return ch
}

// Publish sends an "EVENT" command to the relay r as in NIP-01 and waits for an
// OK response.
func (r *Client) Publish(ctx context.T, ev *event.E) error {
	return r.publish(ctx, ev.ID, ev)
}

// Auth sends an "AUTH" command client->relay as in NIP-42 and waits for an OK
// response.
func (r *Client) Auth(
	ctx context.T, sign signer.I,
) (err error) {
	authEvent := &event.E{
		CreatedAt: timestamp.Now(),
		Kind:      kind.ClientAuthentication,
		Tags: tags.New(
			tag.New("relay", r.URL),
			tag.New([]byte("challenge"), r.challenge),
		),
	}
	if err = authEvent.Sign(sign); chk.E(err) {
		err = fmt.Errorf("error signing auth event: %w", err)
		return
	}
	return r.publish(ctx, authEvent.ID, authEvent)
}

func (r *Client) publish(
	ctx context.T, id []byte, ev *event.E,
) error {
	var err error
	var cancel context.F
	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		ctx, cancel = context.TimeoutCause(
			ctx, 7*time.Second, fmt.Errorf("given up waiting for an OK"),
		)
		defer cancel()
	} else {
		// otherwise make the context cancellable so we can stop everything upon
		// receiving an "OK"
		ctx, cancel = context.Cancel(ctx)
		defer cancel()
	}

	// listen for an OK callback
	gotOk := false
	ids := hex.Enc(id)
	r.okCallbacks.Store(
		ids, func(ok bool, reason string) {
			gotOk = true
			if !ok {
				err = fmt.Errorf("msg: %s", reason)
			}
			cancel()
		},
	)
	defer r.okCallbacks.Delete(ids)
	// publish event
	envb := eventenvelope.NewSubmissionWith(ev).Marshal(nil)
	// envb := ev.Marshal(nil)
	if err = <-r.Write(envb); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			// this will be called when we get an OK or when the context has
			// been canceled
			if gotOk {
				return err
			}
			return ctx.Err()
		case <-r.connectionContext.Done():
			// this is caused when we lose connectivity
			return err
		}
	}
}

// Subscribe sends a "REQ" command to the relay r as in NIP-01. Events are
// returned through the channel sub.Events. The subscription is closed when
// context ctx is cancelled ("CLOSE" in NIP-01).
//
// Remember to cancel subscriptions, either by calling `.Unsub()` on them or
// ensuring their `context.T` will be canceled at some point. Failure to
// do that will result in a huge number of halted goroutines being created.
func (r *Client) Subscribe(
	ctx context.T, ff *filters.T, opts ...SubscriptionOption,
) (sub *Subscription, err error) {
	sub = r.PrepareSubscription(ctx, ff, opts...)
	if r.Connection == nil {
		return nil, fmt.Errorf("not connected to %s", r.URL)
	}
	if err = sub.Fire(); err != nil {
		err = fmt.Errorf(
			"couldn't subscribe to %v at %s: %w", ff.Marshal(nil), r.URL, err,
		)
		return
	}
	return
}

// PrepareSubscription creates a subscription, but doesn't fire it.
//
// Remember to cancel subscriptions, either by calling `.Unsub()` on them or ensuring their `context.T` will be canceled at some point.
// Failure to do that will result in a huge number of halted goroutines being created.
func (r *Client) PrepareSubscription(
	ctx context.T, ff *filters.T, opts ...SubscriptionOption,
) *Subscription {
	current := subscriptionIDCounter.Add(1)
	ctx, cancel := context.Cause(ctx)
	sub := &Subscription{
		Relay:             r,
		Context:           ctx,
		cancel:            cancel,
		counter:           current,
		Events:            make(event.C),
		EndOfStoredEvents: make(chan struct{}, 1),
		ClosedReason:      make(chan string, 1),
		Filters:           ff,
		match:             ff.Match,
	}
	label := ""
	for _, opt := range opts {
		switch o := opt.(type) {
		case WithLabel:
			label = string(o)
			// case WithCheckDuplicate:
			// 	sub.checkDuplicate = o
			// case WithCheckDuplicateReplaceable:
			// 	sub.checkDuplicateReplaceable = o
		}
	}
	// subscription id computation
	buf := subIdPool.Get().([]byte)[:0]
	buf = strconv.AppendInt(buf, sub.counter, 10)
	buf = append(buf, ':')
	buf = append(buf, label...)
	defer subIdPool.Put(buf)
	sub.id = string(buf)
	// we track subscriptions only by their counter, no need for the full id
	r.Subscriptions.Store(int64(sub.counter), sub)
	// start handling events, eose, unsub etc:
	go sub.start()
	return sub
}

// QueryEvents subscribes to events matching the given filter and returns a channel of events.
//
// In most cases it's better to use Pool instead of this method.
func (r *Client) QueryEvents(ctx context.T, f *filter.F) (
	evc event.C, err error,
) {
	var sub *Subscription
	if sub, err = r.Subscribe(ctx, filters.New(f)); chk.E(err) {
		return
	}
	go func() {
		for {
			select {
			case <-sub.ClosedReason:
			case <-sub.EndOfStoredEvents:
			case <-ctx.Done():
			case <-r.Context().Done():
			}
			sub.unsub(errors.New("QueryEvents() ended"))
			return
		}
	}()
	return sub.Events, nil
}

// QuerySync subscribes to events matching the given filter and returns a slice
// of events. This method blocks until all events are received or the context is
// canceled.
//
// If the filter causes a subscription to open, it will stay open until the
// limit is exceeded. So this method will return an error if the limit is nil.
// If the query blocks, the caller needs to cancel the context to prevent the
// thread stalling.
func (r *Client) QuerySync(ctx context.T, f *filter.F) (
	evs event.S, err error,
) {
	if f.Limit == nil {
		err = errors.New("limit must be set for a sync query to prevent blocking")
		return
	}
	var sub *Subscription
	if sub, err = r.Subscribe(ctx, filters.New(f)); chk.E(err) {
		return
	}
	defer sub.unsub(errors.New("QuerySync() ended"))
	evs = make(event.S, 0, *f.Limit)
	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.F
		ctx, cancel = context.TimeoutCause(
			ctx, 7*time.Second, errors.New("QuerySync() took too long"),
		)
		defer cancel()
	}
	lim := 250
	if f.Limit != nil {
		lim = int(*f.Limit)
	}
	events := make(event.S, 0, max(lim, 250))
	ch, err := r.QueryEvents(ctx, f)
	if err != nil {
		return nil, err
	}

	for evt := range ch {
		events = append(events, evt)
	}

	return events, nil
}

// // Count sends a "COUNT" command to the relay and returns the count of events matching the filters.
// func (r *Relay) Count(
// 	ctx context.T,
// 	filters Filters,
// 	opts ...SubscriptionOption,
// ) (int64, []byte, error) {
// 	v, err := r.countInternal(ctx, filters, opts...)
// 	if err != nil {
// 		return 0, nil, err
// 	}
//
// 	return *v.Count, v.HyperLogLog, nil
// }
//
// func (r *Relay) countInternal(ctx context.T, filters Filters, opts ...SubscriptionOption) (CountEnvelope, error) {
// 	sub := r.PrepareSubscription(ctx, filters, opts...)
// 	sub.countResult = make(chan CountEnvelope)
//
// 	if err := sub.Fire(); err != nil {
// 		return CountEnvelope{}, err
// 	}
//
// 	defer sub.unsub(errors.New("countInternal() ended"))
//
// 	if _, ok := ctx.Deadline(); !ok {
// 		// if no timeout is set, force it to 7 seconds
// 		var cancel context.CancelFunc
// 		ctx, cancel = context.WithTimeoutCause(ctx, 7*time.Second, errors.New("countInternal took too long"))
// 		defer cancel()
// 	}
//
// 	for {
// 		select {
// 		case count := <-sub.countResult:
// 			return count, nil
// 		case <-ctx.Done():
// 			return CountEnvelope{}, ctx.Err()
// 		}
// 	}
// }

// Close closes the relay connection.
func (r *Client) Close() error {
	return r.close(errors.New("relay connection closed"))
}

func (r *Client) close(reason error) error {
	r.closeMutex.Lock()
	defer r.closeMutex.Unlock()

	if r.connectionContextCancel == nil {
		return fmt.Errorf("relay already closed")
	}
	r.connectionContextCancel(reason)
	r.connectionContextCancel = nil

	if r.Connection == nil {
		return fmt.Errorf("relay not connected")
	}

	err := r.Connection.Close()
	if err != nil {
		return err
	}

	return nil
}

var subIdPool = sync.Pool{
	New: func() any { return make([]byte, 0, 15) },
}
