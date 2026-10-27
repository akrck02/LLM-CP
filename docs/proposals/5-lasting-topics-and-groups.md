# 5. Topics and groups that last

> Status: built, pending review · Author's approval: proposal ☑ plan ☑

## Why

Some topics are places, not moments: a team's group, the topic a steward reads, a metrics
channel. Today a topic exists only while someone listens to it or a message sits in it, and the
hub keeps everything in memory. When the hub restarts, groups lose their members and theme,
topics lose their kind, and a topic nobody listens to vanishes from the chat. Whoever set them up
must set them up again.

## What

| Id | Requirement | Check |
|---|---|---|
| R1 | A **declared** topic (`llmcp topic declare <topic> conversation|metrics`) and every **group** last: they stay in the hub's state, with their kind, members and theme, even when nobody listens and they have no messages | test `a declared topic is listed with nobody listening` |
| R2 | They are saved to a file and come back when the hub restarts, archived or not | test `a group and a declared topic come back after a restart` |
| R3 | A topic that only appeared because someone published or listened to it behaves as today, and does not come back | test `an undeclared topic is not kept` |
| R4 | `llmcp topic remove` and `llmcp group delete` take them out of the file too | test `a removed topic is not brought back` |
| R5 | The chat lists every declared topic and group, even with no messages | by hand: restart the hub and look |

## Out of scope

- Keeping **messages** across a restart: only the declaration lasts. (Metrics history is already
  kept by llm-center.)
- Sessions and their inboxes: they still end when they leave or go silent.

## Design

### The file

`hub/keep.go`: the hub takes a path (`llmcp hub --keep <file>`, default
`~/.local/share/llmcp/kept.json`; empty keeps nothing, as the tests do). It holds the declared
topics with their kind and whether they are archived, and the groups with their theme and
members:

```json
{"topics": [{"topic": "tests.results", "kind": "metrics"}],
 "groups": [{"name": "all", "theme": "everyone", "members": ["alice", "bob"]}]}
```

- Read once in `New`; a missing file is an empty start; an unreadable one is
  `hub.keep-unreadable` and the hub does not start, rather than lose the declarations silently.
- Written after every change to it (declare, remove, group create/change/remove, archive),
  through a temporary file and a rename, so a crash never leaves half a file. Few changes, small
  file: no batching.

### The state

`State.Kinds` already lists declared topics with or without listeners; a declared topic now
stays in it. The chat makes an empty chat for each declared topic and group it does not know yet,
so they show in the list with "No messages yet".

### Alternatives not taken

- **Keeping messages too**: the author chose declarations only; metrics history is llm-center's.
- **A database (SQLite)**: one small JSON file is enough and needs no dependency.

## Tasks

- [x] T1 — The file: read at start, written on each change · `hub/keep.go`, `hub/errors.go` · R2, R4 · tests `a group and a declared topic come back after a restart`, `a removed topic is not brought back`, `an undeclared topic is not kept`
- [x] T2 — Declared topics stay with nobody listening · `hub/topic.go` · R1 · test `a declared topic is listed with nobody listening`
- [x] T3 — `llmcp hub --keep`, the service's default · `cmd/llmcp/` · R2
- [x] T4 — The chat lists declared topics and groups with no messages · `chat/` · R5 · test on the list
- [x] T5 — Docs: reference of `--keep`, a line in the core ideas · `docs/`

## Result

Built as planned; the hub, chat and command line tests pass, including the five named in the
tasks. R5 was checked by hand after restarting the hub on the machine. Pending human review.
