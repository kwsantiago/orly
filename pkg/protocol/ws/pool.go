package ws

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/puzpuzpuz/xsync/v3"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/interfaces/signer"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/normalize"
)

const (
	seenAlreadyDropTick = time.Minute
)

// Pool manages connections to multiple relays, ensures they are reopened when necessary and not duplicated.
type Pool struct {
	Relays  *xsync.MapOf[string, *Client]
	Context context.T

	authHandler func() signer.I
	cancel      context.C

	eventMiddleware     func(RelayEvent)
	duplicateMiddleware func(relay, id string)
	queryMiddleware     func(relay, pubkey string, kind uint16)

	// custom things not often used
	penaltyBoxMu sync.Mutex
	penaltyBox   map[string][2]float64
	relayOptions []RelayOption
}

// DirectedFilter combines a Filter with a specific relay URL.
type DirectedFilter struct {
	*filter.F
	Relay string
}

// RelayEvent represents an event received from a specific relay.
type RelayEvent struct {
	*event.E
	Relay *Client
}

func (ie RelayEvent) String() string {
	return fmt.Sprintf(
		"[%s] >> %s", ie.Relay.URL, ie.E.Marshal(nil),
	)
}

// PoolOption is an interface for options that can be applied to a Pool.
type PoolOption interface {
	ApplyPoolOption(*Pool)
}

// NewPool creates a new Pool with the given context and options.
func NewPool(c context.T, opts ...PoolOption) (pool *Pool) {
	ctx, cancel := context.Cause(c)
	pool = &Pool{
		Relays:  xsync.NewMapOf[string, *Client](),
		Context: ctx,
		cancel:  cancel,
	}
	for _, opt := range opts {
		opt.ApplyPoolOption(pool)
	}
	return pool
}

// WithRelayOptions sets options that will be used on every relay instance created by this pool.
func WithRelayOptions(ropts ...RelayOption) withRelayOptionsOpt {
	return ropts
}

type withRelayOptionsOpt []RelayOption

func (h withRelayOptionsOpt) ApplyPoolOption(pool *Pool) {
	pool.relayOptions = h
}

// WithAuthHandler must be a function that signs the auth event when called.
// it will be called whenever any relay in the pool returns a `CLOSED` message
// with the "auth-required:" prefix, only once for each relay
type WithAuthHandler func() signer.I

func (h WithAuthHandler) ApplyPoolOption(pool *Pool) {
	pool.authHandler = h
}

// WithPenaltyBox just sets the penalty box mechanism so relays that fail to connect
// or that disconnect will be ignored for a while and we won't attempt to connect again.
func WithPenaltyBox() withPenaltyBoxOpt { return withPenaltyBoxOpt{} }

type withPenaltyBoxOpt struct{}

func (h withPenaltyBoxOpt) ApplyPoolOption(pool *Pool) {
	pool.penaltyBox = make(map[string][2]float64)
	go func() {
		sleep := 30.0
		for {
			time.Sleep(time.Duration(sleep) * time.Second)

			pool.penaltyBoxMu.Lock()
			nextSleep := 300.0
			for url, v := range pool.penaltyBox {
				remainingSeconds := v[1]
				remainingSeconds -= sleep
				if remainingSeconds <= 0 {
					pool.penaltyBox[url] = [2]float64{v[0], 0}
					continue
				} else {
					pool.penaltyBox[url] = [2]float64{v[0], remainingSeconds}
				}

				if remainingSeconds < nextSleep {
					nextSleep = remainingSeconds
				}
			}

			sleep = nextSleep
			pool.penaltyBoxMu.Unlock()
		}
	}()
}

// WithEventMiddleware is a function that will be called with all events received.
type WithEventMiddleware func(RelayEvent)

func (h WithEventMiddleware) ApplyPoolOption(pool *Pool) {
	pool.eventMiddleware = h
}

// WithDuplicateMiddleware is a function that will be called with all duplicate ids received.
type WithDuplicateMiddleware func(relay string, id string)

func (h WithDuplicateMiddleware) ApplyPoolOption(pool *Pool) {
	pool.duplicateMiddleware = h
}

