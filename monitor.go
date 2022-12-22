package p2p

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type MonitorSubscribeFunc func(ctx context.Context, addr string, out chan<- *State) error

type Monitor struct {
	ctx       context.Context
	addr      string
	state     *State
	err       error
	updates   chan *State
	subscribe MonitorSubscribeFunc
}

func NewMonitor(addr string, subscribe MonitorSubscribeFunc) *Monitor {
	return &Monitor{
		ctx:       context.Background(),
		addr:      addr,
		updates:   make(chan *State, 10),
		subscribe: subscribe,
	}
}

func (m *Monitor) SetContext(ctx context.Context) {
	m.ctx = ctx
}

func (m *Monitor) Error() error {
	return m.err
}

func (m *Monitor) Init() tea.Cmd {
	return tea.Batch(m.start, m.listen, m.refresh)
}

func (m *Monitor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *State:
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
	if m.state == nil {
		return ""
	}

	var sb strings.Builder

	wr := tabwriter.NewWriter(&sb, 6, 2, 2, ' ', tabwriter.Debug)

	fmt.Fprintln(wr, "Addr\tName\tSince\tCurrent")

	current := m.state.Current

	if current != nil {
		t, _ := time.Parse(time.RFC3339, current.UpdatedAt)
		since := time.Since(t).Truncate(time.Second)
		fmt.Fprintf(wr, "%s\t%s\t%s\t%s\n", current.Addr, current.Name, since, "*")
	}

	for _, peer := range m.state.Peers {
		t, _ := time.Parse(time.RFC3339, peer.UpdatedAt)
		since := time.Since(t).Truncate(time.Second)
		fmt.Fprintf(wr, "%s\t%s\t%s\t%s\n", peer.Addr, peer.Name, since, "")
	}

	wr.Flush()

	return sb.String()
}

func (m *Monitor) start() tea.Msg {
	done := make(chan error, 1)
	go func() {
		done <- m.subscribe(m.ctx, m.addr, m.updates)
	}()

	select {
	case err := <-done:
		m.err = err
	case <-m.ctx.Done():
		m.err = m.ctx.Err()
	}

	return tea.Quit()
}

func (m *Monitor) listen() tea.Msg {
	return <-m.updates
}

func (m *Monitor) refresh() tea.Msg {
	return <-time.After(time.Second)
}
