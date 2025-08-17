package ws

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
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
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/subscription"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/interfaces/codec"
	"orly.dev/pkg/interfaces/signer"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/normalize"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v3"
)

var subscriptionIDCounter atomic.Int64

// Client represents a connection to a Nostr relay.
type Client struct {
	closeMutex sync.Mutex

	URL           string
	requestHeader http.Header // e.g. for origin header

	Connection    *Connection
	Subscriptions *xsync.MapOf[string, *Subscription]

	ConnectionError         error
	connectionContext       context.Context // will be canceled when the connection closes
	connectionContextCancel context.CancelCauseFunc

	challenge                     []byte       // NIP-42 challenge, we only keep the last
	notices                       chan []byte  // NIP-01 NOTICEs
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
func NewRelay(ctx context.Context, url string, opts ...RelayOption) *Client {
	ctx, cancel := context.WithCancelCause(ctx)
	r := &Client{
		URL:                     string(normalize.URL(url)),
		connectionContext:       ctx,
		connectionContextCancel: cancel,
		Subscriptions:           xsync.NewMapOf[string, *Subscription](),
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
func RelayConnect(ctx context.Context, url string, opts ...RelayOption) (
	*Client, error,
) {
	r := NewRelay(context.Background(), url, opts...)
	err := r.Connect(ctx)
	return r, err
}

// RelayOption is the type of the argument passed when instantiating relay connections.
type RelayOption interface {
	ApplyRelayOption(*Client)
}

var (
	_ RelayOption = (WithCustomHandler)(nil)
	_ RelayOption = (WithRequestHeader)(nil)
)

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
func (r *Client) Context() context.Context { return r.connectionContext }

// IsConnected returns true if the connection to this relay seems to be active.
func (r *Client) IsConnected() bool { return r.connectionContext.Err() == nil }

// Connect tries to establish a websocket connection to r.URL.
// If the context expires before the connection is complete, an error is returned.
// Once successfully connected, context expiration has no effect: call r.Close
// to close the connection.
//
// The given context here is only used during the connection phase. The long-living
// relay connection will be based on the context given to NewRelay().
func (r *Client) Connect(ctx context.Context) error {
	return r.ConnectWithTLS(ctx, nil)
}

func extractSubID(jsonStr string) string {
	// look for "EVENT" pattern
	start := strings.Index(jsonStr, `"EVENT"`)
	if start == -1 {
		return ""
	}

	// move to the next quote
	offset := strings.Index(jsonStr[start+7:], `"`)
	if offset == -1 {
		return ""
	}

	start += 7 + offset + 1

	// find the ending quote
	end := strings.Index(jsonStr[start:], `"`)

	// get the contents
	return jsonStr[start : start+end]
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
	ctx context.Context, tlsConfig *tls.Config,
) error {
	if r.connectionContext == nil || r.Subscriptions == nil {
		return fmt.Errorf("relay must be initialized with a call to NewRelay()")
	}

	if r.URL == "" {
		return fmt.Errorf("invalid relay URL '%s'", r.URL)
	}

	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeoutCause(
			ctx, 7*time.Second, errors.New("connection took too long"),
		)
		defer cancel()
	}

	conn, err := NewConnection(ctx, r.URL, r.requestHeader, tlsConfig)
	if err != nil {
		return fmt.Errorf("error opening websocket to '%s': %w", r.URL, err)
	}
	r.Connection = conn

	// ping every 29 seconds
	ticker := time.NewTicker(29 * time.Second)

	// queue all write operations here so we don't do mutex spaghetti
	go func() {
		var err error
		for {
			select {
			case <-r.connectionContext.Done():
				ticker.Stop()
				r.Connection = nil

				for _, sub := range r.Subscriptions.Range {
					sub.unsub(
						fmt.Errorf(
							"relay connection closed: %w / %w",
							context.Cause(r.connectionContext),
							r.ConnectionError,
						),
					)
				}
				return

			case <-ticker.C:
				err = r.Connection.Ping(r.connectionContext)
				if err != nil && !strings.Contains(
					err.Error(), "failed to wait for pong",
				) {
					log.I.F(
						"{%s} error writing ping: %v; closing websocket", r.URL,
						err,
					)
					r.Close() // this should trigger a context cancelation
					return
				}

			case wr := <-r.writeQueue:
				// all write requests will go through this to prevent races
				log.D.F("{%s} sending %v\n", r.URL, string(wr.msg))
				if err = r.Connection.WriteMessage(
					r.connectionContext, wr.msg,
				); err != nil {
					wr.answer <- err
				}
				close(wr.answer)
			}
		}
	}()

	// general message reader loop
	go func() {

		for {
			buf := new(bytes.Buffer)
			for {
				buf.Reset()
				if err := conn.ReadMessage(
					r.connectionContext, buf,
				); err != nil {
					r.ConnectionError = err
					r.Close()
					break
				}
				message := buf.Bytes()
				log.D.F("{%s} %v\n", r.URL, message)

				var t string
				if t, message, err = envelopes.Identify(message); chk.E(err) {
					continue
				}
				switch t {
				case noticeenvelope.L:
					env := noticeenvelope.New()
					if env, message, err = noticeenvelope.Parse(message); chk.E(err) {
						continue
					}
					// see WithNoticeHandler
					if r.notices != nil {
						r.notices <- env.Message
					} else {
						log.E.F("NOTICE from %s: '%s'\n", r.URL, env.Message)
					}
				case authenvelope.L:
					env := authenvelope.NewChallenge()
					if env, message, err = authenvelope.ParseChallenge(message); chk.E(err) {
						continue
					}
					if len(env.Challenge) == 0 {
						continue
					}
					r.challenge = env.Challenge
				case eventenvelope.L:
					env := eventenvelope.NewResult()
					if env, message, err = eventenvelope.ParseResult(message); chk.E(err) {
						continue
					}
					if len(env.Subscription.T) == 0 {
						continue
					}
					if sub, ok := r.Subscriptions.Load(env.Subscription.String()); !ok {
						log.D.F(
							"{%s} no subscription with id '%s'\n", r.URL,
							env.Subscription,
						)
						continue
					} else {
						// check if the event matches the desired filter, ignore otherwise
						if !sub.Filters.Match(env.Event) {
							log.D.F(
								"{%s} filter does not match: %v ~ %v\n", r.URL,
								sub.Filters, env.Event,
							)
							continue
						}
						// check signature, ignore invalid, except from trusted (AssumeValid) relays
						if !r.AssumeValid {
							if ok, err = env.Event.Verify(); !ok {
								log.E.F(
									"{%s} bad signature on %s\n", r.URL,
									env.Event.IdString(),
								)
								continue
							}
						}
						// dispatch this to the internal .events channel of the subscription
						sub.dispatchEvent(env.Event)
					}
				case eoseenvelope.L:
					env := eoseenvelope.New()
					if env, message, err = eoseenvelope.Parse(message); chk.E(err) {
						continue
					}
					if subscription, ok := r.Subscriptions.Load(env.Subscription.String()); ok {
						subscription.dispatchEose()
					}
				case closedenvelope.L:
					env := closedenvelope.New()
					if env, message, err = closedenvelope.Parse(message); chk.E(err) {
						continue
					}
					if subscription, ok := r.Subscriptions.Load(env.Subscription.String()); ok {
						subscription.handleClosed(env.ReasonString())
					}
				case okenvelope.L:
					env := okenvelope.New()
					if env, message, err = okenvelope.Parse(message); chk.E(err) {
						continue
					}
					if okCallback, exist := r.okCallbacks.Load(env.EventID.String()); exist {
						okCallback(env.OK, env.ReasonString())
					} else {
						log.I.F(
							"{%s} got an unexpected OK message for event %s",
							r.URL,
							env.EventID,
						)
					}
				}
			}
		}
	}()

	return nil
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

// Publish sends an "EVENT" command to the relay r as in NIP-01 and waits for an OK response.
func (r *Client) Publish(ctx context.Context, event *event.E) error {
	return r.publish(
		ctx, event.IdString(), eventenvelope.NewSubmissionWith(event),
	)
}

// Auth sends an "AUTH" command client->relay as in NIP-42 and waits for an OK response.
//
// You don't have to build the AUTH event yourself, this function takes a function to which the
// event that must be signed will be passed, so it's only necessary to sign that.
func (r *Client) Auth(
	ctx context.Context, sign signer.I,
) (err error) {
	authEvent := &event.E{
		CreatedAt: timestamp.Now(),
		Kind:      kind.ClientAuthentication,
		Tags: tags.New(
			tag.New("relay", r.URL),
			tag.New("challenge", string(r.challenge)),
		),
		Content: nil,
	}
	if err = authEvent.Sign(sign); err != nil {
		return fmt.Errorf("error signing auth event: %w", err)
	}

	return r.publish(
		ctx, authEvent.IdString(), authenvelope.NewResponseWith(authEvent),
	)
}

func (r *Client) publish(
	ctx context.Context, id string, env codec.Envelope,
) error {
	var err error
	var cancel context.CancelFunc

	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		ctx, cancel = context.WithTimeoutCause(
			ctx, 7*time.Second, fmt.Errorf("given up waiting for an OK"),
		)
		defer cancel()
	} else {
		// otherwise make the context cancellable so we can stop everything upon receiving an "OK"
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
	}

	// listen for an OK callback
	gotOk := false
	r.okCallbacks.Store(
		id, func(ok bool, reason string) {
			gotOk = true
			if !ok {
				err = fmt.Errorf("msg: %s", reason)
			}
			cancel()
		},
	)
	defer r.okCallbacks.Delete(id)

	// publish event
	envb := env.Marshal(nil)
	if err = <-r.Write(envb); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			// this will be called when we get an OK or when the context has been canceled
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

// Subscribe sends a "REQ" command to the relay r as in NIP-01.
// Events are returned through the channel sub.Events.
// The subscription is closed when context ctx is cancelled ("CLOSE" in NIP-01).
//
// Remember to cancel subscriptions, either by calling `.Unsub()` on them or ensuring their `context.Context` will be canceled at some point.
// Failure to do that will result in a huge number of halted goroutines being created.
func (r *Client) Subscribe(
	ctx context.Context, ff *filters.T, opts ...SubscriptionOption,
) (*Subscription, error) {
	sub := r.PrepareSubscription(ctx, ff, opts...)

	if r.Connection == nil {
		return nil, fmt.Errorf("not connected to %s", r.URL)
	}

	if err := sub.Fire(); err != nil {
		return nil, fmt.Errorf(
			"couldn't subscribe to %v at %s: %w", ff, r.URL, err,
		)
	}

	return sub, nil
}

// PrepareSubscription creates a subscription, but doesn't fire it.
//
// Remember to cancel subscriptions, either by calling `.Unsub()` on them or ensuring their `context.Context` will be canceled at some point.
// Failure to do that will result in a huge number of halted goroutines being created.
func (r *Client) PrepareSubscription(
	ctx context.Context, ff *filters.T, opts ...SubscriptionOption,
) (sub *Subscription) {
	current := subscriptionIDCounter.Add(1)
	ctx, cancel := context.WithCancelCause(ctx)
	sub = &Subscription{
		Client:            r,
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
		case WithCheckDuplicate:
			sub.checkDuplicate = o
		case WithCheckDuplicateReplaceable:
			sub.checkDuplicateReplaceable = o
		}
	}
	// subscription id computation
	buf := subIdPool.Get().([]byte)[:0]
	buf = strconv.AppendInt(buf, sub.counter, 10)
	buf = append(buf, ':')
	buf = append(buf, label...)
	defer subIdPool.Put(buf)
	sub.id = &subscription.Id{T: buf}
	r.Subscriptions.Store(string(buf), sub)
	// start handling events, eose, unsub etc:
	go sub.start()
	return sub
}

// QueryEvents subscribes to events matching the given filter and returns a channel of events.
//
// In most cases it's better to use SimplePool instead of this method.
func (r *Client) QueryEvents(ctx context.Context, f *filter.F) (
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
	evc = sub.Events
	return
}

// QuerySync subscribes to events matching the given filter and returns a slice of events.
// This method blocks until all events are received or the context is canceled.
//
// In most cases it's better to use SimplePool instead of this method.
func (r *Client) QuerySync(ctx context.Context, ff *filter.F) (
	[]*event.E, error,
) {
	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeoutCause(
			ctx, 7*time.Second, errors.New("QuerySync() took too long"),
		)
		defer cancel()
	}
	var lim int
	if ff.Limit != nil {
		lim = int(*ff.Limit)
	}
	events := make([]*event.E, 0, max(lim, 250))
	ch, err := r.QueryEvents(ctx, ff)
	if err != nil {
		return nil, err
	}

	for evt := range ch {
		events = append(events, evt)
	}

	return events, nil
}

// Close closes the relay connection.
func (r *Client) Close() error {
	return r.close(errors.New("Close() called"))
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
