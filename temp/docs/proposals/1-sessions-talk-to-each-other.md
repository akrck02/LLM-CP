# 1. Sessions talk to each other

> Status: done, pending the author's review · Author's approval: proposal ☑ plan ☑ ("aprobado, implementa")

## Why

Several Claude Code sessions working on the same machine cannot talk to each other today: each one
works blind, and the person carries messages between them by hand ("tell the other session that I
changed the API"). A small local protocol, where sessions publish messages to topics and listen to the topics they care about, would let them coordinate on their own,
without a cloud service and without anything leaving the machine.

## What

A Go program, `llmcp`, with two parts: a **hub** that runs in the background, listens on a Unix
socket and speaks gRPC, and a **command line** (`llmcp join`, `llmcp listen`, `llmcp publish`,
`llmcp read`…) that talks to the hub and answers on stdout. A session uses it through its shell:
each call costs a few tokens, and a person can run the same commands by hand.

| Id | Requirement | Check |
|---|---|---|
| R1 | A session joins the hub with a name and keeps it until it leaves, or until it has been silent too long; a name is unique while it is taken | test `a second session cannot take a name already in use` |
| R2 | A session can list the sessions joined right now and the topics each one listens to | test `a session sees the others and their topics` |
| R3 | A session can listen to one or several topics, and stop listening to any of them | test `a session listens to several topics at once` |
| R4 | A message is published to a topic and reaches every session listening to it, except its sender | test `a message reaches everyone listening to its topic and nobody else` |
| R5 | Every session listens to a topic with its own name, so a message can be written to one session alone | test `a message to a session's own topic reaches only that session` |
| R6 | A session chooses how it listens to a topic: only the messages published from now on, or also those the topic already holds | test `a session that listens to the past reads what was published before it came` |
| R7 | A message waits for each listener until that listener reads it, in order; none is lost while the hub runs | test `messages wait in order until they are read` |
| R8 | Every command answers in short plain text by default, and in JSON with `--json` | test `read prints one line per message` |
| R9 | Only the user who started the hub can connect to it | test `the socket is readable only by its owner` |
| R10 | Every error carries a stable code (`hub.session-not-found`), declared with a small `failure` package as in go-engine, and the command line prints it | test `each error code has its own name` |
| R11 | A skill, `sessions`, in `llm-guidelines/skills/`, teaches a session when and how to use the `llmcp` commands: join with a name, listen to the topics of its work, read its messages, ask others before touching shared work. `install.sh` installs it like the others, so Codex and Gemini CLI get it too | by hand: two sessions with the skill coordinate without being told how |

## Out of scope

- Talking across machines: the socket is local by design.
- Keeping messages after the hub stops (no disk, no database).
- Locks, task queues or shared state: messages first; coordination patterns can be built on them later.
- An MCP server: the command line is cheaper in tokens and easier to adapt. One can be added later on top of the same hub.

## Design

One Go module, `github.com/akrck02/LLM-CP`, one program:

| Piece | Where | Job |
|---|---|---|
| `failure` | `internal/failure/` | Error codes, copied from go-engine's `failure` (importing go-engine would bring OpenGL along) |
| `hub` | `hub/` | The rules, in memory, with no gRPC: sessions, topics, what each listener has read |
| `protocol` | `protocol/` | `hub.proto`, the contract, and the Go code generated from it |
| `server` | `server/` | Serves the hub over gRPC on a Unix socket only its owner can open |
| `llmcp` | `cmd/llmcp/` | The command line: one function per command, plain text or `--json` |

- **Topics keep every message** while the hub runs, numbered in publish order. Each listener has a
  cursor: `listen` starts it at the end (only new) or, with `--past`, at the start. `read` returns
  what is past each cursor, oldest first, and moves the cursors.
- **A session's own topic** is `@` and its name (`@alice`). Anyone can publish there; only that
  session listens to it, so it is a private inbox (R5).
- **The command line has no connection to keep**, so a session is named in every call
  (`--session` or `LLMCP_SESSION`), and any call keeps it alive. One silent for 30 minutes is
  dropped, checked on each request rather than with a timer.
- **`read --wait 60`** waits up to that many seconds for a message, so a session can wait for an
  answer without polling and spending tokens.
- **Errors** travel in the gRPC status: the code's name in its details (`ErrorInfo.Reason`), and
  `code: details` as its message for a person. The command line prints that line on stderr and
  exits with 1.
- The socket is `$XDG_RUNTIME_DIR/llmcp.sock` (or `llmcp-<uid>.sock` in the temporary folder),
  changed with `--socket` or `LLMCP_SOCKET`, and set to mode 0600 (R9).

Not taken: an MCP server (more tokens per call, see Out of scope); gRPC with a hand-written JSON
codec instead of protobuf (no `protoc` needed, but no `.proto` for clients in other languages);
starting the hub on the first command (hidden processes are hard to debug; `llmcp hub` is one line).

## Tasks

- [x] T1 — error codes · `internal/failure/`, `hub/errors.go` · R10 · test: `TestEachErrorCodeHasItsOwnName`
- [x] T2 — sessions and topics in memory · `hub/` · R1–R7 · tests: one per requirement
- [x] T3 — the contract and its generated code · `protocol/` · R1–R7
- [x] T4 — the gRPC server on the socket · `server/` · R9, R10 · test: `TestTheSocketIsReadableOnlyByItsOwner`
- [x] T5 — the command line · `cmd/llmcp/` · R8 · test: `TestReadPrintsOneLinePerMessage`
- [x] T6 — README and docs · `README.md`, `docs/`
- [x] T7 — the `sessions` skill · `llm-guidelines/skills/sessions/` · R11 · by hand

## Result

Built: the hub, the protocol, the server, the command line, the docs (overview, getting started,
core ideas, commands, contributing), CI, docs publishing, and the `sessions` skill in llm-guidelines.

- R1–R10 are checked by tests (`go test -race ./...`), and the commands were tried end to end
  against a real hub: join, listen, publish, read, `read --wait`, a socket in use, a missing hub.
- R11 is not checked yet: it needs two real assistant sessions with the skill installed.
- The `reviewer` subagent found sixteen issues; all were fixed except two, left as TODOs:
  `protocol/generate.sh` does not check protoc's checksum, and `internal/failure` is a copy of
  go-engine's instead of a shared module.
- The tests were written alongside the code, not seen failing first.
- A person has to review all of it line by line.
