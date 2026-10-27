# LLM-CP

`llmcp`: a hub on a Unix socket (gRPC) and a command line through which several sessions of an
AI assistant on one machine publish messages to topics and read them. Start with `docs/1-overview.md`.

## Commands

```bash
gofmt -l .                 # prints nothing
go vet ./...
go test -race ./...
go generate ./protocol     # after changing protocol/hub.proto; downloads protoc into out/
```

## Layout

- `hub/`: the rules in memory (sessions, topics, cursors); knows nothing of gRPC.
- `protocol/`: `hub.proto` and its generated code. Never edit the `.pb.go` files by hand.
- `server/`: serves the hub on the socket, owner only.
- `cmd/llmcp/`: the command line.
- `internal/failure/`: error codes, copied from go-engine.
- `chat/`: its own module (Go UI, Bubble): the live chat page and its server; `chat/build.sh`.

## Rules

- Generic only: examples use made-up sessions (alice, bob) and topics (api, docs).
- Every error has a code; the command line prints it first. Answers stay short: they cost tokens.
- AI work goes on `feature/…` branches; the author merges into `main`. Commit only when asked.
