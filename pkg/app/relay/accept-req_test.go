package relay

import (
	"net/http"
	"testing"

	"orly.dev/pkg/app/config"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/utils/context"
)

// mockServer is a simple mock implementation of the Server struct for testing
type mockServer struct {
	authRequired   bool
	publicReadable bool
	ownersPubkeys  [][]byte
}

func (m *mockServer) AuthRequired() bool {
	return m.authRequired || m.LenOwnersPubkeys() > 0
}

func (m *mockServer) PublicReadable() bool {
	return m.publicReadable
}

func (m *mockServer) LenOwnersPubkeys() int {
	return len(m.ownersPubkeys)
}

func (m *mockServer) OwnersFollowed() [][]byte {
	return nil
}

func (m *mockServer) FollowedFollows() [][]byte {
	return nil
}

// AcceptReq implements the Server.AcceptReq method for testing
func (m *mockServer) AcceptReq(
	c context.T, hr *http.Request, ff *filters.T,
	authedPubkey []byte, remote string,
) (allowed *filters.T, accept bool, modified bool) {
	// if auth is required, and not public readable, reject
	if m.AuthRequired() && len(authedPubkey) == 0 && !m.PublicReadable() {
		return
	}
	allowed = ff
	accept = true
	return
}

func TestAcceptReq(t *testing.T) {
	// Create a context and HTTP request for testing
	ctx := context.Bg()
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	// Create test filters
	testFilters := filters.New()

	// Test cases
	tests := []struct {
		name           string
		server         *mockServer
		authedPubkey   []byte
		expectedAccept bool
	}{
		{
			name: "Auth required, no pubkey, not public readable",
			server: &mockServer{
				authRequired:   true,
				publicReadable: false,
			},
			authedPubkey:   nil,
			expectedAccept: false,
		},
		{
			name: "Auth required, no pubkey, public readable",
			server: &mockServer{
				authRequired:   true,
				publicReadable: true,
			},
			authedPubkey:   nil,
			expectedAccept: true,
		},
		{
			name: "Auth required, with pubkey",
			server: &mockServer{
				authRequired:   true,
				publicReadable: false,
			},
			authedPubkey:   []byte("test-pubkey"),
			expectedAccept: true,
		},
		{
			name: "Auth not required",
			server: &mockServer{
				authRequired:   false,
				publicReadable: false,
			},
			authedPubkey:   nil,
			expectedAccept: true,
		},
		{
			name: "Auth required due to owner pubkeys, no pubkey, not public readable",
			server: &mockServer{
				authRequired:   false,
				publicReadable: false,
				ownersPubkeys:  [][]byte{[]byte("owner1")},
			},
			authedPubkey:   nil,
			expectedAccept: false,
		},
		{
			name: "Auth required due to owner pubkeys, no pubkey, public readable",
			server: &mockServer{
				authRequired:   false,
				publicReadable: true,
				ownersPubkeys:  [][]byte{[]byte("owner1")},
			},
			authedPubkey:   nil,
			expectedAccept: true,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the mock server's AcceptReq method
			allowed, accept, modified := tt.server.AcceptReq(ctx, req, testFilters, tt.authedPubkey, "127.0.0.1")

			// Check if the acceptance status matches the expected value
			if accept != tt.expectedAccept {
				t.Errorf("AcceptReq() accept = %v, want %v", accept, tt.expectedAccept)
			}

			// If the request should be accepted, check that the filters are returned
			if tt.expectedAccept {
				if allowed == nil {
					t.Error("AcceptReq() allowed is nil, but request was accepted")
				}
			} else {
				if allowed != nil {
					t.Error("AcceptReq() allowed is not nil, but request was rejected")
				}
			}

			// Modified should be false as the current implementation doesn't modify filters
			if modified {
				t.Error("AcceptReq() modified = true, want false")
			}
		})
	}
}

// TestAcceptReqWithRealServer tests the AcceptReq function with a real Server instance
func TestAcceptReqWithRealServer(t *testing.T) {
	// Create a context and HTTP request for testing
	ctx := context.Bg()
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	// Create test filters
	testFilters := filters.New()

	// Create a Server instance with configuration
	s := &Server{
		C: &config.C{
			AuthRequired:   true,
			PublicReadable: false,
		},
		Lists: new(Lists),
	}

	// Test with no authenticated pubkey
	allowed, accept, modified := s.AcceptReq(ctx, req, testFilters, nil, "127.0.0.1")
	if accept {
		t.Error("AcceptReq() accept = true, want false")
	}
	if allowed != nil {
		t.Error("AcceptReq() allowed is not nil, but request was rejected")
	}
	if modified {
		t.Error("AcceptReq() modified = true, want false")
	}

	// Test with authenticated pubkey
	allowed, accept, modified = s.AcceptReq(ctx, req, testFilters, []byte("test-pubkey"), "127.0.0.1")
	if !accept {
		t.Error("AcceptReq() accept = false, want true")
	}
	if allowed != testFilters {
		t.Error("AcceptReq() allowed is not the same as input filters")
	}
	if modified {
		t.Error("AcceptReq() modified = true, want false")
	}

	// Test with public readable
	s.C.PublicReadable = true
	allowed, accept, modified = s.AcceptReq(ctx, req, testFilters, nil, "127.0.0.1")
	if !accept {
		t.Error("AcceptReq() accept = false, want true")
	}
	if allowed != testFilters {
		t.Error("AcceptReq() allowed is not the same as input filters")
	}
	if modified {
		t.Error("AcceptReq() modified = true, want false")
	}
}
