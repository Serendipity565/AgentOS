# AgentOS

一个按长期可扩展方向整理过的 Go Agent 项目骨架。

英文说明见 [README.en.md](/Users/gaofengjiang/Documents/Code/Go/AgentOS/README.en.md)。

## 目录结构

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

## 分层说明

- `cmd/*`：进程入口，只负责加载配置、初始化运行时并启动 transport。
- `internal/transport`：CLI / HTTP 传输适配层。
- `internal/agent`：Agent 编排、系统指令和运行时装配。
- `internal/llm`：模型提供商封装。
- `internal/memory`：记忆系统抽象。
- `internal/tools`：skill / MCP 能力。
- `internal/domain`：后续承载核心领域类型。
- `pkg/schema`：跨层共享的数据结构。

依赖方向：

```text
cmd
  ↓
transport
  ↓
agent
  ↓
llm / memory / tools
```

## 配置文件

默认配置文件是 [config/config.yaml](/Users/gaofengjiang/Documents/Code/Go/AgentOS/config/config.yaml)。

`llm` 采用多 provider 配置：

- `active`：当前启用的模型配置名
- `providers[]`：可切换的厂商或模型实例
- `provider`：厂商标识，比如 `deepseek`、`openai`
- `driver`：底层协议适配器，当前内置 `mock` 和 `openai-compatible`

这意味着 DeepSeek、OpenAI、Moonshot 这类兼容 OpenAI Chat Completions 的接口都可以共用一套接入层。

## 运行方式

启动 Web Agent：

```bash
go run ./cmd/web -config config/config.yaml
```

启动终端 CLI：

```bash
go run ./cmd/cli -config config/config.yaml
```

也可以直接用：

```bash
make run-web
make run-cli
```
