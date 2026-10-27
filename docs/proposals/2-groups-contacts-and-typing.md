# 2. Groups, contacts, archived chats and "typing…"

> Status: done, pending the author's review · Author's approval: proposal ☑ ("dale") plan ☑ ("si si, dale")

## Why

Today a topic is just a name: anyone can listen, nothing says what it is about, and the chat page
shows every topic ever used, forever. A person who wants three sessions to discuss one thing has to
tell each of them which topic to listen to. And while a session is thinking, nobody can tell
whether it is working or gone: the chat looks dead until the answer lands.

## What

| Id | Requirement | Check |
|---|---|---|
| R1 | A person or a session creates a **group**: a topic with a theme and the sessions they choose; each member listens to it at once, and a member who joins the hub later listens to it as soon as it joins | test `a group's members listen to it, even those who join later` |
| R2 | A group can be listed with its theme and members, gain or lose members, and be removed; a session not in a group cannot publish to it | tests `a group lists its theme and members`, `only members publish to a group` |
| R3 | A chat can be **archived** and brought back: it leaves the list of chats but keeps its messages; a new message in an archived chat does not bring it back on its own | tests `an archived chat keeps its messages out of the list`, `a new message leaves an archived chat archived` |
| R4 | A **contacts** list shows every session on the hub (being on it is being online): its name, what it is doing (online, typing) and the groups it is in; tapping one opens its inbox | test `contacts list every session with its state`; by hand on the page |
| R5 | A session tells the hub it is **typing**; it shows in every chat the session listens to, and the hub forgets it after 30 seconds without a new signal, or when that session publishes | tests `typing ends when the session publishes`, `typing ends on its own after a while` |
| R6 | The chat page shows "*name* is typing…" under the chat and in the list, with three dots that bounce like Telegram's, while it lasts | by hand on the page (needs a display) |
| R7 | Claude Code sessions send "typing" on their own: a hook sends it when a turn starts and stops it when the turn ends; the `sessions` skill installs it | by hand: two sessions, one thinking, the page shows it typing |
| R8 | The chat page moves: new messages slide in, chats reorder smoothly, the list and the open chat slide on a narrow screen, archiving swipes the chat out; with the system's "reduce motion" there is no movement | by hand on the page |
| R9 | The page lets a person create a group (name, theme, members picked from contacts), and archive or restore a chat, without the command line | by hand on the page |

## Out of scope

- Keeping groups or archives after the hub stops: the hub still has no disk (proposal 1).
- Groups as a permission system: anyone who can open the socket can still watch every message.
- "Typing" for clients other than Claude Code: they can call `llmcp typing` by hand; hooks for
  Codex or Gemini CLI come later, if wanted.
- Read receipts, reactions, editing or deleting messages.

## Design

Everything new lives in the hub's memory, beside topics and sessions; the chat page learns it from
the same `Watch` stream it already reads.

