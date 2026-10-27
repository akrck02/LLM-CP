# 7. One conversation per pair

> Status: approved, to build · Author's approval: proposal ☑ plan ☑ (through the organizer, 2026-10-27)

## Why

When `api` writes to the person, it publishes to `@alice`; when the person answers, the
chat publishes to `@api`. One conversation lives in two inboxes, so neither side ever reads
it as one: the chat shows two half-chats, and a session reading its inbox sees only the other
half.

A messaging app keeps one conversation per pair of people, and both read and write in it. LLM-CP
already has the piece for that: a **group** is a topic with members, and only they publish there.
A conversation between two is a group of two.

## What

| Id | Requirement | Check |
|---|---|---|
| R1 | The first message from one session to another (`publish @api` from `alice`) creates their **pair**: a group named `alice+api` (the two names sorted), whose members are those two | hub test `the first message to someone creates the pair` |
| R2 | Every message from one to the other, written to `@name` or to the pair's name, lands in the pair; both read it there, and nobody else may publish there | hub tests `a message to @name lands in the pair`, `a third session cannot publish in a pair` |
| R3 | A pair lasts across hub restarts, like every group (proposal 5); its messages do not | hub test `a pair is kept`, as for groups |
| R4 | `llmcp groups` lists pairs apart from the other groups, and `read` shows a pair's messages as `#12 alice+api api: text` | CLI print tests |
| R5 | In the chat, each pair the person is in is one conversation named after the other member, holding both sides; writing there publishes to the pair | conversation test `a pair is one conversation named after the other`; by hand in the page |
| R6 | Pairs between two sessions show in the chat as read-only conversations, under "Between sessions" | by hand |
| R7 | The sessions skill tells sessions that `@name` reaches the pair, and that a message on a pair is answered there | by hand: two sessions talk following the skill |

## Out of scope

- Moving messages already sent to `@inboxes` into pairs: see "Old messages" below.
- Conversations of three or more: those are groups, as today.
- Changing how topics and groups the person chose are listed (proposal 6).

## Design

### The name

A pair's name is the two session names, sorted, joined by `+`: `alice+api`. Session names
cannot hold a `+` today (letters, digits, `-`, `_`, `.`, `/`), so the `+` is unambiguous: the hub
allows it in topic names only as a pair's separator, and only with a valid name on each side. Sorting
makes it the same name whoever writes first. A pair can be up to 129 characters, two names and the
`+`.

### Who creates it: the hub

The hub is the only place the chat and the command line both pass through, so it creates pairs.
When `A` publishes to `@B` (with `A` and `B` different):

1. If the group `A+B` does not exist, the hub creates it: members `A` and `B`, kind **pair**, kept
   in the keep file like any group. Both listen to it at once if they are on the hub, and when they
   join later, as members of any group do.
2. The message is added to `A+B`, not to `@B`. `publish` prints `#12 published to A+B`.

Writing to the pair's name directly (`publish A+B`) is the same, once it exists. A session
publishing to its own `@A` still writes to its own inbox, as today.

This keeps every existing command working: sessions, the skill's examples, llm-center and the
chat's composer still write `@name`, and the hub files it in the right place. No new command is
needed.

### Does `@inbox` stay?

As an **address**, yes: `@name` is still how anyone reaches a session, including for the first
contact. As a **place**, it empties: once pairs exist, nothing new lands in `@name` except what a
session writes to itself. A session keeps listening to its own `@name`, so nothing breaks for a
program that reads it.

### Old messages

The hub keeps messages in memory, so the inboxes hold only what was said since its last restart.
The change is installed with a hub restart, which empties them anyway; moving them into pairs
would be code that runs once and then never again. So nothing is moved. A session that wants the
context asks again in the pair.

### The command line

- `groups` prints the groups, then a line `pairs: alice+api, api+docs`, so pairs do
  not crowd the list.
- `read` prints a pair's messages with the pair's name as the topic; no other change.
- `group add`, `group remove` and `group delete` refuse a pair (`hub.pair-fixed`): a pair is always
  those two.

### The chat

Proposal 6 made the server keep which topics each person follows; pairs need no subscription,
because the person is a member.

