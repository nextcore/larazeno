package engine

import (
	"context"
	"sync"
)

type Scope struct {
	mu     sync.RWMutex
	vars   map[string]interface{}
	parent *Scope
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		vars:   make(map[string]interface{}),
		parent: parent,
	}
}

func (s *Scope) Set(key string, val interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vars[key] = val
}

func (s *Scope) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	val, ok := s.vars[key]
	s.mu.RUnlock()

	if ok {
		return val, true
	}
	if s.parent != nil {
		return s.parent.Get(key)
	}
	return nil, false
}

func (s *Scope) All() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make(map[string]interface{})
	if s.parent != nil {
		for k, v := range s.parent.All() {
			res[k] = v
		}
	}
	for k, v := range s.vars {
		res[k] = v
	}
	return res
}

type Node struct {
	Name     string
	Value    interface{}
	Children []*Node
	Parent   *Node
	Line     int
	Col      int
}

type HandlerFunc func(ctx context.Context, node *Node, scope *Scope) error

type Engine struct {
	mu       sync.RWMutex
	Registry map[string]HandlerFunc
}

func NewEngine() *Engine {
	return &Engine{
		Registry: make(map[string]HandlerFunc),
	}
}

func (e *Engine) Register(name string, fn HandlerFunc) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Registry[name] = fn
}

func (e *Engine) Execute(ctx context.Context, node *Node, scope *Scope) error {
	if node == nil {
		return nil
	}

	e.mu.RLock()
	fn, exists := e.Registry[node.Name]
	e.mu.RUnlock()

	if exists {
		return fn(ctx, node, scope)
	}

	// Default fallback: execute all children sequentially
	for _, child := range node.Children {
		if err := e.Execute(ctx, child, scope); err != nil {
			return err
		}
	}
	return nil
}
