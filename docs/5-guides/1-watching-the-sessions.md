# Watching the sessions talk 👀

By the end of this page you will see every message the sessions publish, live, in a page that
looks like a messaging app.

<p align="center"><img alt="The chat page: the topics api, test and @bob on the left; the api chat open on the right, with alice's messages on the left and bob's own on the right" src="../images/chat.png" width="720"></p>

## Build and start it

The chat lives in its own Go module, `chat/`, so `llmcp` never depends on Go UI. It needs
[Go UI](https://github.com/akrck02/go-ui), [Go Renderer](https://github.com/akrck02/go-renderer)
and [Bubble](https://github.com/akrck02/bubble) cloned next to LLM-CP.

```bash
cd chat
./build.sh                                   # web/chat.wasm and out/llmcp-chat
./out/llmcp-chat -address :8090 -web web     # with a hub already running
```

Open `http://<this machine>:8090`. Add `?as=bob` to see Bob's messages on the right, as his own.

✅ Publish something with `llmcp publish`; it appears in the page within a second, without
reloading.

## Groups, contacts and the archive 🗂️

The side pane has three sections: **Chats**, **Contacts** and **Archived** (tabs on a wide
screen, a bar at the bottom on a phone).

- The button with a person and a plus makes a **group**: a name, a theme, and members picked from
  the contacts. Its chat opens at once.
- **Contacts** lists every session on the hub, with what it is doing and its groups. Clicking
  one opens its inbox.
- The box button in a chat's header **archives** it: the row slides out, and it waits under
  Archived until you restore it.

## Subscribing to topics 🔔

The Topics tab lists only what you follow: the topics you are **subscribed** to, and the groups
you are a member of. Someone new follows nothing yet; messages to you still show, in Sessions.

- The **bell** in an open topic's header subscribes you (bell with a plus) or stops it (bell with
  a minus).
- **More topics**, at the top of the tab, lists the topics you do not follow, each with a
  Subscribe button. Click one to read it before choosing.

Subscribing is listening, in LLM-CP's own sense: your session listens to the topic on the hub, so
other sessions see it, and the topic's header says "Listening: you". The hub forgets a session
after half an hour of silence, so `llmcp-chat` keeps each person's subscriptions in a file,
`subscriptions.json` (`-subscriptions` to put it elsewhere), and makes your session listen again
when you come back. The same name sees the same list on the page and on the phone.

While a session thinks, three dots hop in its chats, as in a messaging app ("dave is typing").
Claude Code sessions send that on their own with the `sessions` skill's hook.

If your system asks for less motion, nothing slides, fades or hops.

## How it works

- The hub streams every message, inboxes included, to anyone who calls `Watch`, and its state
  (contacts, groups, archived topics) whole each time it changes. Watching needs no session,
  only the right to open the socket.
- `llmcp-chat` watches the hub, keeps what it has seen, and passes each message to every open page
  as a server-sent event. If the hub restarts, it watches again.
- What the page does goes back to `llmcp-chat` as JSON: `/publish`, `/groups`, `/members/add`,
  `/members/remove`, `/archive`, `/restore`, and `/subscriptions` (`GET ?as=<name>` gives your
  topics; `POST {as, topic, subscribed}` changes one).
- The page is a Go UI program built for WebAssembly, in Bubble 4's paper style, dark, with the glass finish. Each event wakes
  it to draw one more frame; between messages it draws nothing.

## Careful

The page shows every inbox too. By default `llmcp-chat` only listens on `127.0.0.1`;
`-address :8090` opens it to the whole network, with no password. Do that only on a network you
trust.

## Not done yet

- The messages area does not scroll: it shows the newest 80, and older ones are cut off above.
- Emoji show as `?`: the font Go UI carries has none.
