package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/url"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/errorf"
	"strings"
	"time"
)

// GenerateChallenge creates a reasonable, 16-byte base64 challenge string
func GenerateChallenge() (b []byte) {
	bb := make([]byte, 12)
	b = make([]byte, 16)
	_, _ = rand.Read(bb)
	base64.URLEncoding.Encode(b, bb)
	return
}

// CreateUnsigned creates an event which should be sent via an "AUTH" command.
// If the authentication succeeds, the user will be authenticated as a pubkey.
func CreateUnsigned(pubkey, challenge []byte, relayURL string) (ev *event.E) {
	return &event.E{
		Pubkey:    pubkey,
		CreatedAt: timestamp.Now(),
		Kind:      kind.ClientAuthentication,
		Tags: tags.New(
			tag.New("relay", relayURL),
			tag.New("challenge", string(challenge)),
		),
	}
}

// helper function for ValidateAuthEvent.
func parseURL(input string) (*url.URL, error) {
	return url.Parse(
		strings.ToLower(
			strings.TrimSuffix(input, "/"),
		),
	)
}

var (
	// ChallengeTag is the tag for the challenge in a NIP-42 auth event
	// (prevents relay attacks).
	ChallengeTag = []byte("challenge")
	// RelayTag is the relay tag for a NIP-42 auth event (prevents cross-server
	// attacks).
	RelayTag = []byte("relay")
)

// Validate checks whether an event is a valid NIP-42 event for a given
// challenge and relayURL. The result of the validation is encoded in the ok
// bool.
func Validate(evt *event.E, challenge []byte, relayURL string) (
	ok bool, err error,
) {
	if evt.Kind.K != kind.ClientAuthentication.K {
		err = errorf.E(
			"event incorrect kind for auth: %d %s",
			evt.Kind.K, kind.GetString(evt.Kind),
		)
		return
	}
	if evt.Tags.GetFirst(tag.New(ChallengeTag, challenge)) == nil {
		err = errorf.E("challenge tag missing from auth response")
		return
	}
	var expected, found *url.URL
	if expected, err = parseURL(relayURL); chk.D(err) {
		return
	}
	r := evt.Tags.
		GetFirst(tag.New(RelayTag, nil)).Value()
	if len(r) == 0 {
		err = errorf.E("relay tag missing from auth response")
		return
	}
	if found, err = parseURL(string(r)); chk.D(err) {
		err = errorf.E("error parsing relay url: %s", err)
		return
	}
	if expected.Scheme != found.Scheme {
		err = errorf.E(
			"HTTP Scheme incorrect: expected '%s' got '%s",
			expected.Scheme, found.Scheme,
		)
		return
	}
	if expected.Host != found.Host {
		err = errorf.E(
			"HTTP Host incorrect: expected '%s' got '%s",
			expected.Host, found.Host,
		)
		return
	}
	if expected.Path != found.Path {
		err = errorf.E(
			"HTTP Path incorrect: expected '%s' got '%s",
			expected.Path, found.Path,
		)
		return
	}

	now := time.Now()
	if evt.CreatedAt.Time().After(now.Add(10*time.Minute)) ||
		evt.CreatedAt.Time().Before(now.Add(-10*time.Minute)) {
		err = errorf.E(
			"auth event more than 10 minutes before or after current time",
		)
		return
	}
	// save for last, as it is the most expensive operation
	return evt.Verify()
}
