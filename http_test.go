package p2p

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHttpAPIHandle_Handle(t *testing.T) {
	mx := http.NewServeMux()
	p2p := New(Options{})
	HttpAPIHandle(p2p, mx)

	srv := httptest.NewServer(mx)
	defer srv.Close()

	p2p.SetCurrentAddr(srv.URL)
	var called bool
	p2p.Handle(HandlerFunc(func(ctx context.Context, m *Message) (any, error) {
		called = true
		return "received", nil
	}))

	message := &Message{
		From:    p2p.current,
		Subject: "message",
		Body:    "Hello World",
	}

	b, _ := json.Marshal(message)

	r, err := http.NewRequest("POST", srv.URL+"/api/handle", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("failed creating new request: %s", err)
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Signature", p2p.signature(p2p.current))

	res, err := srv.Client().Do(r)
	if err != nil {
		t.Fatalf("failed doing request: %s", err)
	}

	if !called {
		t.Error("custom handler was no called")
	}

	body, _ := ioutil.ReadAll(res.Body)
	t.Logf("response received: %s", string(body))
}