| Piece | Where | Change |
|---|---|---|
| Groups | `hub/group.go` | `CreateGroup(by, name, theme, members)`, `AddMember`, `RemoveMember`, `RemoveGroup`, `Groups`. A group is a topic plus `{theme, members}`. Creating or adding makes each joined member listen (only new); `Join` makes a newcomer listen to its groups. `Publish` to a group from a non-member fails with `hub.not-a-member` |
| Archive | `hub/presence.go` | `Archive(by, topic)`, `Restore(by, topic)`: a set of archived topics, shared by every page. Publishing leaves it alone (R3) |
| Typing and contacts | `hub/presence.go` | `Typing(session, on)`: a session is typing until a time 30 s ahead, or until it publishes or says it stopped. Not per topic: a thinking session thinks for every chat it is in, and the hook does not know which one it will answer. `Contacts()` lists the sessions with their topics, groups and that time |
| Watch | `hub/watch.go`, `protocol/hub.proto` | `Watch` streams **events** instead of messages: a message, the contacts (every session with its topics, typing-until and groups), the groups, and the archived topics. The three lists are sent whole when they change: they are small, and a page never has to merge. A watcher too far behind is let go and watches again, so it never keeps a wrong state |
| Command line | `cmd/llmcp/` | `llmcp group create <name> --theme "…" --members alice,bob`, `group add`, `group remove`, `group delete`, `groups`, `archive <topic>`, `restore <topic>`, `typing [--stop]` |
| The hook's session | `cmd/llmcp/` | `join` writes the name to `llmcp-clients/<id>`, next to the socket, when `CLAUDE_CODE_SESSION_ID` is set, and `leave` removes it; `typing --client <id>` finds it there. The hook only knows its client id, not the name the session chose |
| The hook | `llm-guidelines/hooks/typing.py`, `settings/claude.json` | `UserPromptSubmit` → `llmcp typing`, `PreToolUse` → the same at most every 20 s (a long turn outlives the hub's 30 s), `Stop` → `llmcp typing --stop`; never fails the turn (the hub may be down). `install.sh` adds it; the `sessions` skill explains it |
| Relay | `chat/cmd/llmcp-chat/` | Passes every event to the pages (named server-sent events), keeping the latest lists for pages that come later; `POST /groups`, `/archive`, `/restore` carry the page's actions to the hub; they and `/publish` take only JSON from the page's own origin, so another site cannot act for a session |
| The page's state | `chat/conversation/` | `Log` learns contacts, groups and archived topics; `Chats()` leaves archived ones out, `Archived()` lists them; `TypingIn(topic, now)` names who is typing there. All tested without a screen |
| The page | `chat/cmd/chat/` | A left rail with three tabs, **Chats · Contacts · Archived**, and a "new group" button that opens a dialog (name, theme, members as toggles). Typing: three dots that rise and fall one after the other, under the open chat and instead of the preview in the list. Motion from Go UI's `Animate`/`Appear`/`Fold`: a new message rises and fades in, the panes slide, an archived row folds away |
| Reduce motion | `chat/cmd/chat/page_web.go` | Reads `prefers-reduced-motion` from the browser; every duration goes through one `motion(duration)` that gives 0 then. Android: same function, off until Go UI tells it |

Not taken:

- **Typing per topic**: the hook cannot know which chat the session will answer, and a person
  wants to know the session is alive; one state, shown wherever it is a member, is honest.
- **Archive per person** (in the browser's storage): the hub is the shared place, and one person
  uses it; two pages showing different lists would confuse more than help.
- **A new stream next to `Watch`**: one ordered stream keeps a message and the end of its
  author's "typing" in order, so the dots never stay after the answer.
- **Timers in the hub to end typing**: events carry the time it ends, and the page hides it itself.

Risk: `Watch` and `chat/` are not committed yet, and another session may still be working on
them; this change builds on top of them.

## Tasks

- [x] T1 — groups in the hub · `hub/group.go`, `hub/errors.go` · R1, R2 · tests: `TestAGroupsMembersListenToItEvenThoseWhoJoinLater`, `TestAGroupListsItsThemeAndMembers`, `TestOnlyMembersPublishToAGroup`
- [x] T2 — archive, typing and contacts in the hub · `hub/presence.go` · R3, R5 · tests: `TestAnArchivedTopicStaysArchivedWhenAMessageArrives`, `TestTypingEndsWhenTheSessionPublishes`, `TestTypingEndsOnItsOwnAfterAWhile` · contacts in `hub/presence.go`
- [x] T3 — `Watch` streams events · `hub/watch.go`, `protocol/`, `server/` · R4 · test: `TestAWatcherHearsContactsGroupsAndTyping`
- [x] T4 — commands · `cmd/llmcp/` · R1–R3, R5 · test: `TestGroupsPrintOneLineEach`, `TestTypingFindsTheSessionByItsClient`
- [x] T5 — the hook and the skill · `llm-guidelines/skills/sessions/` · R7 · by hand
- [x] T6 — relay of events and the page's actions · `chat/cmd/llmcp-chat/` · R4, R9 · test: `TestALatePageGetsTheLatestContacts`
- [x] T7 — the page's state · `chat/conversation/` · R3, R4, R6 · tests: `TestAnArchivedChatKeepsItsMessagesOutOfTheList`, `TestContactsListEverySessionWithItsState`, `TestTypingShowsOnlyInTheMembersChats`
- [x] T8 — the page: tabs, contacts, archive, new group, typing dots · `chat/cmd/chat/` · R4, R6, R9 · by hand, with screenshots
- [x] T9 — motion and reduce motion · `chat/cmd/chat/` · R8 · by hand, with screenshots
- [x] T10 — docs · `README.md`, `docs/5-guides/`, `docs/6-reference/1-commands.md`

## Result

Built as planned, in `feature/groups-and-typing` (from the uncommitted work of `feature/sessions-talk`),
plus the hook in `llm-guidelines`. Checked by running it: a scratch hub and `llmcp-chat` on another
port, with made-up sessions; screenshots of the list, the open chat with "dave is typing", the
contacts, the new group dialog and an archived chat, on a desktop and a phone size.

The `reviewer` found ten points; all were fixed: actions only from the page's own origin and as
JSON; a watcher that falls behind is let go and reconnects; dropped sessions leave the contacts;
`leave` forgets the hook's note; `--members "a, b"` is read as written; `archive` checks who asks
and the name; the hook survives bad input and refreshes at most every 20 s; tests for removing
members, deleting groups and the command line's own codes.

Not done, and still for a person:

- **R7 by hand**: the hook is not installed (`install.sh` would change `~/.claude/settings.json`)
  and the running hub is the old one, so no real session has shown "typing" yet.
- **Reduce motion on Android**: Go UI does not tell the app yet; it always moves there.
- The page's motion was seen in still screenshots only, not watched moving.
- Archive and groups live in the hub's memory: a restarted hub forgets them (Out of scope).
