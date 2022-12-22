package p2p

import (
	"errors"
	"fmt"
	"path"
)

var ErrNoneMatchedPeers = errors.New("none peer matched the pattern")

func (p *P2P) Broadcast(pattern string, subject string, body []byte) error {

	for _, peer := range p.peers {
		matched, err := path.Match(pattern, peer.Name)
		if err != nil {
			return fmt.Errorf("failed at pattern match: %w", err)
		}

		if !matched {
			continue
		}

		p.transport.Send(p.current, peer, subject, body)

	}

	return nil
}

func (p *P2P) Request(pattern string, subj string, body []byte) ([]byte, error) {
	for _, peer := range p.peers {
		matched, err := path.Match(pattern, peer.Name)
		if err != nil {
			return nil, fmt.Errorf("failed at pattern match: %w", err)
		}

		if !matched {
			continue
		}

		return p.transport.Send(p.current, peer, subj, body)
	}

	return nil, fmt.Errorf("%w %s", ErrNoneMatchedPeers, pattern)

}