// WithAuthorKindQueryMiddleware is a function that will be called with every combination of relay+pubkey+kind queried
// in a .SubMany*() call -- when applicable (i.e. when the query contains a pubkey and a kind).
type WithAuthorKindQueryMiddleware func(relay, pubkey string, kind uint16)

func (h WithAuthorKindQueryMiddleware) ApplyPoolOption(pool *Pool) {
	pool.queryMiddleware = h
}

var (
	_ PoolOption = (WithAuthHandler)(nil)
	_ PoolOption = (WithEventMiddleware)(nil)
	_ PoolOption = WithPenaltyBox()
	_ PoolOption = WithRelayOptions(WithRequestHeader(http.Header{}))
)

const MAX_LOCKS = 50

var namedMutexPool = make([]sync.Mutex, MAX_LOCKS)

//go:noescape
//go:linkname memhash runtime.memhash
func memhash(p unsafe.Pointer, h, s uintptr) uintptr

func namedLock[V ~[]byte | ~string](name V) (unlock func()) {
	sptr := unsafe.StringData(string(name))
	idx := uint64(
		memhash(
			unsafe.Pointer(sptr), 0, uintptr(len(name)),
		),
	) % MAX_LOCKS
	namedMutexPool[idx].Lock()
	return namedMutexPool[idx].Unlock
}

