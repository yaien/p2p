package p2p

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yaien/p2p/site"
)

type HttpServer struct {
	p2p        *P2P
	subscriber *Subscriber
	key        string
	http.Server
}

func NewHttpServer(p *P2P, s *Subscriber, key string) *HttpServer {
	h := &HttpServer{p2p: p, subscriber: s, key: key}
	handler := http.NewServeMux()
	h.Register(handler)
	h.Server.Handler = handler
	return h
}

// HttpHandler combine the ui handler and the connect handler
func (s *HttpServer) Register(mx *http.ServeMux) {
	s.RegisterUI(mx)
	s.RegisterAPI(mx)
}

// HttpUIHandle the p2p2 real time user interface handler
func (s *HttpServer) RegisterUI(mx *http.ServeMux) {

	mx.Handle("/p2p/", http.StripPrefix("/p2p/", http.FileServer(http.FS(site.FS))))

	mx.HandleFunc("/p2p/sse", func(w http.ResponseWriter, r *http.Request) {

		h := w.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("Transfer-Encoding", "chunked")
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Headers", "Cache-Control")
		h.Set("Access-Control-Allow-Credentials", "true")

		f := w.(http.Flusher)

		updated, unsubscribe := s.subscriber.Subscribe()
		defer unsubscribe()

		for state := range updated {
			msg, _ := json.Marshal(state)
			_, err := fmt.Fprintf(w, "data: %s\n\n", string(msg))
			if err != nil {
				break
			}
			f.Flush()
		}

	})
}

// HttpAPIHandle set the p2p connection endpoints
func (s *HttpServer) RegisterAPI(mx *http.ServeMux) {

	mx.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(s.p2p.State())
	})

	mx.HandleFunc("/api/connect", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")

		var peer Peer
		err := json.NewDecoder(r.Body).Decode(&peer)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}

		if r.Header.Get("X-Signature") != signature(&peer, s.key) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{"error": "invalid signature"})
			return
		}

		s.p2p.Save(&peer)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(s.p2p.State())
	})

	mx.HandleFunc("/api/handle", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("content-type", "application/json")

		var req HttpMessage
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}

		if r.Header.Get("X-Signature") != signature(req.From, s.key) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{"error": "invalid signature"})
			return
		}

		body, err := s.p2p.handler.ServeP2P(r.Context(), &MessageRequest{From: req.From, Subject: req.Subject, Body: req.Body})
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&HttpMessageReply{Body: body})
	})
}
