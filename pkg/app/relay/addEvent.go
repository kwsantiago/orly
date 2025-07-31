package relay

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"orly.dev/pkg/crypto/ec/secp256k1"
	"orly.dev/pkg/protocol/httpauth"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
	realy_lol "orly.dev/pkg/version"
	"regexp"
	"strings"

	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/interfaces/relay"
	"orly.dev/pkg/interfaces/store"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/normalize"
)

var (
	NIP20prefixmatcher = regexp.MustCompile(`^\w+: `)
)

var userAgent = fmt.Sprintf("orly/%s", realy_lol.V)

type WriteCloser struct {
	*bytes.Buffer
}

func (w *WriteCloser) Close() error {
	w.Buffer.Reset()
	return nil
}

func NewWriteCloser(w []byte) *WriteCloser {
	return &WriteCloser{bytes.NewBuffer(w)}
}

// AddEvent processes an incoming event, saves it if valid, and delivers it to
// subscribers.
//
// # Parameters
//
//   - c: context for request handling
//
//   - rl: relay interface
//
//   - ev: the event to be added
//
//   - hr: HTTP request related to the event (if any)
//
//   - origin: origin of the event (if any)
//
//   - authedPubkey: public key of the authenticated user (if any)
//
// # Return Values
//
//   - accepted: true if the event was successfully processed, false otherwise
//
//   - message: additional information or error message related to the
//     processing
//
// # Expected Behaviour:
//
// - Validates the incoming event.
//
// - Saves the event using the Publish method if it is not ephemeral.
//
// - Handles duplicate events by returning an appropriate error message.
//
// - Delivers the event to subscribers via the listeners' Deliver method.
//
// - Returns a boolean indicating whether the event was accepted and any
// relevant message.
func (s *Server) AddEvent(
	c context.T, rl relay.I, ev *event.E, hr *http.Request, origin string,
	pubkey []byte,
) (accepted bool, message []byte) {

	if ev == nil {
		return false, normalize.Invalid.F("empty event")
	}
	if ev.Kind.IsEphemeral() {
	} else {
		if saveErr := s.Publish(c, ev); saveErr != nil {
			if errors.Is(saveErr, store.ErrDupEvent) {
				return false, []byte(saveErr.Error())
			}
			errmsg := saveErr.Error()
			if NIP20prefixmatcher.MatchString(errmsg) {
				if strings.Contains(errmsg, "tombstone") {
					return false, normalize.Error.F(
						"%s event was deleted, not storing it again",
						origin,
					)
				}
				if strings.HasPrefix(errmsg, string(normalize.Blocked)) {
					return false, []byte(errmsg)
				}
				return false, []byte(errmsg)
			} else {
				return false, []byte(errmsg)
			}
		}
	}
	// notify subscribers
	s.listeners.Deliver(ev)
	// push the new event to replicas if replicas are configured, and the relay
	// has an identity key.
	//
	// TODO: add the chain of pubkeys of replicas that send and were received from replicas sending so they can
	//  be skipped for large (5+) clusters.
	var err error
	if len(s.Peers.Addresses) > 0 &&
		len(s.Peers.I.Sec()) == secp256k1.SecKeyBytesLen {
		evb := ev.Marshal(nil)
		var payload io.ReadCloser
		payload = NewWriteCloser(evb)
		for i, a := range s.Peers.Addresses {
			// the peer address index is the same as the list of pubkeys
			// (they're unpacked from a string containing both, appended at the
			// same time), so if the pubkey the http event endpoint sent us here
			// matches the index of this address, we can skip it.
			if bytes.Equal(s.Peers.Pubkeys[i], pubkey) {
				log.I.F(
					"not sending back to replica that just sent us this event %0x",
					ev.ID,
				)
				continue
			}
			var ur *url.URL
			if ur, err = url.Parse(a + "/api/event"); chk.E(err) {
				continue
			}
			var r *http.Request
			r = &http.Request{
				Method:        "POST",
				URL:           ur,
				Proto:         "HTTP/1.1",
				ProtoMajor:    1,
				ProtoMinor:    1,
				Header:        make(http.Header),
				Body:          payload,
				ContentLength: int64(len(evb)),
				Host:          ur.Host,
			}
			r.Header.Add("User-Agent", userAgent)
			if err = httpauth.AddNIP98Header(
				r, ur, "POST", "", s.Peers.I, 0,
			); chk.E(err) {
				continue
			}
			r.GetBody = func() (rc io.ReadCloser, err error) {
				rc = payload
				return
			}
			client := &http.Client{}
			if _, err = client.Do(r); chk.E(err) {
				continue
			}
			log.I.F("event pushed to replica %s", ur.String())
		}
	}
	accepted = true
	return
}
