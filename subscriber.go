package p2p

import "sync"

type Subscriber struct {
	source        <-chan *State
	subscriptions sync.Map
}

func NewSubscriber(source <-chan *State) *Subscriber {
	s := &Subscriber{source: source}
	go s.start()
	return s
}

func (s *Subscriber) Subscribe() (ch <-chan *State, unsubscribe func()) {
	ch = make(chan *State)
	s.subscriptions.Store(ch, true)
	unsubscribe = func() {
		s.subscriptions.Delete(ch)
	}
	return ch, unsubscribe
}

func (s *Subscriber) start() {
	for state := range s.source {
		s.subscriptions.Range(func(key, _ any) bool {
			subscription := key.(chan *State)
			subscription <- state
			return true
		})
	}
}
