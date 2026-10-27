# Contributing 🛠️

This page is for whoever wants to change LLM-CP: where things are, and how to check a change.

## The layout

| Folder | What it holds |
|---|---|
| `hub/` | The rules, in memory: sessions, topics, the cursor of each listener. No gRPC here. |
| `protocol/` | `hub.proto`, the contract, and the Go code generated from it. |
| `server/` | Serves a hub over gRPC on a Unix socket only its owner can open. |
| `cmd/llmcp/` | The command line: options, commands, plain text or JSON answers. |
| `chat/` | A module of its own: the chat page (`cmd/chat-web`, Go UI in WebAssembly), its server (`cmd/llmcp-chat`, which also keeps each person's subscriptions) and `conversation`, the chats built from messages. |
| `internal/failure/` | Error codes, copied from go-engine. |
| `docs/proposals/` | Why each bigger change was made, and how. |

Dependencies point one way: `cmd/llmcp` uses `server`, which uses `hub` and `protocol`. `hub`
uses nothing of the others, so its rules are tested without any socket.

## Checking a change

```bash
gofmt -l .            # prints nothing
go vet ./...
go test -race ./...
```

CI runs the same three on every push. The chat is a module of its own: run them in `chat/` too.

## Changing the protocol

Edit `protocol/hub.proto`, then:

```bash
go generate ./protocol
```

It downloads `protoc` into `out/` (ignored by git) at the version pinned in
`protocol/generate.sh`, installs the Go plugins, and rewrites the `.pb.go` files. Commit those
with the `.proto`.

## Bigger changes

A change that spans several files starts as a proposal in `docs/proposals/`, approved before it is
built. See [the first one](proposals/1-sessions-talk-to-each-other.md).
