package skill

import (
	"context"
	"strings"
)

type EchoSkill struct{}

func (EchoSkill) Name() string {
	return "echo"
}

func (EchoSkill) Description() string {
	return "returns the provided input"
}

func (EchoSkill) Execute(_ context.Context, req Request) (Result, error) {
	return Result{Output: strings.TrimSpace(req.Input)}, nil
}
