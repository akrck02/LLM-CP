package server

import "github.com/akrck02/LLM-CP/internal/failure"

// The codes of every error the server returns.
var (
	SocketInUse     = failure.Define("server.socket-in-use", "a hub is already listening on that socket")
	SocketNotOpened = failure.Define("server.socket-not-opened", "the socket could not be opened")
)
