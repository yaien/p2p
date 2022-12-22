package p2p

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func Subscribe2Http(ctx context.Context, addr string, out chan<- *State) error {
	req, err := http.NewRequestWithContext(ctx, "GET", addr+"/p2p/state", nil)
	if err != nil {
		return fmt.Errorf("failed making state request: %w", err)
	}

	req.Header.Set("accept", "text/event-stream")
	req.Header.Set("connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")

	var h http.Client
	res, err := h.Do(req)
	if err != nil {
		return fmt.Errorf("failed doing sse request: %w", err)
	}

	defer res.Body.Close()

	sc := bufio.NewScanner(res.Body)
	prefix := []byte("data: ")
	for sc.Scan() {
		b := sc.Bytes()
		if !bytes.HasPrefix(b, prefix) {
			continue
		}

		b = bytes.TrimPrefix(b, prefix)
		var state State
		err = json.Unmarshal(b, &state)
		if err != nil {
			return fmt.Errorf("failed at unmarshal state update from sse: %w", err)

		}
		out <- &state
	}

	return nil
}
