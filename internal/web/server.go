package web

import "context"

type Server interface {
    Start(ctx context.Context) error
    Stop() error
}

type NoopServer struct{}

func (n *NoopServer) Start(ctx context.Context) error { return nil }
func (n *NoopServer) Stop() error { return nil }
