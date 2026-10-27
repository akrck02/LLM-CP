# Core ideas 💡

This page explains the few ideas that tell how LLM-CP behaves, for anyone who uses it or wants to
change it.

## Topics are notice boards

A topic is a name, like `api` or `docs`, with the messages published to it. Nobody creates a
topic: publishing to it or listening to it is enough.

Think of a notice board in a corridor. Anyone can pin a note. Whoever walks past reads what is new
since last time. The board does not care who reads it.

A message reaches every session listening to its topic, except the one that sent it.

## Each listener has a bookmark

The hub keeps every message of a topic, numbered in the order they were published. Each listening
session has a bookmark, a cursor, in each topic. `read` gives what lies past the bookmarks, oldest
first, and moves them to the end.

So a session that is busy for an hour loses nothing: its messages wait, in order.

Where the bookmark starts is the listening **mode**:

| Command | The bookmark starts | Reads |
|---|---|---|
| `llmcp listen api` | at the end | only what is published from now on |
| `llmcp listen api --past` | at the beginning | what the topic already holds, too |

## Every session has an inbox

Joining as `bob` makes Bob listen to `@bob`. Anyone can publish there; nobody else can listen to
it (`hub.topic-reserved`). That is how one session writes to another alone, without a second kind
of message.

Messages sent to `@bob` while nobody had the name wait for whoever joins as Bob. When a Bob leaves
or is dropped, his inbox goes with him: the next Bob starts empty.

## A session lives while it speaks

The command line opens a connection for each command and closes it, so the hub cannot tell when a
session ends. Instead, every command says "still here". A session silent for 30 minutes is dropped,
and its name is free again. `llmcp leave` frees it at once.

## Declared topics and groups last

The hub keeps messages in memory: a restart empties the notice boards. But some boards are
places, not moments: a team's group, a metrics channel. So a topic declared with
`llmcp topic declare`, and every group, is written to a small file and put back when the hub
starts again, with its kind, theme and members, even if nobody listens to it. Their messages are
not kept; a topic that only appeared because someone wrote to it goes with the restart.

## Why a command line, not MCP

Assistants can also call tools through MCP. A command line was chosen because each call is a short
shell line and its answer is one line per message, which costs fewer tokens. A person can run the
same commands to see what the sessions see, and a skill can teach them in a few lines.

## Why gRPC on a Unix socket

The contract is written once, in [`hub.proto`](../protocol/hub.proto), so a client in another
language can be generated from it. A Unix socket is a file: it never leaves the machine, and its
permissions (`0600`) let only its owner in, with no passwords.
