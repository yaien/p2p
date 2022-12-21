package p2p

import (
	"context"
	"fmt"
)

type Handler interface {
	ServeP2P(ctx context.Context, r *MessageRequest) (data []byte, err error)
}

type HandlerFunc func(ctx context.Context, r *MessageRequest) ([]byte, error)

func (h HandlerFunc) ServeP2P(ctx context.Context, r *MessageRequest) ([]byte, error) {
	return h(ctx, r)
}

type ServeMux struct {
	handlers map[string]Handler
}

func NewServeMux() *ServeMux {
	return &ServeMux{handlers: make(map[string]Handler)}
}

func (mx *ServeMux) Handle(subject string, handler Handler) {
	mx.handlers[subject] = handler
}

func (mx *ServeMux) HandleFunc(subject string, handler func(context.Context, *MessageRequest) ([]byte, error)) {
	mx.Handle(subject, HandlerFunc(handler))
}

func (mx *ServeMux) ServeP2P(ctx context.Context, m *MessageRequest) ([]byte, error) {
	h, ok := mx.handlers[m.Subject]
	if ok {
		return h.ServeP2P(ctx, m)
	}

	return nil, fmt.Errorf("unregistered handler for subject '%s'", m.Subject)
}
