package p2p_test

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yaien/p2p"
	"golang.org/x/net/nettest"
)

func mockup(h p2p.HandlerFunc) (p2p.Handler, *bool) {
	var called bool
	handler := p2p.HandlerFunc(func(ctx context.Context, r *p2p.MessageRequest) ([]byte, error) {
		called = true
		return h.ServeP2P(ctx, r)
	})
	return handler, &called
}

func TestP2P_Http_Broadcast(t *testing.T) {
	network := &p2p.HttpNetwork{}
	from := p2p.New(p2p.Options{Network: network})

	mx := http.NewServeMux()
	srv := httptest.NewServer(mx)
	defer srv.Close()

	to := p2p.New(p2p.Options{Addr: srv.URL, Name: "target-p2p", Network: network})
	p2p.NewHttpServer(to, nil, "").Register(mx)

	handler, called := mockup(func(ctx context.Context, m *p2p.MessageRequest) ([]byte, error) {
		t.Log("message received", m)
		return []byte(`{ "message": "received" }`), nil
	})

	to.Handle(handler)

	err := from.Discover(to.CurrentAddr())
	if err != nil {
		log.Fatalf("failed at discover: %s", err)
	}

	err = from.Broadcast("*", "message", []byte(`{ "message": "Hello World" }`))
	if err != nil {
		t.Fatalf("failed at broadcast: %s", err)
	}

	if !*called {
		t.Error("custom handler was no called")
	}
}

func TestP2P_Http_Request(t *testing.T) {
	network := &p2p.HttpNetwork{}
	from := p2p.New(p2p.Options{Network: network})

	mx := http.NewServeMux()
	srv := httptest.NewServer(mx)
	defer srv.Close()

	to := p2p.New(p2p.Options{Addr: srv.URL, Name: "target-p2p", Network: network})

	handler, called := mockup(func(ctx context.Context, m *p2p.MessageRequest) ([]byte, error) {
		t.Log("message received", m)
		return []byte(`{ "message": "received" }`), nil
	})

	to.Handle(handler)

	p2p.NewHttpServer(to, nil, "").Register(mx)

	err := from.Discover(to.CurrentAddr())
	if err != nil {
		log.Fatalf("failed at discover: %s", err)
	}

	data, err := from.Request("target-p2p", "message", []byte(`{"message": "Hello World" }`))
	if err != nil {
		t.Fatalf("failed at request: %s", err)
	}

	if !*called {
		t.Error("custom handler was no called")
	}

	t.Log("data received:", string(data))
}

func TestP2P_Grpc_Broadcast(t *testing.T) {
	network := &p2p.GrpcNetwork{}
	from := p2p.New(p2p.Options{Network: network})

	lis, err := nettest.NewLocalListener("tcp")
	if err != nil {
		t.Fatalf("failed at creating testing listener: %s", err)
	}

	to := p2p.New(p2p.Options{Addr: lis.Addr().String(), Name: "target-p2p", Network: network})

	handler, called := mockup(func(ctx context.Context, m *p2p.MessageRequest) ([]byte, error) {
		t.Log("message received", m)
		return []byte(`{ "message": "received" }`), nil
	})

	to.Handle(handler)

	srv := p2p.NewGrpcServer(to, nil)
	go srv.Serve(lis)
	defer srv.Close()

	err = from.Discover(to.CurrentAddr())
	if err != nil {
		log.Fatalf("failed at discover: %s", err)
	}

	err = from.Broadcast("*", "message", []byte(`{ "message": "Hello World" }`))
	if err != nil {
		t.Fatalf("failed at broadcast: %s", err)
	}

	if !*called {
		t.Error("custom handler was no called")
	}
}

func TestP2P_Grpc_Request(t *testing.T) {
	network := &p2p.GrpcNetwork{}
	from := p2p.New(p2p.Options{Network: network})

	lis, err := nettest.NewLocalListener("tcp")
	if err != nil {
		t.Fatalf("failed creating testing listener: %s", err)
	}

	to := p2p.New(p2p.Options{Addr: lis.Addr().String(), Name: "target-p2p", Network: network})

	handler, called := mockup(func(ctx context.Context, m *p2p.MessageRequest) ([]byte, error) {
		t.Log("message received", m)
		return []byte(`{ "message": "received" }`), nil
	})

	to.Handle(handler)

	srv := p2p.NewGrpcServer(to, nil)
	go srv.Serve(lis)
	defer srv.Close()

	err = from.Discover(to.CurrentAddr())
	if err != nil {
		log.Fatalf("failed at discover: %s", err)
	}

	data, err := from.Request("target-p2p", "message", []byte(`{"message": "Hello World" }`))
	if err != nil {
		t.Fatalf("failed at request: %s", err)
	}

	if !*called {
		t.Error("custom handler was no called")
	}

	t.Log("data received:", string(data))
}
