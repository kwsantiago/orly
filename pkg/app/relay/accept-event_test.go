package relay

import (
	"bytes"
	"net/http"
	"testing"

	"orly.dev/pkg/app/config"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/utils/context"
)

// mockServerForEvent is a simple mock implementation of the Server struct for testing AcceptEvent
type mockServerForEvent struct {
	authRequired    bool
	ownersFollowed  [][]byte
	followedFollows [][]byte
}

func (m *mockServerForEvent) AuthRequired() bool {
	return m.authRequired
}

func (m *mockServerForEvent) OwnersFollowed() [][]byte {
	return m.ownersFollowed
}

func (m *mockServerForEvent) FollowedFollows() [][]byte {
	return m.followedFollows
}

// AcceptEvent implements the Server.AcceptEvent method for testing
func (m *mockServerForEvent) AcceptEvent(
	c context.T, ev *event.E, hr *http.Request, authedPubkey []byte,
	remote string,
) (accept bool, notice string, afterSave func()) {
	// if auth is required and the user is not authed, reject
	if m.AuthRequired() && len(authedPubkey) == 0 {
		return
	}
	// check if the authed user is on the lists
	list := append(m.OwnersFollowed(), m.FollowedFollows()...)
	for _, u := range list {
		if utils.FastEqual(u, authedPubkey) {
			accept = true
			break
		}
	}
	return
}

func TestAcceptEvent(t *testing.T) {
	// Create a context and HTTP request for testing
	ctx := context.Bg()
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	// Create a test event
	testEvent := &event.E{}

	// Test cases
	tests := []struct {
		name           string
		server         *mockServerForEvent
		authedPubkey   []byte
		expectedAccept bool
	}{
		{
			name: "Auth required, no pubkey",
			server: &mockServerForEvent{
				authRequired: true,
			},
			authedPubkey:   nil,
			expectedAccept: false,
		},
		{
			name: "Auth required, with pubkey, not on lists",
			server: &mockServerForEvent{
				authRequired: true,
				ownersFollowed: [][]byte{
					[]byte("followed1"),
					[]byte("followed2"),
				},
				followedFollows: [][]byte{
					[]byte("follow1"),
					[]byte("follow2"),
				},
			},
			authedPubkey:   []byte("test-pubkey"),
			expectedAccept: false,
		},
		{
			name: "Auth required, with pubkey, on owners followed list",
			server: &mockServerForEvent{
				authRequired: true,
				ownersFollowed: [][]byte{
					[]byte("followed1"),
					[]byte("test-pubkey"),
					[]byte("followed2"),
				},
				followedFollows: [][]byte{
					[]byte("follow1"),
					[]byte("follow2"),
				},
			},
			authedPubkey:   []byte("test-pubkey"),
			expectedAccept: true,
		},
		{
			name: "Auth required, with pubkey, on followed follows list",
			server: &mockServerForEvent{
				authRequired: true,
				ownersFollowed: [][]byte{
					[]byte("followed1"),
					[]byte("followed2"),
				},
				followedFollows: [][]byte{
					[]byte("follow1"),
					[]byte("test-pubkey"),
					[]byte("follow2"),
				},
			},
			authedPubkey:   []byte("test-pubkey"),
			expectedAccept: true,
		},
		{
			name: "Auth not required, no pubkey, not on lists",
			server: &mockServerForEvent{
				authRequired: false,
				ownersFollowed: [][]byte{
					[]byte("followed1"),
					[]byte("followed2"),
				},
				followedFollows: [][]byte{
					[]byte("follow1"),
					[]byte("follow2"),
				},
			},
			authedPubkey:   nil,
			expectedAccept: false,
		},
		{
			name: "Auth not required, with pubkey, on lists",
			server: &mockServerForEvent{
				authRequired: false,
				ownersFollowed: [][]byte{
					[]byte("followed1"),
					[]byte("test-pubkey"),
					[]byte("followed2"),
				},
				followedFollows: [][]byte{
					[]byte("follow1"),
					[]byte("follow2"),
				},
			},
			authedPubkey:   []byte("test-pubkey"),
			expectedAccept: true,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				// Use the mock server's AcceptEvent method
				accept, notice, afterSave := tt.server.AcceptEvent(
					ctx, testEvent, req, tt.authedPubkey, "127.0.0.1",
				)

				// Check if the acceptance status matches the expected value
				if accept != tt.expectedAccept {
					t.Errorf(
						"AcceptEvent() accept = %v, want %v", accept,
						tt.expectedAccept,
					)
				}

				// Notice should be empty in the current implementation
				if notice != "" {
					t.Errorf(
						"AcceptEvent() notice = %v, want empty string", notice,
					)
				}

				// afterSave should be nil in the current implementation
				if afterSave != nil {
					t.Error("AcceptEvent() afterSave is not nil, but should be nil")
				}
			},
		)
	}
}

