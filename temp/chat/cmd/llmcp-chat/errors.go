package main

import "github.com/akrck02/LLM-CP/internal/failure"

// The codes of the errors the chat server makes itself; those of the hub come from the hub.
var (
	SubscriptionsUnreadable = failure.Define("chat.subscriptions-unreadable", "the file of people's subscriptions cannot be read")
	SubscriptionsUnwritable = failure.Define("chat.subscriptions-unwritable", "the file of people's subscriptions could not be written")
)
