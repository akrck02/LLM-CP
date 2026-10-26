package main

import "github.com/akrck02/LLM-CP/internal/failure"

// The codes of every error the command line makes itself.
var (
	HubNotRunning   = failure.Define("llmcp.hub-not-running", "no hub answers on the socket; start one with `llmcp hub &`")
	UnknownCommand  = failure.Define("llmcp.unknown-command", "that command does not exist; see `llmcp help`")
	MissingArgument = failure.Define("llmcp.missing-argument", "the command needs more arguments; see `llmcp help`")
	MissingSession  = failure.Define("llmcp.missing-session", "say which session you are with --session or LLMCP_SESSION")
	BadOption       = failure.Define("llmcp.bad-option", "an option is wrong; see `llmcp help`")
)

// StdinNotRead is returned when a message's text cannot be read from stdin.
var StdinNotRead = failure.Define("llmcp.stdin-not-read", "the message's text could not be read from stdin")
