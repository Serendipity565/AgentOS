package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"AgentOS/config"
	"AgentOS/internal/agent"
	transporthttp "AgentOS/internal/transport/http"
)

// main starts the long-running web service process.
func main() {
	configPath := flag.String("config", "config/config.yaml", "path to config yaml")
	addr := flag.String("addr", "", "web server listen address, overrides config when set")
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

	webAddr := cfg.Transport.Web.Addr
	if *addr != "" {
		webAddr = *addr
	}
	if webAddr == "" {
		webAddr = ":8080"
	}

	if err := transporthttp.Run(ctx, webAddr, application.Service); err != nil {
		log.Fatalf("web interface failed: %v", err)
	}
}
