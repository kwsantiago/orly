package ws

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"orly.dev/pkg/crypto/p256k"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/normalize"

	"golang.org/x/net/websocket"
)

// func TestPublish(t *testing.T) {
// 	// test note to be sent over websocket
// 	var err error
// 	signer := &p256k.Signer{}
// 	if err = signer.Generate(); chk.E(err) {
// 		t.Fatal(err)
// 	}
// 	textNote := &event.E{
// 		Kind:      kind.TextNote,
// 		Content:   []byte("hello"),
// 		CreatedAt: timestamp.FromUnix(1672068534), // random fixed timestamp
// 		Pubkey:    signer.Pub(),
// 	}
// 	if err = textNote.Sign(signer); chk.E(err) {
// 		t.Fatalf("textNote.Sign: %v", err)
// 	}
// 	// fake relay server
// 	var published bool
// 	ws := newWebsocketServer(
// 		func(conn *websocket.Conn) {
// 			// receive message
// 			var raw []json.RawMessage
// 			if err := websocket.JSON.Receive(conn, &raw); chk.T(err) {
// 				t.Errorf("websocket.JSON.Receive: %v", err)
// 			}
// 			// check that it's an EVENT message
// 			if len(raw) < 2 {
// 				t.Errorf("message too short: %v", raw)
// 			}
// 			var msgType string
// 			if err := json.Unmarshal(raw[0], &msgType); chk.T(err) {
// 				t.Errorf("json.Unmarshal: %v", err)
// 			}
// 			if msgType != "EVENT" {
// 				t.Errorf("expected EVENT message, got %q", msgType)
// 			}
// 			// check that the event is the one we sent
// 			var ev event.E
// 			if err := json.Unmarshal(raw[1], &ev); chk.T(err) {
// 				t.Errorf("json.Unmarshal: %v", err)
// 			}
// 			published = true
// 			if !bytes.Equal(ev.ID, textNote.ID) {
// 				t.Errorf(
// 					"event ID mismatch: got %x, want %x",
// 					ev.ID, textNote.ID,
// 				)
// 			}
// 			if !bytes.Equal(ev.Pubkey, textNote.Pubkey) {
// 				t.Errorf(
// 					"event pubkey mismatch: got %x, want %x",
// 					ev.Pubkey, textNote.Pubkey,
// 				)
// 			}
// 			if !bytes.Equal(ev.Content, textNote.Content) {
// 				t.Errorf(
// 					"event content mismatch: got %q, want %q",
// 					ev.Content, textNote.Content,
// 				)
// 			}
// 			fmt.Printf(
// 				"received event: %s\n",
// 				textNote.Serialize(),
// 			)
// 			// send back an ok nip-20 command result
// 			var res []byte
// 			if res = okenvelope.NewFrom(
// 				textNote.ID, true, nil,
// 			).Marshal(res); chk.E(err) {
// 				t.Fatal(err)
// 			}
// 			if err := websocket.Message.Send(conn, res); chk.T(err) {
// 				t.Errorf("websocket.Message.Send: %v", err)
// 			}
// 		},
// 	)
// 	defer ws.Close()
// 	// connect a client and send the text note
// 	rl := mustRelayConnect(ws.URL)
// 	err = rl.Publish(context.Background(), textNote)
// 	if err != nil {
// 		t.Errorf("publish should have succeeded")
// 	}
// 	if !published {
// 		t.Errorf("fake relay server saw no event")
// 	}
// }
//
// func TestPublishBlocked(t *testing.T) {
// 	// test note to be sent over websocket
// 	var err error
// 	signer := &p256k.Signer{}
// 	if err = signer.Generate(); chk.E(err) {
// 		t.Fatal(err)
// 	}
// 	textNote := &event.E{
// 		Kind:      kind.TextNote,
// 		Content:   []byte("hello"),
// 		CreatedAt: timestamp.FromUnix(1672068534), // random fixed timestamp
// 		Pubkey:    signer.Pub(),
// 	}
// 	if err = textNote.Sign(signer); chk.E(err) {
// 		t.Fatalf("textNote.Sign: %v", err)
// 	}
// 	// fake relay server
// 	ws := newWebsocketServer(
// 		func(conn *websocket.Conn) {
// 			// discard received message; not interested
// 			var raw []json.RawMessage
// 			if err := websocket.JSON.Receive(conn, &raw); chk.T(err) {
// 				t.Errorf("websocket.JSON.Receive: %v", err)
// 			}
// 			// send back a not ok nip-20 command result
// 			var res []byte
// 			if res = okenvelope.NewFrom(
// 				textNote.ID, false,
// 				normalize.Msg(normalize.Blocked, "no reason"),
// 			).Marshal(res); chk.E(err) {
// 				t.Fatal(err)
// 			}
// 			if err := websocket.Message.Send(conn, res); chk.T(err) {
// 				t.Errorf("websocket.Message.Send: %v", err)
// 			}
// 			// res := []any{"OK", textNote.ID, false, "blocked"}
// 		},
// 	)
// 	defer ws.Close()
//
// 	// connect a client and send a text note
// 	rl := mustRelayConnect(ws.URL)
// 	if err = rl.Publish(context.Background(), textNote); !chk.E(err) {
// 		t.Errorf("should have failed to publish")
// 	}
// }

