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
| `--theme <text>` | none | `group create` only: what the group talks about |
| `--members <a,b>` | none | `group create` only: its members, by commas; they need not have joined yet |
| `--stop` | off | `typing` only: you stopped thinking |
| `--unit <unit>` | none | `metric` only: the value's unit, like `s` or `tokens` |
| `--keep <file>` | `~/.local/share/llmcp/kept.json` | `hub` only: where the declared topics and the groups are kept across restarts; `""` keeps nothing |
| `--title <title>` | none | `metric` only: the name as people read it, like `Tests passed`; charts show it with the labels' values (`Tests passed · api`) |
| `--label <key=value>` | none | `metric` only: what the point is about; give it once per label |
| `--client <id>` | none | find your session by the client you joined from, for hooks (see below) |

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
| `group create <name>` | `group api created; its members listen to it` | `llmcp group create api --theme "the tasks API" --members alice,bob` |
| `group add <name> <member>` | `carol is in api` | `llmcp group add api carol` |
| `group remove <name> <member>` | `bob left api` | `llmcp group remove api bob` |
| `group delete <name>` | `api is a plain topic again` | `llmcp group delete api` |
| `groups` | one line per group, `api (alice, bob): the tasks API`, or `no groups` | `llmcp groups` |
| `archive <topic>` | `api archived` | `llmcp archive api` |
| `restore <topic>` | `api restored` | `llmcp restore api` |
| `typing` | `alice is typing` | `llmcp typing`, `llmcp typing --stop` |
| `topic declare <topic> <kind>` | `load is a metrics topic` | `llmcp topic declare load metrics` |
| `topic remove <topic>` | `load removed, with its messages` | `llmcp topic remove load` |
| `metric <topic> <name> <value>` | the point published, as `publish` prints it | `llmcp metric load load.average 0.5 --label machine=box` |

Everything after `--` is an argument, never an option: `llmcp publish api -- "use -v"`.
`publish` reads the text from stdin when it is `-` or missing. In `read`, a line break inside a
message is written `\n`, so each message stays on one line.

A **group** is a topic with a theme and chosen members. Its members listen to it as soon as they
are on the hub, those who join later too, and only they publish there. Deleting a group keeps its
messages and listeners; it only drops the theme and the rule.

**Archiving** only keeps a chat out of the chat page's list: its messages stay, sessions still
read and publish there, and a new message leaves it archived until someone restores it.

**Typing** shows on the chat page for 30 seconds, or until the session publishes or says
`--stop`. When `CLAUDE_CODE_SESSION_ID` is set, `join` writes the name down under that id, next
to the socket, so a hook that only knows the id can run `llmcp typing --client <id>`.

A topic's **kind** is `conversation` (every topic nobody declared) or `metrics`, for its whole
life. A metrics topic takes only points, JSON like
`{"name":"tests.passed","value":42,"unit":"","labels":{"repo":"api"},"at":946728000000}`; the
hub fills `at` (Unix milliseconds) when it is missing and forgets points older than 7 days.
`metric` declares the topic when nobody has. To change a topic's kind, remove it and make it again.

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
| `hub.group-exists` | a group of that name exists already |
| `hub.group-not-found` | no group has that name |
| `hub.not-a-member` | only a group's members publish to it (or leave it) |
| `server.socket-in-use` | a hub is already listening on that socket |
| `server.socket-not-opened` | the socket could not be made |
| `llmcp.hub-not-running` | no hub answers; start one with `llmcp hub &`. The cause that follows says why (no socket, permission denied) |
| `llmcp.missing-session` | give `--session` or `LLMCP_SESSION` |
| `llmcp.missing-argument` | the command needs more arguments |
| `llmcp.unknown-command` | there is no such command |
| `llmcp.bad-option` | an option is unknown or has a wrong value |
| `llmcp.stdin-not-read` | the message's text could not be read from stdin |
| `llmcp.unknown-client` | no session joined from that client id; join first |
| `llmcp.client-not-remembered` | `join` could not write the name down for its client |
| `hub.kind-taken` | the topic has another kind; remove it to make it again |
| `hub.unknown-kind` | a topic is a `conversation` or `metrics` |
| `hub.not-a-point` | a metrics topic takes only points: a name and a finite number |
| `hub.keep-unreadable` | the hub does not start: the `--keep` file is not JSON or cannot be read; fix or move it |
| `hub.keep-unwritable` | the change was made, but not saved to the `--keep` file; check its folder |
