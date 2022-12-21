package p2p

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Subscribe2Grpc(ctx context.Context, addr string, out chan<- *State) error {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed at connection: %w", err)
	}

	client := NewP2PClient(conn)
	sc, err := client.State(ctx, &StateRequest{})
	if err != nil {
		return fmt.Errorf("failed at subscribing to state: %w", err)
	}

	for {
		res, err := sc.Recv()
		if err != nil {
			return fmt.Errorf("failed at recv: %w", err)
		}

		out <- res.State
	}
}
