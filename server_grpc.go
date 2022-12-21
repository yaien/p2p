package p2p

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GrpcServer struct {
	UnimplementedP2PServer
	p2p        *P2P
	subscriber *Subscriber
}

func NewGrpcP2PServer(p2p *P2P, s *Subscriber) *GrpcServer {
	return &GrpcServer{p2p: p2p, subscriber: s}
}

func (s *GrpcServer) Serve(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed at listen: %w", err)
	}

	srv := grpc.NewServer()
	RegisterP2PServer(srv, s)
	reflection.Register(srv)
	return srv.Serve(lis)
}

func (s *GrpcServer) Connect(ctx context.Context, r *ConnectRequest) (*ConnectResponse, error) {
	s.p2p.Save(r.Current)
	res := &ConnectResponse{State: s.p2p.State()}
	return res, nil
}

func (s *GrpcServer) State(_ *StateRequest, srv P2P_StateServer) error {
	err := srv.Send(&StateResponse{State: s.p2p.State()})
	if err != nil {
		return fmt.Errorf("failed sending state: %w", err)
	}

	subscription, unsubscribe := s.subscriber.Subscribe()
	defer unsubscribe()

	for state := range subscription {
		err := srv.Send(&StateResponse{State: state})
		if err != nil {
			break
		}
	}

	return nil

}

func (s *GrpcServer) Message(ctx context.Context, r *MessageRequest) (*MessageResponse, error) {
	body, err := s.p2p.handler.ServeP2P(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("failed p2p: %w", err)
	}

	return &MessageResponse{Body: body}, nil
}
