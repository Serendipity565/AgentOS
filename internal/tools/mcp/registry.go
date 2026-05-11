package mcp

import (
	"context"
	"fmt"
	"sort"
)

type Request struct {
	Input string `json:"input"`
}

type Result struct {
	Output string `json:"output"`
}

type Server interface {
	Name() string
	Description() string
	Call(ctx context.Context, req Request) (Result, error)
}

type Registry struct {
	servers map[string]Server
}

func NewRegistry() *Registry {
	return &Registry{servers: make(map[string]Server)}
}

func (r *Registry) Register(server Server) {
	r.servers[server.Name()] = server
}

func (r *Registry) Call(ctx context.Context, name string, req Request) (Result, error) {
	server, ok := r.servers[name]
	if !ok {
		return Result{}, fmt.Errorf("mcp server %q not found", name)
	}
	return server.Call(ctx, req)
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.servers))
	for name := range r.servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *Registry) Descriptions() []string {
	names := r.Names()
	descs := make([]string, 0, len(names))
	for _, name := range names {
		descs = append(descs, fmt.Sprintf("%s(%s)", name, r.servers[name].Description()))
	}
	return descs
}
