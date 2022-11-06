package p2p

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestP2P_Broadcast(t *testing.T) {
	p2p := New(Options{})

	mx := http.NewServeMux()
	srv := httptest.NewServer(mx)
	defer srv.Close()

	target := New(Options{Addr: srv.URL, Name: "target-p2p"})

	var called bool
	target.Handle(HandlerFunc(func(ctx context.Context, m *Message) (any, error) {
		called = true
		t.Log("message received", m)
		return "received", nil
	}))

	HttpAPIHandle(target, mx)

	p2p.register(target.current)

	err := p2p.Broadcast("*", "message", "Hello World")
	if err != nil {
		t.Fatalf("failed at broadcast: %s", err)
	}

	if !called {
		t.Error("custom handler was no called")
	}
}

func TestP2P_Request(t *testing.T) {
	p2p := New(Options{})

	mx := http.NewServeMux()
	srv := httptest.NewServer(mx)
	defer srv.Close()

	target := New(Options{
		Addr: srv.URL,
		Name: "target-p2p",
	})

	var called bool
	target.Handle(HandlerFunc(func(ctx context.Context, m *Message) (any, error) {
		called = true
		t.Log("message received", m)
		return "received", nil
	}))

	HttpAPIHandle(target, mx)
	p2p.register(target.current)

	t.Logf("current client signature: '%s'", p2p.signature(p2p.current))
	t.Logf("target client %s: %s, signature: '%s'", target.current.Name, target.current.Addr, target.signature(target.current))

	data, err := p2p.Request("target-p2p", "message", "Hello World")
	if err != nil {
		t.Fatalf("failed at broadcast: %s", err)
	}

	if !called {
		t.Error("custom handler was no called")
	}

	t.Log("data received:", data)
}
