package p2p

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
)

type HttpMessage struct {
	From    *Peer           `json:"from"`
	Subject string          `json:"subject"`
	Body    json.RawMessage `json:"body"`
}

type HttpMessageReply struct {
	Body  json.RawMessage `json:"body"`
	Error string          `json:"error"`
}

type HttpNetwork struct {
	Key string
}

func signature(p *Peer, key string) string {
	source := []byte(p.Id + p.Name + p.Addr + key)
	return fmt.Sprintf("%x", sha256.Sum256(source))
}

func (n *HttpNetwork) Connect(from *Peer, addr string) (*State, error) {
	var buff bytes.Buffer
	err := json.NewEncoder(&buff).Encode(from)
	if err != nil {
		return nil, fmt.Errorf("failed encoding current: %w", err)
	}

	url := fmt.Sprintf("%s/api/connect", addr)
	req, err := http.NewRequest("POST", url, &buff)
	if err != nil {
		return nil, fmt.Errorf("failed creating request: %w", err)
	}

	req.Header.Set("X-Signature", signature(from, n.Key))
	req.Header.Set("Content-Type", "application/json")

	var h http.Client
	res, err := h.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed at post request: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("req to %s failed with status %d", req.URL, res.StatusCode)
	}

	var state State
	err = json.NewDecoder(res.Body).Decode(&state)
	if err != nil {
		return nil, fmt.Errorf("failed decoding response body: %w", err)
	}

	return &state, nil
}

func (n *HttpNetwork) Send(from, to *Peer, subject string, body []byte) ([]byte, error) {
	var buff bytes.Buffer
	err := json.NewEncoder(&buff).Encode(&HttpMessage{From: from, Subject: subject, Body: body})
	if err != nil {
		return nil, fmt.Errorf("failed encoding message: %w", err)
	}

	req, err := http.NewRequest("POST", to.Addr+"/api/handle", &buff)
	if err != nil {
		return nil, fmt.Errorf("failed creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", signature(from, n.Key))

	var h http.Client
	res, err := h.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed doing request: %w", err)
	}

	var r HttpMessageReply
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return nil, fmt.Errorf("failed at decoding response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", res.StatusCode, r.Error)
	}

	if r.Error != "" {
		return nil, fmt.Errorf("reply error: %s", r.Error)
	}

	return r.Body, nil
}
