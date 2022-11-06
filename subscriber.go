package p2p

import "sync"

type Subscriber struct {
	source <-chan *State
	subs   sync.Map
}

func NewSubscriber(source <-chan *State) *Subscriber {
	s := &Subscriber{source: source}
	go s.start()
	return s
}

func (s *Subscriber) Subscribe() (clients <-chan *State, unsub func()) {
	ch := make(chan *State)
	s.subs.Store(ch, true)
	unsub = func() {
		s.subs.Delete(ch)
	}
	return ch, unsub
}

func (s *Subscriber) start() {
	for state := range s.source {
		s.subs.Range(func(key, _ any) bool {
			ch := key.(chan *State)
			ch <- state
			return true
		})
	}
}
