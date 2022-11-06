package p2p

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestP2P_Broadcast(t *testing.T) {
	mx := http.NewServeMux()
	p2p := New(Options{})
	HttpAPIHandle(p2p, mx)

	srv := httptest.NewServer(mx)
	defer srv.Close()

	p2p.register(&Client{
		ID:        uuid.New().String(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Addr:      srv.URL,
		Name:      "test-p2p-peer",
	})

	var called bool
	p2p.Handle(HandlerFunc(func(ctx context.Context, m *Message) (any, error) {
		called = true
		log.Println("message received", m)
		return "received", nil
	}))

	err := p2p.Broadcast("*", "message", "Hello World")
	if err != nil {
		t.Fatalf("failed at broadcast: %s", err)
	}

	if !called {
		t.Error("custom handler was no called")
	}
}

func TestP2P_Request(t *testing.T) {
	mx := http.NewServeMux()
	p2p := New(Options{})
	HttpAPIHandle(p2p, mx)

	srv := httptest.NewServer(mx)
	defer srv.Close()

	p2p.register(&Client{
		ID:        uuid.New().String(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Addr:      srv.URL,
		Name:      "test-p2p-peer",
	})

	var called bool
	p2p.Handle(HandlerFunc(func(ctx context.Context, m *Message) (any, error) {
		called = true
		log.Println("message received", m)
		return "received", nil
	}))

	data, err := p2p.Request("test-p2p-peer", "message", "Hello World")
	if err != nil {
		t.Fatalf("failed at broadcast: %s", err)
	}

	if !called {
		t.Error("custom handler was no called")
	}

	log.Println("data received:", string(data))
}
