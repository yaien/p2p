package p2p

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Transport interface {
	Connect(from *Peer, addr string) (*State, error)
	Send(from, to *Peer, subject string, body []byte) ([]byte, error)
}

type Server interface {
	Serve(l net.Listener) error
	Close() error
}

type P2P struct {
	current   *Peer
	lookup    []string
	peers     map[string]*Peer
	channel   chan *State
	mutex     sync.RWMutex
	handler   Handler
	transport Transport
}

type Options struct {
	Name      string
	Addr      string
	Lookup    []string
	Transport Transport
}

func New(opts Options) *P2P {

	current := &Peer{
		Id:          uuid.New().String(),
		Name:        opts.Name,
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
		Addr:        opts.Addr,
		RefreshedAt: time.Now().Format(time.RFC3339),
	}

	p := &P2P{
		current:   current,
		lookup:    opts.Lookup,
		peers:     make(map[string]*Peer),
		channel:   make(chan *State),
		handler:   NewServeMux(),
		transport: opts.Transport,
	}

	return p
}

func (p *P2P) CurrentAddr() string {
	return p.current.Addr
}

func (p *P2P) SetCurrentAddr(addr string) {
	p.current.Addr = addr
	p.current.UpdatedAt = time.Now().Format(time.RFC3339)
	p.current.RefreshedAt = time.Now().Format(time.RFC3339)
}

func (p *P2P) SetTransport(n Transport) {
	p.transport = n
}

func (p *P2P) Handle(h Handler) {
	p.handler = h
}

func (p *P2P) HandleFunc(f func(ctx context.Context, r *MessageRequest) ([]byte, error)) {
	p.Handle(HandlerFunc(f))
}

func (p *P2P) Channel() <-chan *State {
	return p.channel
}

func (p *P2P) Peers() []*Peer {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	clients := make([]*Peer, 0, len(p.peers))
	for _, cl := range p.peers {
		clients = append(clients, cl)
	}
	return clients
}

func (p *P2P) State() *State {
	return &State{Current: p.current, Peers: p.Peers()}
}

func (p *P2P) Start() {
	for {
		p.scan()
		time.Sleep(5 * time.Second)
	}
}

func (p *P2P) Save(peer *Peer) {
	p.register(peer)
	p.notify()
}

func (p *P2P) scan() {
	if len(p.peers) == 0 && len(p.lookup) > 0 {
		for _, addr := range p.lookup {
			err := p.Discover(addr)
			if err != nil {
				log.Printf("failed lookup %s\n", err)
				continue
			}

			log.Println("connected to", addr)
		}
	}

	for id, client := range p.peers {
		err := p.Discover(client.Addr)
		if err != nil {
			p.mutex.Lock()
			delete(p.peers, id)
			p.mutex.Unlock()
			log.Println("client disconnected", client.Addr, err)
			continue
		}
	}

	p.notify()
}

func (p *P2P) register(peer *Peer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.current.Id == peer.Id {
		return
	}

	peer.RefreshedAt = time.Now().Format(time.RFC3339)
	p.peers[peer.Id] = peer
}

func (p *P2P) Discover(target string) error {
	state, err := p.transport.Connect(p.current, target)
	if err != nil {
		return fmt.Errorf("failed at transport connect: %w", err)
	}

	peers := append(state.Peers, state.Current)
	for _, peer := range peers {
		p.register(peer)
	}
	return nil
}

func (p *P2P) notify() {
	go func() {
		p.channel <- p.State()
	}()
}
