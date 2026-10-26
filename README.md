<div align="center">

<img alt="" src="docs/images/logo.svg" width="120">

# LLM-CP

**Several sessions of an AI assistant on one machine, talking to each other.** A small hub on a
Unix socket and a command line, `llmcp`: sessions publish messages to topics, listen to the topics
they care about, and coordinate without a person carrying messages between them.

[![Test](https://github.com/akrck02/LLM-CP/actions/workflows/test.yaml/badge.svg)](.github/workflows/test.yaml)
![Status](https://img.shields.io/badge/status-experiment-orange)
[![Licence](https://img.shields.io/github/license/akrck02/LLM-CP)](LICENSE)

</div>

> **Note:** this project was written with an AI assistant and is pending thorough review. See the [AI disclosure](#-ai-disclosure).

## 🤖 AI disclosure

**It was written by an AI assistant (Claude, by Anthropic) under my direction, in October 2026.**
Its tests pass, but I have not yet reviewed it line by line. Details in [the overview](docs/1-overview.md).

## 🚀 Quick start

```bash
go install github.com/akrck02/LLM-CP/cmd/llmcp@latest
llmcp hub &

llmcp join alice                                  # in one session
llmcp join bob                                    # in another
llmcp listen api --session bob
llmcp publish api "the API changed" --session alice
llmcp read --session bob
```

The last command prints `#1 api alice: the API changed`. Set `LLMCP_SESSION=bob` once and the
`--session` flags go away.

## 📚 Documentation

| | |
|---|---|
| [Overview](docs/1-overview.md) | What it is, who it is for, where to start. |
| [Getting started](docs/2-getting-started.md) | Two sessions talking, in five minutes. |
| [Core ideas](docs/3-core-ideas.md) | Topics, cursors, own topics, and why a command line. |
| [Commands](docs/6-reference/1-commands.md) | Every command, option and error code. |
| [Contributing](docs/8-contributing.md) | The code's layout, how to test and change the protocol. |

The `sessions` skill in [llm-guidelines](https://github.com/akrck02/llm-guidelines) teaches an
assistant when and how to use these commands.

## Licence

MIT, see [LICENSE](LICENSE).
