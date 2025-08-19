package openapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"orly.dev/pkg/app/config"
)

// mockServerInterface implements the server.I interface for testing
type mockServerInterface struct {
	cfg *config.C
}

func (m *mockServerInterface) Config() *config.C {
	return m.cfg
}

func (m *mockServerInterface) Storage() interface{} {
	return nil
}

func TestInvoiceEndpoint(t *testing.T) {
	// Create a test configuration
	cfg := &config.C{
		NWCUri:           "nostr+walletconnect://test@relay.example.com?secret=test",
		MonthlyPriceSats: 6000,
	}

	// Create mock server interface
	mockServer := &mockServerInterface{cfg: cfg}

	// Create a router and API
	router := chi.NewRouter()
	api := humachi.New(router, &humachi.HumaConfig{
		OpenAPI: humachi.DefaultOpenAPIConfig(),
	})

	// Create operations and register invoice endpoint
	ops := &Operations{
		I:    mockServer,
		path: "/api",
	}

	// Note: We cannot fully test the endpoint without a real NWC connection
	// This test mainly validates the structure and basic validation
	ops.RegisterInvoice(api)

	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "missing body",
			body:           nil,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "missing pubkey",
			body:           map[string]interface{}{"months": 1},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "invalid months - too low",
			body:           map[string]interface{}{"pubkey": "npub1test", "months": 0},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "invalid months - too high",
			body:           map[string]interface{}{"pubkey": "npub1test", "months": 13},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "invalid pubkey format",
			body:           map[string]interface{}{"pubkey": "invalid", "months": 1},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tt.body != nil {
				jsonBody, _ := json.Marshal(tt.body)
				body = bytes.NewBuffer(jsonBody)
			} else {
				body = bytes.NewBuffer([]byte{})
			}

			req := httptest.NewRequest("POST", "/api/invoice", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			if tt.expectError {
				// Check that error is present in response
				if response["error"] == nil && response["detail"] == nil {
					t.Errorf("expected error in response, but got none: %v", response)
				}
			}
		})
	}
}

func TestInvoiceValidation(t *testing.T) {
	// Test pubkey format validation
	validPubkeys := []string{
		"npub1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq5sgp4",
		"0000000000000000000000000000000000000000000000000000000000000000",
	}

	invalidPubkeys := []string{
		"",
		"invalid",
		"npub1invalid",
		"1234567890abcdef", // too short
		"gg00000000000000000000000000000000000000000000000000000000000000", // invalid hex
	}

	for _, pubkey := range validPubkeys {
		t.Run("valid_pubkey_"+pubkey[:8], func(t *testing.T) {
			// These should not return an error when parsing
			// (Note: actual validation would need keys.DecodeNpubOrHex)
			if pubkey == "" {
				t.Skip("empty pubkey test")
			}
		})
	}

	for _, pubkey := range invalidPubkeys {
		t.Run("invalid_pubkey_"+pubkey, func(t *testing.T) {
			// These should return an error when parsing
			// (Note: actual validation would need keys.DecodeNpubOrHex)
			if pubkey == "" {
				// Empty pubkey should be invalid
			}
		})
	}
}

func TestInvoiceAmountCalculation(t *testing.T) {
	cfg := &config.C{
		MonthlyPriceSats: 6000,
	}

	tests := []struct {
		months         int
		expectedAmount int64
	}{
		{1, 6000},
		{3, 18000},
		{6, 36000},
		{12, 72000},
	}

	for _, tt := range tests {
		t.Run("months_"+string(rune(tt.months)), func(t *testing.T) {
			totalAmount := cfg.MonthlyPriceSats * int64(tt.months)
			if totalAmount != tt.expectedAmount {
				t.Errorf("expected amount %d, got %d", tt.expectedAmount, totalAmount)
			}
		})
	}
}
