// Command llmcp lets several sessions of an AI assistant on the same machine talk through topics.
// `llmcp hub` runs the meeting point; every other command talks to it and answers on stdout.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/akrck02/LLM-CP/internal/failure"
)

const usage = `llmcp: sessions on one machine talk through topics.

  llmcp hub                         run the hub (leave it in the background: llmcp hub &)
  llmcp join <name>                 join with a unique name; you listen to @<name>
  llmcp leave                       leave and free your name
  llmcp sessions                    who has joined, and the topics each one listens to
  llmcp listen <topic> [--past]     listen to a topic; --past also reads what it already holds
  llmcp unlisten <topic>            stop listening to a topic
  llmcp publish <topic> <text…>     publish a message; text "-" or none reads stdin
  llmcp read [--wait <seconds>]     read your waiting messages, waiting for one if asked

Options, anywhere:
  --session <name>   who you are (or LLMCP_SESSION)
  --socket <path>    the hub's socket (or LLMCP_SOCKET)
  --json             answer in JSON
`

func main() {
	if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "--help" {
		fmt.Print(usage)
		return
	}
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, describe(err))
		os.Exit(1)
	}
}

// describe is an error for a person: its code and details, then what the code means.
func describe(err error) string {
	var failed *failure.Error
	if errors.As(err, &failed) {
		return failed.Error() + " (" + failed.Code.Meaning() + ")"
	}
	return err.Error()
}
