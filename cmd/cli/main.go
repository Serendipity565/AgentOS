package main

import (
	"AgentOS/config"
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"AgentOS/internal/agent"
	"AgentOS/internal/transport/cli"
)

// main starts the interactive CLI entrypoint.
func main() {
	configPath := flag.String("config", "config/config.yaml", "path to config yaml")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	application, err := agent.Initialize(cfg)
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}

	activeProvider, err := cfg.LLM.ActiveProvider()
	if err != nil {
		log.Fatalf("resolve active llm provider failed: %v", err)
	}

	if err := cli.Run(ctx, application.Service, cli.Options{
		Provider: activeProvider.Provider,
		Model:    activeProvider.Model,
		Skills:   countEnabledSkills(cfg),
		MCPs:     countEnabledMCP(cfg),
	}); err != nil {
		log.Fatalf("cli interface failed: %v", err)
	}
}

func countEnabledSkills(cfg config.Config) int {
	if len(cfg.Skills) == 0 {
		return 0
	}

	count := 0
	for _, skill := range cfg.Skills {
		if skill.Enabled {
			count++
		}
	}
	return count
}

func countEnabledMCP(cfg config.Config) int {
	if len(cfg.MCP) == 0 {
		return 0
	}

	count := 0
	for _, server := range cfg.MCP {
		if server.Enabled {
			count++
		}
	}
	return count
}