func TestPublishWriteFailed(t *testing.T) {
	// test note to be sent over websocket
	var err error
	signer := &p256k.Signer{}
	if err = signer.Generate(); chk.E(err) {
		t.Fatal(err)
	}
	textNote := &event.E{
		Kind:      kind.TextNote,
		Content:   []byte("hello"),
		CreatedAt: timestamp.FromUnix(1672068534), // random fixed timestamp
		Pubkey:    signer.Pub(),
	}
	if err = textNote.Sign(signer); chk.E(err) {
		t.Fatalf("textNote.Sign: %v", err)
	}
	// fake relay server
	ws := newWebsocketServer(
		func(conn *websocket.Conn) {
			// reject receive - force send error
			conn.Close()
		},
	)
	defer ws.Close()

	// connect a client and send a text note
	rl := mustRelayConnect(ws.URL)
	// Force brief period of time so that publish always fails on closed socket.
	time.Sleep(1 * time.Millisecond)
	err = rl.Publish(context.Background(), textNote)
	if err == nil {
		t.Errorf("should have failed to publish")
	}
}

func TestConnectContext(t *testing.T) {
	// fake relay server
	var mu sync.Mutex // guards connected to satisfy go test -race
	var connected bool
	ws := newWebsocketServer(
		func(conn *websocket.Conn) {
			mu.Lock()
			connected = true
			mu.Unlock()
			io.ReadAll(conn) // discard all input
		},
	)
	defer ws.Close()

	// relay client
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	r, err := RelayConnect(ctx, ws.URL)
	if err != nil {
		t.Fatalf("RelayConnectContext: %v", err)
	}
	defer r.Close()

	mu.Lock()
	defer mu.Unlock()
	if !connected {
		t.Error("fake relay server saw no client connect")
	}
}

func TestConnectContextCanceled(t *testing.T) {
	// fake relay server
	ws := newWebsocketServer(discardingHandler)
	defer ws.Close()

	// relay client
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // make ctx expired
	_, err := RelayConnect(ctx, ws.URL)
	if !errors.Is(err, context.Canceled) {
		t.Errorf(
			"RelayConnectContext returned %v error; want context.Canceled", err,
		)
	}
}

func TestConnectWithOrigin(t *testing.T) {
	// fake relay server
	// default handler requires origin golang.org/x/net/websocket
	ws := httptest.NewServer(websocket.Handler(discardingHandler))
	defer ws.Close()

	// relay client
	r := NewRelay(context.Background(), string(normalize.URL(ws.URL)))
	r.requestHeader = http.Header{"origin": {"https://example.com"}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.Connect(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func discardingHandler(conn *websocket.Conn) {
	io.ReadAll(conn) // discard all input
}

func newWebsocketServer(handler func(*websocket.Conn)) (server *httptest.Server) {
	return httptest.NewServer(
		&websocket.Server{
			Handshake: anyOriginHandshake,
			Handler:   handler,
		},
	)
}

// anyOriginHandshake is an alternative to default in golang.org/x/net/websocket
// which checks for origin. nostr client sends no origin and it makes no difference
// for the tests here anyway.
var anyOriginHandshake = func(
	conf *websocket.Config, r *http.Request,
) (err error) {
	return nil
}

func mustRelayConnect(url string) (client *Relay) {
	rl, err := RelayConnect(context.Background(), url)
	if err != nil {
		panic(err.Error())
	}
	return rl
}
