# 4. Topics of a kind: conversations and metrics

> Status: building · Author's approval: proposal ☑ plan ☐

## Why

Sessions specialise: one measures, another plans, another writes. What they measure (test
times, coverage, a build's size, how long a task took) travels today as sentences in chats,
where no program can read it back. A topic that declares it carries **metrics** gives
structured data its own channel, between sessions and between other systems, and lets
llm-center draw it.

## What

| Id | Requirement | Check |
|---|---|---|
| R1 | Every topic has a kind: `conversation` (the default, as today) or `metrics` | test `a topic nobody declared is a conversation` |
| R2 | A session declares a topic's kind once and it stays for the topic's life; declaring another kind is refused with `hub.kind-taken` | test `a topic keeps the kind it was declared with` |
| R2b | `llmcp topic remove <topic>` deletes a topic, its messages and its kind, so it can be made again as another kind | test `a removed topic can be declared again` |
| R3 | A metrics topic takes only points: a name, a number, and optionally a unit, labels and a time; anything else is refused with `hub.not-a-point` | test `a metrics topic refuses a sentence` |
| R4 | `llmcp metric <topic> <name> <value> [--unit u] [--label k=v]` publishes a point; `llmcp read` gives points back as JSON | test `a point published is read back whole` |
| R5 | A metrics topic keeps the points of the last 7 days; older ones are dropped | test `a metrics topic forgets points older than a week` |
| R6 | `llmcp sessions` and the chat's state say each topic's kind; the chat lists metrics topics apart and shows their latest values, not as messages | test on the state; by hand in the chat |
| R8 | Every session publishes, through a hook, how long each turn took and the tokens it spent, on the topic `sessions.metrics`, labelled with its name; the sessions skill tells them to publish their test results there too | test of the hook on a sample transcript; by hand |
| R7 | Another program on the machine can publish and read points without being a Claude session, through the same CLI or gRPC | by hand: a shell loop publishing the load average |

## Out of scope

- Charts: llm-center draws them (its proposal 2). The chat only shows the latest values.
- Keeping points across a restart of the hub: the hub stays in memory; llm-center keeps the
  history it draws.
- Other kinds (files, events, tasks). The kind is a field, so they can come later.
- Reaching the hub from other machines.

## Design

### In the hub

- `Topic` gains a `kind`: `conversation` (the zero value, so every topic today stays one) or
  `metrics`. It lives in `hub/topic.go` beside the messages, so it is lost with them.
- `Declare(by, topic, kind)`: makes the topic with that kind, or answers fine if it already has
  it; another kind is `hub.kind-taken`. Publishing to a topic nobody declared makes it a
  conversation, as today.
- `RemoveTopic(by, topic)`: deletes the messages, the kind and every cursor on it.
- `Publish` to a metrics topic parses the text as a point (`hub/point.go`): `name` (letters,
  digits, `.` `_` `-`), `value` (a finite number), `unit`, `labels` (string to string), `at`
  (Unix milliseconds; the hub's clock if missing). Anything else is `hub.not-a-point`. The text
  is stored as the hub rewrites it, with `at` filled in, so every reader sees the same time.
- Points older than 7 days are dropped when a new one arrives on the topic: no timer, nothing to
  stop.

### The protocol

`Declare(TopicKindRequest)` and `RemoveTopic(TopicRequest)` are added; `State` gains
`repeated TopicKind kinds` (topic, kind, publishers), so the chat and llm-center know which
topics are metrics and who publishes them without asking. `Message` does not change.

### The command line

- `llmcp topic declare <topic> metrics`, `llmcp topic remove <topic>`.
- `llmcp metric <topic> <name> <value> [--unit u] [--label k=v]…`: declares the topic if it
  must, builds the point and publishes it.

### The hook

`hooks/metrics.py` in llm-guidelines, on `Stop`: from the transcript Claude Code passes, it adds
up the turn's tokens and its time since the turn's prompt, and runs `llmcp metric
sessions.metrics turn.seconds …` and `turn.tokens …` with `--label session=<name>`. Like the
typing hook, it never blocks a turn and does nothing for a session that never joined.

### The chat

Metrics topics leave the Topics tab for a small **Metrics** list under it: each topic with its
latest value per series. Charts are llm-center's.

### Alternatives not taken

- **A new message type in the protocol for points**: cleaner on the wire, but every client
  would change at once. JSON in the text keeps old clients reading.
- **Keeping points by count**: the author chose time; a week is what llm-center draws.

## Tasks

- [x] T1 — Topic kinds, declare and remove · `hub/topic.go`, `hub/errors.go` · R1, R2, R2b · tests `a topic nobody declared is a conversation`, `a topic keeps the kind it was declared with`, `a removed topic can be declared again`
- [x] T2 — Points: parse, check, fill the time, drop after 7 days · `hub/point.go` · R3, R5 · tests `a metrics topic refuses a sentence`, `a metrics topic forgets points older than a week`
- [x] T3 — Protocol and server: Declare, RemoveTopic, kinds in State · `protocol/`, `server/` · R6 · test on the state
- [x] T4 — Command line: `topic declare|remove`, `metric` · `cmd/llmcp/` · R4, R7 · test `a point published is read back whole`
- [x] T5 — The chat: metrics topics apart, latest values · `chat/` · R6 · screenshot
- [x] T6 — The hook and the skill's rule · llm-guidelines `hooks/metrics.py`, `skills/sessions/SKILL.md` · R8 · test on a sample transcript
- [x] T7 — Docs: the reference of the new commands, a guide "Publishing metrics" · `docs/` · R4, R7

## Result

_Pending._