// EnsureRelay ensures that a relay connection exists and is active.
// If the relay is not connected, it attempts to connect.
func (p *Pool) EnsureRelay(url string) (*Client, error) {
	nm := string(normalize.URL(url))
	defer namedLock(nm)()
	relay, ok := p.Relays.Load(nm)
	if ok && relay == nil {
		if p.penaltyBox != nil {
			p.penaltyBoxMu.Lock()
			defer p.penaltyBoxMu.Unlock()
			v, _ := p.penaltyBox[nm]
			if v[1] > 0 {
				return nil, fmt.Errorf("in penalty box, %fs remaining", v[1])
			}
		}
	} else if ok && relay.IsConnected() {
		// already connected, unlock and return
		return relay, nil
	}
	// try to connect
	// we use this ctx here so when the p dies everything dies
	ctx, cancel := context.TimeoutCause(
		p.Context,
		time.Second*15,
		errors.New("connecting to the relay took too long"),
	)
	defer cancel()
	relay = NewRelay(context.Bg(), url, p.relayOptions...)
	if err := relay.Connect(ctx); err != nil {
		if p.penaltyBox != nil {
			// putting relay in penalty box
			p.penaltyBoxMu.Lock()
			defer p.penaltyBoxMu.Unlock()
			v, _ := p.penaltyBox[nm]
			p.penaltyBox[nm] = [2]float64{
				v[0] + 1, 30.0 + math.Pow(2, v[0]+1),
			}
		}
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	p.Relays.Store(nm, relay)
	return relay, nil
}

// PublishResult represents the result of publishing an event to a relay.
type PublishResult struct {
	Error    error
	RelayURL string
	Relay    *Client
}

// todo: this didn't used to be in this package... probably don't want to add it
//  either.
//
// PublishMany publishes an event to multiple relays and returns a
// channel of results emitted as they're received.

// func (pool *Pool) PublishMany(
// 	ctx context.T, urls []string, evt *event.E,
// ) chan PublishResult {
// 	ch := make(chan PublishResult, len(urls))
// 	wg := sync.WaitGroup{}
// 	wg.Add(len(urls))
// 	go func() {
// 		for _, url := range urls {
// 			go func() {
// 				defer wg.Done()
// 				relay, err := pool.EnsureRelay(url)
// 				if err != nil {
// 					ch <- PublishResult{err, url, nil}
// 					return
// 				}
// 				if err = relay.Publish(ctx, evt); err == nil {
// 					// success with no auth required
// 					ch <- PublishResult{nil, url, relay}
// 				} else if strings.HasPrefix(
// 					err.Error(), "msg: auth-required:",
// 				) && pool.authHandler != nil {
// 					// try to authenticate if we can
// 					if authErr := relay.Auth(
// 						ctx, pool.authHandler(),
// 					); authErr == nil {
// 						if err := relay.Publish(ctx, evt); err == nil {
// 							// success after auth
// 							ch <- PublishResult{nil, url, relay}
// 						} else {
// 							// failure after auth
// 							ch <- PublishResult{err, url, relay}
// 						}
// 					} else {
// 						// failure to auth
// 						ch <- PublishResult{
// 							fmt.Errorf(
// 								"failed to auth: %w", authErr,
// 							), url, relay,
// 						}
// 					}
// 				} else {
// 					// direct failure
// 					ch <- PublishResult{err, url, relay}
// 				}
// 			}()
// 		}
//
// 		wg.Wait()
// 		close(ch)
// 	}()
//
// 	return ch
// }

// SubscribeMany opens a subscription with the given filter to multiple relays
// the subscriptions ends when the context is canceled or when all relays return a CLOSED.
func (p *Pool) SubscribeMany(
	ctx context.T,
	urls []string,
	filter *filter.F,
	opts ...SubscriptionOption,
) chan RelayEvent {
	return p.subMany(ctx, urls, filters.New(filter), nil, opts...)
}

// FetchMany opens a subscription, much like SubscribeMany, but it ends as soon as all Relays
// return an EOSE message.
func (p *Pool) FetchMany(
	ctx context.T,
	urls []string,
	filter *filter.F,
	opts ...SubscriptionOption,
) chan RelayEvent {
	return p.SubManyEose(ctx, urls, filters.New(filter), opts...)
}

// Deprecated: SubMany is deprecated: use SubscribeMany instead.
func (p *Pool) SubMany(
	ctx context.T,
	urls []string,
	filters *filters.T,
	opts ...SubscriptionOption,
) chan RelayEvent {
	return p.subMany(ctx, urls, filters, nil, opts...)
}

// SubscribeManyNotifyEOSE is like SubscribeMany, but takes a channel that is closed when
// all subscriptions have received an EOSE
func (p *Pool) SubscribeManyNotifyEOSE(
	ctx context.T,
	urls []string,
	filter *filter.F,
	eoseChan chan struct{},
	opts ...SubscriptionOption,
) chan RelayEvent {
	return p.subMany(ctx, urls, filters.New(filter), eoseChan, opts...)
}

type ReplaceableKey struct {
	PubKey string
	D      string
}

// FetchManyReplaceable is like FetchMany, but deduplicates replaceable and addressable events and returns
// only the latest for each "d" tag.
func (p *Pool) FetchManyReplaceable(
	ctx context.T,
	urls []string,
	f *filter.F,
	opts ...SubscriptionOption,
) *xsync.MapOf[ReplaceableKey, *event.E] {
	ctx, cancel := context.Cause(ctx)
	results := xsync.NewMapOf[ReplaceableKey, *event.E]()
	wg := sync.WaitGroup{}
	wg.Add(len(urls))
	// todo: this is a hack for compensating for retarded relays that don't
	// 	filter replaceable events because it streams them back over a channel.
	// 	this is out of spec anyway so should not be handled. replaceable events
	// 	are supposed to delete old versions. the end. this is for the incorrect
	// 	behaviour of fiatjaf's database code, which he obviously thinks is clever
	// 	for using channels, and not sorting results before dispatching them
	// 	before EOSE.
	_ = 0
	// seenAlreadyLatest := xsync.NewMapOf[ReplaceableKey,
	// *timestamp.T]() opts = append(
	// 	opts, WithCheckDuplicateReplaceable(
	// 		func(rk ReplaceableKey, ts Timestamp) bool {
	// 			updated := false
	// 			seenAlreadyLatest.Compute(
	// 				rk, func(latest Timestamp, _ bool) (
	// 					newValue Timestamp, delete bool,
	// 				) {
	// 					if ts > latest {
	// 						updated = true // we are updating the most recent
	// 						return ts, false
	// 					}
	// 					return latest, false // the one we had was already more recent
	// 				},
	// 			)
	// 			return updated
	// 		},
	// 	),
	// )

	for _, url := range urls {
		go func(nm string) {
			defer wg.Done()

			if mh := p.queryMiddleware; mh != nil {
				if f.Kinds != nil && f.Authors != nil {
					for _, kind := range f.Kinds.K {
						for _, author := range f.Authors.ToStringSlice() {
							mh(nm, author, kind.K)
						}
					}
				}
			}

			relay, err := p.EnsureRelay(nm)
			if err != nil {
				log.D.F("error connecting to %s with %v: %s", nm, f, err)
				return
			}

			hasAuthed := false

		subscribe:
			sub, err := relay.Subscribe(ctx, filters.New(f), opts...)
			if err != nil {
				log.D.F(
					"error subscribing to %s with %v: %s", relay, f, err,
				)
				return
			}

			for {
				select {
				case <-ctx.Done():
					return
				case <-sub.EndOfStoredEvents:
					return
				case reason := <-sub.ClosedReason:
					if strings.HasPrefix(
						reason, "auth-required:",
					) && p.authHandler != nil && !hasAuthed {
						// relay is requesting auth. if we can we will perform auth and try again
						err = relay.Auth(
							ctx, p.authHandler(),
						)
						if err == nil {
							hasAuthed = true // so we don't keep doing AUTH again and again
							goto subscribe
						}
					}
					log.D.F("CLOSED from %s: '%s'\n", nm, reason)
					return
				case evt, more := <-sub.Events:
					if !more {
						return
					}

					ie := RelayEvent{E: evt, Relay: relay}
					if mh := p.eventMiddleware; mh != nil {
						mh(ie)
					}

					results.Store(
						ReplaceableKey{hex.Enc(evt.Pubkey), evt.Tags.GetD()},
						evt,
					)
				}
			}
		}(string(normalize.URL(url)))
	}

	// this will happen when all subscriptions get an eose (or when they die)
	wg.Wait()
	cancel(errors.New("all subscriptions ended"))

	return results
}

func (p *Pool) subMany(
	ctx context.T,
	urls []string,
	ff *filters.T,
	eoseChan chan struct{},
	opts ...SubscriptionOption,
) chan RelayEvent {
	ctx, cancel := context.Cause(ctx)
	_ = cancel // do this so `go vet` will stop complaining
	events := make(chan RelayEvent)
	seenAlready := xsync.NewMapOf[string, *timestamp.T]()
	ticker := time.NewTicker(seenAlreadyDropTick)

	eoseWg := sync.WaitGroup{}
	eoseWg.Add(len(urls))
	if eoseChan != nil {
		go func() {
			eoseWg.Wait()
			close(eoseChan)
		}()
	}

	pending := xsync.NewCounter()
	pending.Add(int64(len(urls)))
	for i, url := range urls {
		url = string(normalize.URL(url))
		urls[i] = url
		if idx := slices.Index(urls, url); idx != i {
			// skip duplicate relays in the list
			eoseWg.Done()
			continue
		}

		eosed := atomic.Bool{}
		firstConnection := true

		go func(nm string) {
			defer func() {
				pending.Dec()
				if pending.Value() == 0 {
					close(events)
					cancel(fmt.Errorf("aborted: %w", context.GetCause(ctx)))
				}
				if eosed.CompareAndSwap(false, true) {
					eoseWg.Done()
				}
			}()

			hasAuthed := false
			interval := 3 * time.Second
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				var sub *Subscription

				if mh := p.queryMiddleware; mh != nil {
					for _, f := range ff.F {
						if f.Kinds != nil && f.Authors != nil {
							for _, k := range f.Kinds.K {
								for _, author := range f.Authors.ToSliceOfBytes() {
									mh(nm, hex.Enc(author), k.K)
								}
							}
						}
					}
				}

				relay, err := p.EnsureRelay(nm)
				if err != nil {
					// if we never connected to this just fail
					if firstConnection {
						return
					}

					// otherwise (if we were connected and got disconnected) keep trying to reconnect
					log.D.F("%s reconnecting because connection failed\n", nm)
					goto reconnect
				}
				firstConnection = false
				hasAuthed = false

			subscribe:
				sub, err = relay.Subscribe(
					ctx, ff,
					// append(
					opts...,
				// WithCheckDuplicate(
				// 	func(id, relay string) bool {
				// 		_, exists := seenAlready.Load(id)
				// 		if exists && p.duplicateMiddleware != nil {
				// 			p.duplicateMiddleware(relay, id)
				// 		}
				// 		return exists
				// 	},
				// ),
				// )...,
				)
				if err != nil {
					log.D.F("%s reconnecting because subscription died\n", nm)
					goto reconnect
				}

				go func() {
					<-sub.EndOfStoredEvents

					// guard here otherwise a resubscription will trigger a duplicate call to eoseWg.Done()
					if eosed.CompareAndSwap(false, true) {
						eoseWg.Done()
					}
				}()

				// reset interval when we get a good subscription
				interval = 3 * time.Second

				for {
					select {
					case evt, more := <-sub.Events:
						if !more {
							// this means the connection was closed for weird reasons, like the server shut down
							// so we will update the filters here to include only events seem from now on
							// and try to reconnect until we succeed
							now := timestamp.Now()
							for i := range ff.F {
								ff.F[i].Since = now
							}
							log.D.F(
								"%s reconnecting because sub.Events is broken\n",
								nm,
							)
							goto reconnect
						}

						ie := RelayEvent{E: evt, Relay: relay}
						if mh := p.eventMiddleware; mh != nil {
							mh(ie)
						}

						select {
						case events <- ie:
						case <-ctx.Done():
							return
						}
					case <-ticker.C:
						if eosed.Load() {
							old := timestamp.New(time.Now().Add(-seenAlreadyDropTick).Unix())
							for id, value := range seenAlready.Range {
								if value.I64() < old.I64() {
									seenAlready.Delete(id)
								}
							}
						}
					case reason := <-sub.ClosedReason:
						if strings.HasPrefix(
							reason, "auth-required:",
						) && p.authHandler != nil && !hasAuthed {
							// relay is requesting auth. if we can we will perform auth and try again
							err = relay.Auth(
								ctx, p.authHandler(),
							)
							if err == nil {
								hasAuthed = true // so we don't keep doing AUTH again and again
								goto subscribe
							}
						} else {
							log.D.F("CLOSED from %s: '%s'\n", nm, reason)
						}

						return
					case <-ctx.Done():
						return
					}
				}

			reconnect:
				// we will go back to the beginning of the loop and try to connect again and again
				// until the context is canceled
				time.Sleep(interval)
				interval = interval * 17 / 10 // the next time we try we will wait longer
			}
		}(url)
	}

	return events
}

