# 6. The chat shows the topics you are subscribed to

> Status: built, waits for review · Author's approval: proposal ☑ (2026-10-27, with the answers below) plan ☑ (2026-10-27)

## Why

The chat (`llmcp-chat`, on :8090) lists every topic on the hub: every session's topic, every
metrics topic, every group. As more sessions join, the list grows past what a person follows, and
there is no way to say "this one is not for me".

The hub already has the idea: a session **listens** to topics (`llmcp listen`, `llmcp unlisten`,
the `Listen` and `Unlisten` calls). The person at the chat is a session too: the chat server
joins the hub under their name when they write. So "subscribed" can mean "listening", and
everyone sees it in each chat's header ("Listening: …").

There is one catch. The hub forgets a session after half an hour of silence, and on a restart.
The person's listens would vanish with it, so the chat must keep them somewhere else and give
them back.

## What

| Id | Requirement | Check |
|---|---|---|
| R1 | The Topics section lists only the topics the person is subscribed to, metrics topics included, plus their groups (a member always sees its group) | test: a state with three topics, subscribed to one, lists one |
| R2 | From an open topic, the person subscribes or unsubscribes with one button in its header; the list changes at once | by hand in the page and the app; test of the server's endpoint |
| R3 | The topics they are not subscribed to are one tap away, in a "More topics" list (like "Archived"), with a Subscribe button each | test: the unsubscribed topics are listed there |
| R4 | A subscription is the person's session listening to the topic on the hub, so other sessions and the header see it | test: after subscribing, the hub lists the topic among the person's |
| R5 | Subscriptions outlast a silent half hour and a hub restart: the chat server keeps them per person and listens again when it joins them again | test: drop the session, write again, the listens are back |
| R6 | The same person sees the same subscriptions on the web page and on the phone app | follows from R5: they are kept per name on the server |
| R7 | Someone new starts subscribed to nothing: they see their own inbox and their groups, and add topics from "More topics" | test: a new name has no subscriptions, and still sees its inbox |

## Out of scope

- **Inboxes** (the Sessions section) are not subscriptions: a message to you always shows.
- **Leaving a group** stays what it is today, removing a member. Unsubscribe applies to topics,
  not groups.
- **Archive** stays as it is: it hides a chat for everyone, while unsubscribing hides it for one
  person.
- **Notifications** per topic.

## Design

**The author's answers (2026-10-27):** new people start subscribed to nothing, their own inbox
still showing (R7). Metrics topics are subscribed to like any other (R1).

**Who knows the subscriptions.** The chat server keeps them, one list per name, in a JSON file
next to it (`subscriptions.json` in its working folder: `~/.local/share/llmcp-chat` as it runs
today). The file is the truth. The hub's listens mirror it, so other sessions and the header see
them (R4).

- **Server (`chat/cmd/llmcp-chat`).**
  - A `subscriptions` store reads and writes the file (load once, save after each change, written
    to a temporary file and renamed, so a crash never leaves half a file).
  - A `/subscriptions` endpoint. `GET ?as=<name>` gives the person's topics. `POST {as, topic,
    subscribed}` changes one: it calls the hub's `Listen` or `Unlisten` through `withClient`, then
    keeps the change.
  - `withClient`, after joining someone again, listens to their kept topics again (R5). A `GET`
    joins and listens too, so opening the page puts the person back on the hub.
- **Page and app (`chat/cmd/chat`).** They ask for the person's subscriptions when they start,
  and send each change. The page uses `fetch`; the app uses `http`, as `send` does today.
  `conversation.Log` learns which topics are subscribed, and gives the Topics section only those
  plus the person's groups (R1), and "More topics" the others (R3). The open topic's header gets a
  subscribe button (Lucide `bell` or `bell-off`) (R2).
- **The protocol and the hub stay as they are.**

**Not taken:**

- Keeping the subscriptions only in the browser: per device, invisible to other sessions, and
  not a real LLM-CP listen.
- Taking them from the hub's state (`Contact.Topics`): the person's session is often not on the
  hub when the page opens, so the list would be empty.
- Making the hub remember people's sessions: a bigger change to the hub, for a need the chat can
  meet alone.

## Tasks

- [x] T1 — The subscriptions store: per-name topics, kept in a file, read again on start ·
  `chat/cmd/llmcp-chat/subscriptions.go` and its test · serves R5, R6, R7 · tests: `a subscription
  is kept across a restart of the server`, `a new name has no subscriptions`
- [x] T2 — The `/subscriptions` endpoint and listening again after a rejoin · `subscriptions.go`,
  `actions.go` (`withClient`), `main.go` (the route) · serves R2, R4, R5 · tests on an in-process
  hub, as `publish_test.go` does: `subscribing makes the person listen on the hub`, `after a
  silence the person listens to their topics again`
- [x] T3 — `conversation.Log` splits the topics: subscribed and groups in Topics, the rest in
  "More topics"; the inbox always shows · `chat/conversation/conversation.go` and its test ·
  serves R1, R3, R7 · tests: `only subscribed topics and groups are listed`, `the other topics are
  more topics`
- [x] T4 — Page and app: fetch the subscriptions at start, the header's subscribe button, the
  "More topics" list · `chat/cmd/chat/sections.go`, `draw.go`, `page_web.go`, `page_native.go` ·
  serves R2, R3, R6 · check: in headless Chromium, subscribe, unsubscribe, reload, the list stays
- [ ] T5 — Docs: what subscribing means in the chat, and the endpoint · `docs/5-guides/
  1-watching-the-sessions.md`, `docs/8-contributing.md` (the chat server's routes) · serves R1–R7

Each task leaves the chat working. The filter (T3) only switches on in T4, once the page knows the
person's subscriptions. Until then the chat lists everything, as today.

## Result

Built on `feature/chat-subscriptions`, not pushed; AI-written, pending human review.

- The server keeps the subscriptions in `subscriptions.json` (option `-subscriptions`), with two
  error codes of its own, `chat.subscriptions-unreadable` and `chat.subscriptions-unwritable`.
- `/subscriptions`: `GET ?as=` gives the list (`[]` for someone new) and listens again on the hub;
  `POST {as, topic, subscribed}` changes one, on the hub first and then in the file.
- The split of topics lives in `chat/conversation/subscriptions.go` rather than
  `conversation.go`: it is a part of its own.
- The header's button shows what a click does: `bell-plus` to subscribe, `bell-minus` to stop.
  Inboxes and the groups the person is in have none. "More topics" uses `compass`.
- Checked in headless Chromium on a private hub: a new name sees no topics; subscribing from More
  topics and from the header lists the topic and makes the session listen; a reload keeps it;
  unsubscribing takes it away, from the file and from the hub.
- Not checked: the Android app (no device), and other browsers.
- Known: when the first message to arrive is in a topic the person does not follow, it still
  opens as the first chat, as before.
