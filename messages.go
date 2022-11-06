package p2p

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
)

var ErrNoneMatchedPeers = errors.New("none peer matched the pattern")

type Message struct {
	From    *Client `json:"from"`
	Subject string  `json:"string"`
	Body    any     `json:"body"`
}

type reply struct {
	Data  any    `json:"data"`
	Error string `json:"error"`
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

	message := &Message{From: p.current, Subject: subj, Body: body}
	b, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed marshall: %w", err)
	}

	for _, c := range p.clients {
		matched, err := path.Match(pattern, c.Name)
		if err != nil {
			return fmt.Errorf("failed at pattern match: %w", err)
		}

		if !matched {
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

func (p *P2P) Request(pattern string, subj string, body any) (data []byte, err error) {

	message := &Message{From: p.current, Subject: subj, Body: body}
	b, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed marshall: %w", err)
	}

	for _, c := range p.clients {
		matched, err := path.Match(pattern, c.Name)
		if err != nil {
			return nil, fmt.Errorf("failed at pattern match: %w", err)
		}

		if !matched {
			continue
		}

		req, err := http.NewRequest("POST", c.Addr+"/api/handle", bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("failed creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Signature", p.signature(p.current))

		var h http.Client
		res, err := h.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed doing request: %w", err)
		}

		var r reply
		err = json.NewDecoder(res.Body).Decode(&r)
		if err != nil {
			return nil, fmt.Errorf("failed at decoding response: %w", err)
		}

		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("request failed with status %d: %s", res.StatusCode, r.Error)
		}

		return json.Marshal(r.Data)
	}

	return nil, fmt.Errorf("%w %s", ErrNoneMatchedPeers, pattern)

}
