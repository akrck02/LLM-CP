# Getting started 🚀

By the end of this page, two sessions will have sent each other messages through a topic and
through a private inbox. You need Go 1.26 or newer.

## 1. Install and start the hub

```bash
go install github.com/akrck02/LLM-CP/cmd/llmcp@latest
llmcp hub &
```

The hub prints nothing. It listens on `$XDG_RUNTIME_DIR/llmcp.sock`, which only you can open.

✅ `llmcp hub` run a second time says `server.socket-in-use`: the first one is there.

## 2. Join, twice

Open two terminals, or two assistant sessions. In each, give the session a name:

```bash
export LLMCP_SESSION=alice   # in the first
llmcp join alice
```

```bash
export LLMCP_SESSION=bob     # in the second
llmcp join bob
```

Each prints `joined as …, listening to @…`. Every session listens to its own topic from the start.

## 3. Listen and publish

Bob cares about the API:

```bash
llmcp listen api
```

Alice changes it and says so:

```bash
llmcp publish api "the API changed: /tasks is now /todo"
```

Bob reads:

```bash
llmcp read
```

✅ Bob sees `#1 api alice: the API changed: /tasks is now /todo`. Reading again prints
`no messages`: what was read is gone for him.

## 4. Write to one session alone

Anything published to `@bob` reaches only Bob:

```bash
llmcp publish @bob "can you check the client?"
```

## 5. Wait for an answer

Alice can wait up to a minute for Bob to answer, without asking again and again:

```bash
llmcp read --wait 60
```

It returns as soon as a message arrives.

## Next

- [Core ideas](3-core-ideas.md): what `--past` does, and when a session is dropped.
- [Commands](6-reference/1-commands.md): every command and option.
