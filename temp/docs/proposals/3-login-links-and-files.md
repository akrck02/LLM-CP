# 3. The app remembers you, opens links, and carries files

> Status: proposed · Author's approval: proposal ☐ plan ☐

## Why

On the phone, the app asks for the server and the name every time it starts. A link in a message
is plain text that cannot be followed. And sessions often need to pass a file (a log, a
screenshot, a patch), which today they can only paste as text, or not at all.

## What

| Id | Requirement | Check |
|---|---|---|
| R1 | The app remembers the server and the name it last connected with, and connects on its own when it starts; a "sign out" in the side pane forgets them | test `the link is kept and read back`; by hand on a phone |
| R2 | A link in a message (`http://`, `https://`) is drawn as a link; tapping it opens the phone's or computer's browser | test `links are found in a message's text`; by hand |
| R3 | A message can carry one file: a session sends it with `llmcp publish <topic> --file <path> [text]`, and a person from the page or the app | test `a file published reaches its readers with its name, type and size` |
| R4 | `llmcp read` names a message's file (`[file: report.log, 12 KB, #7]`) and `llmcp file <number> [-o path]` saves it, so a session can read what it was sent | test `a session saves the file of a message` |
| R5 | The page and the app show a picture inline (PNG, JPEG, GIF, WebP), and any other file as a card with its name, type and size that opens or downloads it | by hand, with screenshots |
| R6 | Files are kept in the hub's memory with limits: at most 10 MB each and 200 MB in all; when full, the oldest files are forgotten first and their messages say so | tests `a file over the limit is refused`, `the oldest file is forgotten when the hub is full` |

## Out of scope

- Keeping files after the hub stops (the hub has no disk, as before).
- Several files in one message, previews of PDFs or videos, editing a sent file.
- Choosing a file **on Android** in this change: Go Renderer has no way yet to open the system's
  file chooser, and adding one is Java and JNI work in that repository. The app shows and opens
  files; sending from the phone works through the page in the phone's browser until then.
  If you prefer, it becomes a task of its own here, in a `feature/` branch of Go Renderer.

## Design

_To fill once the proposal is approved._

## Tasks

_To fill once the proposal is approved._

## Result

_Pending._
