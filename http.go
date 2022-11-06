package p2p

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yaien/p2p/site"
)

// HttpHandler combine the ui handler and the connect handler
func HttpHandle(p2p *P2P, mx *http.ServeMux) {
	HttpUIHandle(p2p, mx)
	HttpAPIHandle(p2p, mx)
}

// HttpUIHandle the p2p2 real time user interface handler
func HttpUIHandle(p2p *P2P, mx *http.ServeMux) {

	sb := NewSubscriber(p2p.Channel())

	mx.Handle("/p2p/", http.StripPrefix("/p2p/", http.FileServer(http.FS(site.FS))))

	mx.Handle("/p2p/sse", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		h := w.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("Transfer-Encoding", "chunked")
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Headers", "Cache-Control")
		h.Set("Access-Control-Allow-Credentials", "true")

		f := w.(http.Flusher)

		updated, unsub := sb.Subscribe()
		defer unsub()

		for state := range updated {
			msg, _ := json.Marshal(state)
			_, err := fmt.Fprintf(w, "data: %s\n\n", string(msg))
			if err != nil {
				break
			}
			f.Flush()
		}

	}))
}

// HttpAPIHandle set the p2p connection endpoints
func HttpAPIHandle(p2p *P2P, mx *http.ServeMux) {

	mx.Handle("/api/state", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(p2p.State())
	}))

	mx.Handle("/api/connect", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var cl Client
		err := json.NewDecoder(r.Body).Decode(&cl)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}

		signature := r.Header.Get("X-Signature")

		err = p2p.Save(signature, &cl)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(p2p.State())
	}))

	mx.Handle("/api/handle", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("content-type", "application/json")

		var message Message
		err := json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}

		if r.Header.Get("X-Signature") != p2p.signature(p2p.current) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{"error": "invalid signature"})
			return
		}

		data, err := p2p.handler.Handle(r.Context(), &message)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": data})

	}))
}
