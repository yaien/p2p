package p2p

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
)

var ErrNoneMatchedPeers = errors.New("none peer matched the pattern")

type Message struct {
	From    *Client `json:"from"`
	Subject string  `json:"string"`
	Body    any     `json:"body"`
}

type Handler interface {
	Handle(ctx context.Context, m *Message) (data any, err error)
}

type HandlerFunc func(ctx context.Context, m *Message) (any, error)

func (h HandlerFunc) Handle(ctx context.Context, m *Message) (any, error) {
	return h(ctx, m)
}

var DefaultHandler = HandlerFunc(func(ctx context.Context, m *Message) (any, error) {
	data := map[string]any{"status": "received", "subject": m.Subject, "sender": m.From, "body": m.Body}
	return data, nil
})

func (p *P2P) Handle(h Handler) {
	p.handler = h
}

func (p *P2P) Broadcast(pattern string, subj string, body any) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	message := &Message{From: p.current, Body: body}
	b, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed marshall: %w", err)
	}

	for _, c := range p.clients {
		if !regex.MatchString(c.Name) {
			continue
		}

		req, err := http.NewRequest("POST", c.Addr+"/api/handle", bytes.NewReader(b))
		if err != nil {
			return fmt.Errorf("failed creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Signature", p.signature(p.current))

		var h http.Client
		h.Do(req)
	}

	return nil
}

func (p *P2P) Request(pattern string, body any) (*http.Response, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	message := &Message{From: p.current, Body: body}
	b, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed marshall: %w", err)
	}

	for _, c := range p.clients {
		if !regex.MatchString(c.Name) {
			continue
		}

		req, err := http.NewRequest("POST", c.Addr+"/api/handle", bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("failed creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Signature", p.signature(p.current))

		var h http.Client
		return h.Do(req)
	}

	return nil, fmt.Errorf("%w %s", ErrNoneMatchedPeers, pattern)

}
