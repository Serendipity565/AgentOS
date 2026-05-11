# Architecture

`AgentOS` follows the structure below:

```text
cmd
  ↓
internal/transport
  ↓
internal/agent
  ↓
internal/llm
internal/memory
internal/tools
```

- `cmd/cli` and `cmd/web` are process entrypoints.
- `internal/transport/cli` and `internal/transport/http` adapt external protocols.
- `internal/agent` owns orchestration, system commands, runtime prompt construction, and app assembly.
- `internal/llm`, `internal/memory`, and `internal/tools` provide runtime capabilities.
- `internal/domain` is reserved for core domain types as the project grows.
- `pkg/schema` contains shared request and response shapes.
