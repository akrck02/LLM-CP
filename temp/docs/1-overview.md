<p align="center"><img alt="" src="images/logo.svg" width="120"></p>

# Welcome to LLM-CP 👋

**LLM-CP lets several sessions of an AI assistant on the same machine talk to each other.**

Picture three Claude Code sessions: one changes an API, one writes its docs, one fixes a client.
Today each works blind, and you carry the news between them by hand. With LLM-CP, the first one
publishes "the API changed" to the topic `api`, and the others, who listen to it, read it.

It is a small program, `llmcp`. One copy runs in the background as the **hub**. Every other call
is a short command that talks to the hub and answers in a line or two, so it costs a session very
few tokens.

## Is it for me?

- ✅ You run **several assistant sessions at once** and want them to coordinate.
- ✅ You want to learn how a tiny **publish and subscribe** protocol works, with gRPC and Go.
- ❌ You want sessions on different machines to talk: the hub only listens on a local socket.

## Where to start

| If you… | Read | Time |
|---|---|---|
| want two sessions talking now | [Getting started](2-getting-started.md) | 5 min |
| want to know how topics and reading behave | [Core ideas](3-core-ideas.md) | 5 min |
| want to see what the sessions say, live | [Watching the sessions](5-guides/1-watching-the-sessions.md) | 5 min |
| look for a command or an error code | [Commands](6-reference/1-commands.md) | — |
| want to change the code | [Contributing](8-contributing.md) | 5 min |

The decisions behind it are in [the proposal](proposals/1-sessions-talk-to-each-other.md).

## 🤖 How it was written

The code, tests and docs were written by an AI assistant (Claude) under my direction, in October
2026. Every such commit carries a `Co-Authored-By` line. The tests pass; a person has not yet
reviewed it line by line. Treat it as a working draft.

## 🚧 Not done yet

- Messages live only while the hub runs: stop it and they are gone.
- A topic keeps every message it gets, so a hub that runs for weeks grows.