// Deprecated: SubManyEose is deprecated: use FetchMany instead.
func (p *Pool) SubManyEose(
	ctx context.T,
	urls []string,
	filters *filters.T,
	opts ...SubscriptionOption,
) chan RelayEvent {
	// seenAlready := xsync.NewMapOf[string, struct{}]()
	return p.subManyEoseNonOverwriteCheckDuplicate(
		ctx, urls, filters,
		// WithCheckDuplicate(
		// 	func(id, relay string) bool {
		// 		_, exists := seenAlready.LoadOrStore(id, struct{}{})
		// 		if exists && p.duplicateMiddleware != nil {
		// 			p.duplicateMiddleware(relay, id)
		// 		}
		// 		return exists
		// 	},
		// ),
		opts...,
	)
}

func (p *Pool) subManyEoseNonOverwriteCheckDuplicate(
	ctx context.T,
	urls []string,
	filters *filters.T,
	// wcd WithCheckDuplicate,
	opts ...SubscriptionOption,
) chan RelayEvent {
	ctx, cancel := context.Cause(ctx)

	events := make(chan RelayEvent)
	wg := sync.WaitGroup{}
	wg.Add(len(urls))

	// opts = append(opts, wcd)

	go func() {
		// this will happen when all subscriptions get an eose (or when they die)
		wg.Wait()
		cancel(errors.New("all subscriptions ended"))
		close(events)
	}()

	for _, url := range urls {
		go func(nm string) {
			defer wg.Done()

			if mh := p.queryMiddleware; mh != nil {
				for _, filter := range filters.F {
					if filter.Kinds != nil && filter.Authors != nil {
						for _, k := range filter.Kinds.K {
							for _, author := range filter.Authors.ToSliceOfBytes() {
								mh(nm, hex.Enc(author), k.K)
							}
						}
					}
				}
			}

			relay, err := p.EnsureRelay(nm)
			if err != nil {
				log.D.F(
					"error connecting to %s with %v: %s", nm, filters, err,
				)
				return
			}

			hasAuthed := false

		subscribe:
			sub, err := relay.Subscribe(ctx, filters, opts...)
			if err != nil {
				log.D.F(
					"error subscribing to %s with %v: %s", relay, filters, err,
				)
				return
			}

			for {
				select {
				case <-ctx.Done():
					return
				case <-sub.EndOfStoredEvents:
					return
				case reason := <-sub.ClosedReason:
					if strings.HasPrefix(
						reason, "auth-required:",
					) && p.authHandler != nil && !hasAuthed {
						// relay is requesting auth. if we can we will perform auth and try again
						err = relay.Auth(
							ctx, p.authHandler(),
						)
						if err == nil {
							hasAuthed = true // so we don't keep doing AUTH again and again
							goto subscribe
						}
					}
					log.D.F("CLOSED from %s: '%s'\n", nm, reason)
					return
				case evt, more := <-sub.Events:
					if !more {
						return
					}

					ie := RelayEvent{E: evt, Relay: relay}
					if mh := p.eventMiddleware; mh != nil {
						mh(ie)
					}

					select {
					case events <- ie:
					case <-ctx.Done():
						return
					}
				}
			}
		}(string(normalize.URL(url)))
	}

	return events
}

