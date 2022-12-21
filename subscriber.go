package p2p

import "sync"

type Subscriber struct {
	source        <-chan *State
	subscriptions sync.Map
}

type UnsubscribeFunc func()

func NewSubscriber(source <-chan *State) *Subscriber {
	s := &Subscriber{source: source}
	go s.start()
	return s
}

func (s *Subscriber) Subscribe() (<-chan *State, UnsubscribeFunc) {
	channel := make(chan *State)
	s.subscriptions.Store(channel, true)
	unsubscribe := func() {
		s.subscriptions.Delete(channel)
	}
	return channel, unsubscribe
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
