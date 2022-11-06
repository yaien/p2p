package p2p

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidSignature = errors.New("invalid signature")

type Client struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Addr       string    `json:"addr"`
	RefresedAt time.Time `json:"refreshedAt"`
}

type P2P struct {
	current *Client
	lookup  []string
	key     string
	clients map[string]*Client
	channel chan *State
	mutex   sync.RWMutex
	addr    string
	handler Handler
}

type Options struct {
	Name   string
	Addr   string
	Key    string
	Lookup []string
}

func New(opts Options) *P2P {

	current := &Client{
		ID:         uuid.New().String(),
		Name:       opts.Name,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Addr:       opts.Addr,
		RefresedAt: time.Now(),
	}

	p := &P2P{
		addr:    opts.Addr,
		current: current,
		lookup:  opts.Lookup,
		key:     opts.Key,
		clients: make(map[string]*Client),
		channel: make(chan *State),
		handler: DefaultHandler,
	}

	return p
}

func (p *P2P) Addr() string {
	return p.addr
}

func (p *P2P) CurrentAddr() string {
	return p.current.Addr
}

func (p *P2P) SetCurrentAddr(addr string) {
	p.current.Addr = addr
	p.current.UpdatedAt = time.Now()
	p.current.RefresedAt = time.Now()
}

func (p *P2P) Handle(h Handler) {
	p.handler = h
}

func (p *P2P) Channel() <-chan *State {
	return p.channel
}

func (p *P2P) Clients() []*Client {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	clients := make([]*Client, 0, len(p.clients))
	for _, cl := range p.clients {
		clients = append(clients, cl)
	}
	return clients
}

func (p *P2P) State() *State {
	return &State{Current: p.current, Clients: p.Clients()}
}

func (p *P2P) Start() {
	for {
		p.scan()
		time.Sleep(5 * time.Second)
	}
}

func (p *P2P) Save(signature string, client *Client) error {
	hash := p.signature(client)
	if hash != signature {
		return ErrInvalidSignature
	}
	p.register(client)
	p.notify()
	return nil
}

func (p *P2P) scan() {
	if len(p.clients) == 0 && len(p.lookup) > 0 {
		for _, addr := range p.lookup {
			err := p.discover(addr)
			if err != nil {
				log.Printf("failed lookup %s\n", err)
			}
			log.Println("connected to", addr)
		}
	}

	for addr, client := range p.clients {
		err := p.discover(client.Addr)
		if err != nil {
			p.mutex.Lock()
			delete(p.clients, addr)
			p.mutex.Unlock()
			log.Println("client disconnected", client.Addr, err)
			continue
		}
	}

	p.notify()
}

func (p *P2P) register(cl *Client) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.current.Addr == cl.Addr {
		return
	}

	cl.RefresedAt = time.Now()
	p.clients[cl.Addr] = cl
}

func (p *P2P) discover(target string) error {

	var buff bytes.Buffer
	err := json.NewEncoder(&buff).Encode(p.current)
	if err != nil {
		return fmt.Errorf("failed encoding current: %w", err)
	}

	url := fmt.Sprintf("%s/api/connect", target)
	req, err := http.NewRequest("POST", url, &buff)
	if err != nil {
		return fmt.Errorf("failed creating request: %w", err)
	}

	req.Header.Set("X-Signature", p.signature(p.current))
	req.Header.Set("Content-Type", "application/json")

	var h http.Client
	res, err := h.Do(req)
	if err != nil {
		return fmt.Errorf("failed at post request: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("req to %s failed with status %d", req.URL, res.StatusCode)
	}

	var st State
	err = json.NewDecoder(res.Body).Decode(&st)
	if err != nil {
		return fmt.Errorf("failed decoding body: %w", err)
	}

	clients := append(st.Clients, st.Current)
	for _, cl := range clients {
		p.register(cl)
	}
	return nil
}

func (p *P2P) signature(client *Client) string {
	source := []byte(client.ID + client.Name + client.Addr + p.key)
	return fmt.Sprintf("%x", sha256.Sum256(source))
}

func (p *P2P) notify() {
	go func() {
		p.channel <- p.State()
	}()
}

type State struct {
	Current *Client   `json:"current"`
	Clients []*Client `json:"clients"`
}
