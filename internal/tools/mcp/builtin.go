package mcp

import (
	"context"
	"time"
)

type TimeServer struct{}

func (TimeServer) Name() string {
	return "time"
}

func (TimeServer) Description() string {
	return "returns the current server time"
}

func (TimeServer) Call(_ context.Context, _ Request) (Result, error) {
	return Result{Output: time.Now().Format(time.RFC3339)}, nil
}