// TestAcceptEventWithRealServer tests the AcceptEvent function with a real Server instance
func TestAcceptEventWithRealServer(t *testing.T) {
	// Create a context and HTTP request for testing
	ctx := context.Bg()
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	// Create a test event
	testEvent := &event.E{}

	// Create a Server instance with configuration
	s := &Server{
		C: &config.C{
			AuthRequired: true,
		},
		Lists: new(Lists),
	}

	// Test with no authenticated pubkey
	accept, notice, afterSave := s.AcceptEvent(
		ctx, testEvent, req, nil, "127.0.0.1",
	)
	if accept {
		t.Error("AcceptEvent() accept = true, want false")
	}
	if notice != "client isn't authed" {
		t.Errorf(
			"AcceptEvent() notice = %v, want 'client isn't authed'", notice,
		)
	}
	if afterSave != nil {
		t.Error("AcceptEvent() afterSave is not nil, but should be nil")
	}

	// Test with authenticated pubkey but not on any list
	accept, notice, afterSave = s.AcceptEvent(
		ctx, testEvent, req, []byte("test-pubkey"), "127.0.0.1",
	)
	if accept {
		t.Error("AcceptEvent() accept = true, want false")
	}

	// Add the pubkey to the owners followed list
	s.SetOwnersFollowed([][]byte{[]byte("test-pubkey")})

	// Test with authenticated pubkey on the owners followed list
	accept, notice, afterSave = s.AcceptEvent(
		ctx, testEvent, req, []byte("test-pubkey"), "127.0.0.1",
	)
	if !accept {
		t.Error("AcceptEvent() accept = false, want true")
	}

	// Clear the owners followed list and add the pubkey to the followed follows list
	s.SetOwnersFollowed(nil)
	s.SetFollowedFollows([][]byte{[]byte("test-pubkey")})

	// Test with authenticated pubkey on the followed follows list
	accept, notice, afterSave = s.AcceptEvent(
		ctx, testEvent, req, []byte("test-pubkey"), "127.0.0.1",
	)
	if !accept {
		t.Error("AcceptEvent() accept = false, want true")
	}

	// Test with muted user
	s.SetOwnersMuted([][]byte{[]byte("test-pubkey")})
	accept, notice, afterSave = s.AcceptEvent(
		ctx, testEvent, req, []byte("test-pubkey"), "127.0.0.1",
	)
	if accept {
		t.Error("AcceptEvent() accept = true, want false")
	}
	if notice != "event author is banned from this relay" {
		t.Errorf(
			"AcceptEvent() notice = %v, want 'event author is banned from this relay'",
			notice,
		)
	}
}

// TestAcceptEventWithBlacklist tests the blacklist functionality when auth is not required
func TestAcceptEventWithBlacklist(t *testing.T) {
	// Create a context and HTTP request for testing
	ctx := context.Bg()
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	// Test pubkey bytes
	testPubkey := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c,
		0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
	}
	blockedPubkey := []byte{
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c,
		0x1d, 0x1e, 0x1f, 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
		0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30,
	}

	// Test with public relay mode (auth not required) and no blacklist
	s := &Server{
		C: &config.C{
			AuthRequired: false,
		},
		Lists: new(Lists),
	}

	// Create event with test pubkey
	testEvent := &event.E{}
	testEvent.Pubkey = testPubkey

	// Should accept when no blacklist
	accept, notice, _ := s.AcceptEvent(ctx, testEvent, req, nil, "127.0.0.1")
	if !accept {
		t.Error("AcceptEvent() accept = false, want true")
	}
	if notice != "" {
		t.Errorf("AcceptEvent() notice = %v, want empty string", notice)
	}

	// Add blacklist with different pubkey
	s.blacklistPubkeys = [][]byte{blockedPubkey}

	// Should still accept when author not in blacklist
	accept, notice, _ = s.AcceptEvent(ctx, testEvent, req, nil, "127.0.0.1")
	if !accept {
		t.Error("AcceptEvent() accept = false, want true")
	}
	if notice != "" {
		t.Errorf("AcceptEvent() notice = %v, want empty string", notice)
	}

	// Create event with blocked pubkey
	blockedEvent := &event.E{}
	blockedEvent.Pubkey = blockedPubkey

	// Should reject when author is in blacklist
	accept, notice, _ = s.AcceptEvent(ctx, blockedEvent, req, nil, "127.0.0.1")
	if accept {
		t.Error("AcceptEvent() accept = true, want false")
	}
	if notice != "event author is blacklisted" {
		t.Errorf(
			"AcceptEvent() notice = %v, want 'event author is blacklisted'",
			notice,
		)
	}

	// Test with auth required - blacklist should not apply
	s.C.AuthRequired = true
	accept, notice, _ = s.AcceptEvent(ctx, blockedEvent, req, nil, "127.0.0.1")
	if accept {
		t.Error("AcceptEvent() accept = true, want false")
	}
	if notice != "client isn't authed" {
		t.Errorf(
			"AcceptEvent() notice = %v, want 'client isn't authed'", notice,
		)
	}
}