// // CountMany aggregates count results from multiple relays using NIP-45 HyperLogLog
// func (pool *Pool) CountMany(
// 	ctx context.T,
// 	urls []string,
// 	filter *filter.F,
// 	opts []SubscriptionOption,
// ) int {
// 	hll := hyperloglog.New(0) // offset is irrelevant here
//
// 	wg := sync.WaitGroup{}
// 	wg.Add(len(urls))
// 	for _, url := range urls {
// 		go func(nm string) {
// 			defer wg.Done()
// 			relay, err := pool.EnsureRelay(url)
// 			if err != nil {
// 				return
// 			}
// 			ce, err := relay.countInternal(ctx, Filters{filter}, opts...)
// 			if err != nil {
// 				return
// 			}
// 			if len(ce.HyperLogLog) != 256 {
// 				return
// 			}
// 			hll.MergeRegisters(ce.HyperLogLog)
// 		}(NormalizeURL(url))
// 	}
//
// 	wg.Wait()
// 	return int(hll.Count())
// }

// QuerySingle returns the first event returned by the first relay, cancels everything else.
func (p *Pool) QuerySingle(
	ctx context.T,
	urls []string,
	filter *filter.F,
	opts ...SubscriptionOption,
) *RelayEvent {
	ctx, cancel := context.Cause(ctx)
	for ievt := range p.SubManyEose(
		ctx, urls, filters.New(filter), opts...,
	) {
		cancel(errors.New("got the first event and ended successfully"))
		return &ievt
	}
	cancel(errors.New("SubManyEose() didn't get yield events"))
	return nil
}

