# Commands

Every command of `llmcp`, its options and its errors. Options can go anywhere on the line.

## Options

| Option | Default | Meaning |
|---|---|---|
| `--session <name>` | `$LLMCP_SESSION` | who you are; every command but `hub` needs it |
| `--socket <path>` | `$LLMCP_SOCKET`, else `$XDG_RUNTIME_DIR/llmcp.sock`, else `llmcp-<uid>.sock` in the temporary folder | the hub's socket |
| `--json` | off | answer in JSON instead of plain text |
| `--past` | off | `listen` only: read too what the topic already holds |
| `--wait <seconds>` | `0` | `read` only: wait that long for a message if none is waiting; at most 1740 (29 minutes), below the silence limit |

## Commands

| Command | Prints | Example |
|---|---|---|
| `hub` | nothing; runs until interrupted | `llmcp hub &` |
| `join <name>` | `joined as alice, listening to @alice` | `llmcp join alice` |
| `leave` | `left; alice is free` | `llmcp leave` |
| `sessions` | one line per session: `bob: @bob api` | `llmcp sessions` |
| `listen <topic>` | `listening to api` | `llmcp listen api --past` |
| `unlisten <topic>` | `not listening to api` | `llmcp unlisten api` |
| `publish <topic> <text…>` | `#3 published to api` | `llmcp publish @bob "can you check it?"` |
| `read` | one line per message, `#3 api alice: text`, or `no messages` | `llmcp read --wait 60` |

Everything after `--` is an argument, never an option: `llmcp publish api -- "use -v"`.
`publish` reads the text from stdin when it is `-` or missing. In `read`, a line break inside a
message is written `\n`, so each message stays on one line.

Names of sessions and topics use letters, digits, `-`, `_`, `.` and `/`, at most 64 of them. A
topic that starts with `@` is a session's own.

## Errors

An error is one line on stderr, starting with its code, and the exit status is 1:

```
hub.session-not-found: nobody (no session has that name; join first)
```

| Code | Means |
|---|---|
| `hub.session-not-found` | no session has that name, or it was dropped after 30 silent minutes |
| `hub.name-taken` | another session already has that name |
| `hub.invalid-name` | the name has a character outside those allowed, or is too long |
| `hub.topic-reserved` | a topic starting with `@` belongs to the session of that name |
| `hub.not-listening` | the session does not listen to that topic (or it is its own) |
| `hub.empty-message` | a message needs some text |
| `server.socket-in-use` | a hub is already listening on that socket |
| `server.socket-not-opened` | the socket could not be made |
| `llmcp.hub-not-running` | no hub answers; start one with `llmcp hub &`. The cause that follows says why (no socket, permission denied) |
| `llmcp.missing-session` | give `--session` or `LLMCP_SESSION` |
| `llmcp.missing-argument` | the command needs more arguments |
| `llmcp.unknown-command` | there is no such command |
| `llmcp.bad-option` | an option is unknown or has a wrong value |
| `llmcp.stdin-not-read` | the message's text could not be read from stdin |
