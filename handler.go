package p2p

import (
	"context"
	"fmt"
)

type Handler interface {
	ServeP2P(ctx context.Context, m *Message) (data any, err error)
}

type HandlerFunc func(ctx context.Context, m *Message) (any, error)

func (h HandlerFunc) ServeP2P(ctx context.Context, m *Message) (any, error) {
	return h(ctx, m)
}

var DefaultHandler = HandlerFunc(func(ctx context.Context, m *Message) (any, error) {
	data := map[string]any{"status": "received", "subject": m.Subject, "sender": m.From, "body": m.Body}
	return data, nil
})

type MuxHandler struct {
	handlers map[string]Handler
}

func (mx *MuxHandler) Handle(subject string, handler Handler) {
	mx.handlers[subject] = handler
}

func (mx *MuxHandler) HandleFunc(subject string, handler func(context.Context, *Message) (any, error)) {
	mx.Handle(subject, HandlerFunc(handler))
}

func (mx *MuxHandler) ServeP2P(ctx context.Context, m *Message) (any, error) {
	h, ok := mx.handlers[m.Subject]
	if ok {
		return h.ServeP2P(ctx, m)
	}

	return nil, fmt.Errorf("unregistred handler for subject '%s'", m.Subject)
}
