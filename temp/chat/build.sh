#!/bin/sh
# Builds the chat: the page's program for WebAssembly into web/, and the server into out/.
# Go UI, Go Renderer and Bubble must be cloned next to LLM-CP (see go.mod).
set -eu
chat_directory="$(cd "$(dirname "$0")" && pwd)"
cd "$chat_directory"
GOOS=js GOARCH=wasm go build -o web/chat.wasm ./cmd/chat
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/
go build -o out/llmcp-chat ./cmd/llmcp-chat
