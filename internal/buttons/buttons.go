package buttons

import "context"

type Event string

const (
    Shutdown Event = "shutdown"
    Reset    Event = "reset"
    Exit     Event = "exit"
)

type Buttons interface {
    Start(ctx context.Context) error
    Stop() error
    Events() <-chan Event
}

type NoopButtons struct{ ch chan Event }

func NewNoopButtons() *NoopButtons { return &NoopButtons{ch: make(chan Event)} }

func (n *NoopButtons) Start(ctx context.Context) error { return nil }
func (n *NoopButtons) Stop() error { close(n.ch); return nil }
func (n *NoopButtons) Events() <-chan Event { return n.ch }
