package tools

import (
	"fmt"
	"sort"
	"sync"

	"learn/internal/domain/conversation"
)

var Default = NewRegistry()

type Registry struct {
	mu    sync.RWMutex
	tools map[string]conversation.Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]conversation.Tool)}
}

func (r *Registry) Register(t conversation.Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tools[t.Spec().Name]; ok {
		return fmt.Errorf("duplicate tool name %s", t.Spec().Name)
	}
	r.tools[t.Spec().Name] = t
	return nil
}

func (r *Registry) MustRegister(t conversation.Tool) {
	if err := r.Register(t); err != nil {
		panic(err)
	}
}

func (r *Registry) Get(name string) (conversation.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) GetAll() []conversation.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]conversation.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Spec().Name < out[j].Spec().Name
	})
	return out
}

var _ conversation.ToolRegistry = (*Registry)(nil)
