package p2p

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/tabwriter"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Monitor struct {
	ctx     context.Context
	addr    string
	state   State
	err     error
	updates chan State
}

func NewMonitor(addr string) *Monitor {
	return &Monitor{
		ctx:     context.Background(),
		addr:    addr,
		updates: make(chan State, 10),
	}
}

func (m *Monitor) SetContext(ctx context.Context) {
	m.ctx = ctx
}

func (m *Monitor) Error() error {
	return m.err
}

func (m *Monitor) Init() tea.Cmd {
	return tea.Batch(tea.Sequence(m.fetch, m.start), m.listen, m.refresh)
}

func (m *Monitor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case State:
		m.state = msg
		return m, m.listen
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case time.Time:
		return m, m.refresh
	}
	return m, nil
}

func (m *Monitor) View() string {
	var sb strings.Builder

	wr := tabwriter.NewWriter(&sb, 6, 2, 2, ' ', tabwriter.Debug)

	fmt.Fprintln(wr, "Addr\tName\tSince\tCurrent")

	curr := m.state.Current
	if curr != nil {
		since := time.Since(curr.UpdatedAt).Truncate(time.Second)
		fmt.Fprintf(wr, "%s\t%s\t%s\t%s\n", curr.Addr, curr.Name, since, "*")
	}

	for _, cl := range m.state.Clients {
		since := time.Since(cl.UpdatedAt).Truncate(time.Second)
		fmt.Fprintf(wr, "%s\t%s\t%s\t%s\n", cl.Addr, cl.Name, since, "")
	}

	wr.Flush()

	return sb.String()
}

func (m *Monitor) start() tea.Msg {
	done := make(chan error, 1)
	go func() {
		done <- m.sse()
	}()

	select {
	case err := <-done:
		m.err = err
	case <-m.ctx.Done():
		m.err = m.ctx.Err()
	}

	return tea.Quit()
}

func (m *Monitor) sse() error {
	req, err := http.NewRequestWithContext(m.ctx, "GET", m.addr+"/p2p/sse", nil)
	if err != nil {
		return fmt.Errorf("failed making sse request: %w", err)
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
			return fmt.Errorf("failed at unmarshalling state update from sse: %w", err)

		}
		m.updates <- state
	}

	return nil
}

func (m *Monitor) fetch() tea.Msg {
	res, err := http.Get(m.addr + "/api/state")
	if err != nil {
		m.err = fmt.Errorf("failed getting api state: %w", err)
		return tea.Quit()
	}

	var state State
	err = json.NewDecoder(res.Body).Decode(&state)
	if err != nil {
		m.err = fmt.Errorf("failed at decoding state: %w", err)
		return tea.Quit()
	}

	return state
}

func (m *Monitor) listen() tea.Msg {
	return <-m.updates
}

func (m *Monitor) refresh() tea.Msg {
	return <-time.After(time.Second)
}
