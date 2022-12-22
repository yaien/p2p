package p2p

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GrpcTransport struct {
	clients sync.Map
}

func (n *GrpcTransport) Connect(from *Peer, addr string) (*State, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed creating client connection: %w", err)
	}

	client := NewP2PClient(conn)
	res, err := client.Connect(context.TODO(), &ConnectRequest{Current: from})
	if err != nil {
		return nil, fmt.Errorf("failed at p2p connection: %w", err)
	}

	n.clients.Store(res.State.Current.Id, client)

	return res.State, nil
}

func (n *GrpcTransport) Send(from, to *Peer, subject string, body []byte) ([]byte, error) {
	v, ok := n.clients.Load(to.Id)
	if !ok {
		return nil, fmt.Errorf("missing client for peer id %s", to.Id)
	}

	client := v.(P2PClient)
	res, err := client.Message(context.TODO(), &MessageRequest{From: from, Subject: subject, Body: body})
	if err != nil {
		return nil, fmt.Errorf("failed at sensing message: %w", err)
	}

	return res.Body, nil
}