// BatchedSubManyEose performs batched subscriptions to multiple relays with different filters.
func (p *Pool) BatchedSubManyEose(
	ctx context.T,
	dfs []DirectedFilter,
	opts ...SubscriptionOption,
) chan RelayEvent {
	res := make(chan RelayEvent)
	wg := sync.WaitGroup{}
	wg.Add(len(dfs))
	// seenAlready := xsync.NewMapOf[string, struct{}]()
	for _, df := range dfs {
		go func(df DirectedFilter) {
			for ie := range p.subManyEoseNonOverwriteCheckDuplicate(
				ctx,
				[]string{df.Relay},
				filters.New(df.F),
				// WithCheckDuplicate(
				// 	func(id, relay string) bool {
				// 		_, exists := seenAlready.LoadOrStore(id, struct{}{})
				// 		if exists && p.duplicateMiddleware != nil {
				// 			p.duplicateMiddleware(relay, id)
				// 		}
				// 		return exists
				// 	},
				// ),
				opts...,
			) {
				select {
				case res <- ie:
				case <-ctx.Done():
					wg.Done()
					return
				}
			}
			wg.Done()
		}(df)
	}

	go func() {
		wg.Wait()
		close(res)
	}()

	return res
}

// Close closes the pool with the given reason.
func (p *Pool) Close(reason string) {
	p.cancel(fmt.Errorf("pool closed with reason: '%s'", reason))
}