- **Sessions** lists the pairs the person is in, one row per other member, named after them, with
  both sides of the conversation in order. Opening a contact who has no pair yet shows an empty
  conversation; the first message creates the pair (R1).
- **Between sessions** (inside Topics, folded) lists the pairs the person is not in. They are
  read-only: the composer is hidden, because the hub refuses a third publisher.
- The conversation key is the topic again, which is simpler than the first version of this
  proposal (which glued two inboxes together in the chat).

### The sessions skill

Section 3's "Answer where you were asked" gains a line: a message on a pair (`a+b`) is answered on
that pair, or to `@other`, which lands in the same place. Section 4's example `llmcp publish @bob`
stays, with a note that it opens or continues the pair `you+bob`.

### Shared files

Hub: `hub/names.go`, `hub/hub.go` (`Publish`), `hub/group.go`, `hub/keep.go`, `hub/errors.go`.
The LLM-CP-groups tree has someone else's uncommitted edits in `hub/`, `protocol/` and
`cmd/llmcp/` today; T0 commits them as they are, in a commit of their own, and the build goes on
top. Chat: `conversation.go`, `sections.go`, `draw.go`, on top of proposal 6.

### Alternatives not taken

- **The chat glues two inboxes together** (this proposal's first version): the chat reads one
  conversation, but each session still sees only half of it.
- **A new `talk` command**: clearer, but every session, the skill and llm-center would have to
  change what they write; redirecting `@name` changes nothing for them.
- **The chat creates the pair**: sessions talking to each other without the chat would still be
  split.
- **A `+` that sessions type**: the hub sorts the names, so nobody has to remember the order.

### The author's answers

1. `@name` **redirects** into the pair.
2. Old inbox messages are **not moved**.
3. Pairs between two sessions **show read-only**, folded under Topics.

## Tasks

On `feature/chat-subscriptions`, after proposal 3 of llm-center; one commit per task, no push.
A pair is recognised by the `+` in its name, so the protocol needs no new field.

- [ ] T0 — Commit the shut-down session's uncommitted edits in `hub/`, `protocol/`, `cmd/llmcp/` and the docs as they are, in one commit of their own (`llmcp: commit the work left uncommitted`), before anything else · · run `go test -race ./...` first and say in the commit if anything fails
- [ ] T1 — Pair names: `PairOf(a, b)` (sorted, joined by `+`), `validTopic` accepts a pair of two valid names, length up to two names and the `+` · `hub/names.go` · R1 · tests `a pair's name is the same whoever writes first`, `a plus sign is only allowed between two names`
- [ ] T2 — `Publish` to `@B` from `A` (A ≠ B) creates the pair group if needed (members A and B, both listening, kept) and adds the message to it; `group add`, `remove` and `delete` refuse a pair with `hub.pair-fixed` · `hub/hub.go`, `hub/group.go`, `hub/errors.go` · R1–R3 · tests `the first message to someone creates the pair`, `a message to @name lands in the pair`, `a third session cannot publish in a pair`, `a pair is kept`, `a pair cannot change its members`, and the errors test for the new code
- [ ] T3 — The command line: `publish` prints the pair it landed in; `groups` prints pairs on one line after the groups · `cmd/llmcp/print.go` · R4 · print tests
- [ ] T4 — The chat's conversations: a pair the person is in is one conversation titled with the other member; a pair without the person is read-only; opening a contact with no pair shows an empty conversation whose first message goes to `@contact` · `chat/conversation/`, `chat/cmd/chat/sections.go`, `draw.go` · R5, R6 · tests `a pair is one conversation named after the other`, `a pair without me is read-only`; screenshots desktop and phone in `docs/images/`
- [ ] T5 — Docs and skill: pairs in `docs/3-core-ideas.md` and `docs/6-reference/1-commands.md` (the `+`, the redirect, `hub.pair-fixed`), the chat guide; the sessions skill (`~/projects/llm-guidelines/skills/sessions/SKILL.md`: edit in place, do not commit, it holds others' edits) gets the two lines of "The sessions skill" above · R7 · by hand: two sessions talk through a pair following the skill
- [ ] T6 — Hand-over to deploy-manager: build `llmcp` and the chat; installing means a hub restart, so deploy-manager warns everyone on `center` first · · by hand on :8090

## Result

*Filled at the end.*
