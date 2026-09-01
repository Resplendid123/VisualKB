package conversation

import (
	"context"
	"time"
)

const BashName = "bash"

type Spec struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// Content and Error are mutually exclusive.
type Result struct {
	Content string
	Error   string
}

type Traits struct {
	Concurrent bool // shares parallel slot with batch peers
	Timeout    time.Duration
	Message    func(args map[string]any) string // action description for UI

	PausesTurn bool
}

type Tool interface {
	Spec() Spec
	Invoke(ctx context.Context, args map[string]any) Result
	Traits() Traits
}

type ToolRegistry interface {
	Register(t Tool) error
	Get(name string) (Tool, bool)
	GetAll() []Tool
}
