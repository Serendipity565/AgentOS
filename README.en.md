# AgentOS

A Go agent scaffold organized around a narrow `cmd -> transport -> agent`
dependency flow.

## Layout

```text
AgentOS/
├── cmd/
│   ├── cli/
│   │   └── main.go
│   └── web/
│       └── main.go
├── internal/
│   ├── agent/
│   │   ├── app.go
│   │   └── service.go
│   ├── config/
│   │   └── config.go
│   ├── domain/
│   ├── transport/
│   │   ├── cli/
│   │   └── http/
│   ├── llm/
│   │   ├── provider.go
│   │   ├── openai.go
│   │   └── mock.go
│   ├── memory/
│   │   └── store.go
│   └── tools/
│       ├── skill/
│       └── mcp/
├── pkg/
│   └── schema/
├── docs/
└── Makefile
```

## Dependency Direction

```text
cmd
  ↓
transport
  ↓
agent
  ↓
llm / memory / tools
```

## Run

```bash
go run ./cmd/web -config config/config.yaml
go run ./cmd/cli -config config/config.yaml
```

The `llm` section supports multiple providers through an `active + providers[]`
layout. `provider` identifies the vendor, while `driver` selects the transport
adapter. Right now `mock` and `openai-compatible` are built in, so vendors like
DeepSeek and OpenAI can coexist in one config file.
