package skill

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

type Skill interface {
	Name() string
	Description() string
	Execute(ctx context.Context, req Request) (Result, error)
}

type Registry struct {
	skills map[string]Skill
}

func NewRegistry() *Registry {
	return &Registry{skills: make(map[string]Skill)}
}

func (r *Registry) Register(s Skill) {
	r.skills[s.Name()] = s
}

func (r *Registry) Execute(ctx context.Context, name string, req Request) (Result, error) {
	s, ok := r.skills[name]
	if !ok {
		return Result{}, fmt.Errorf("skill %q not found", name)
	}
	return s.Execute(ctx, req)
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.skills))
	for name := range r.skills {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *Registry) Descriptions() []string {
	names := r.Names()
	descs := make([]string, 0, len(names))
	for _, name := range names {
		descs = append(descs, fmt.Sprintf("%s(%s)", name, r.skills[name].Description()))
	}
	return descs
}
