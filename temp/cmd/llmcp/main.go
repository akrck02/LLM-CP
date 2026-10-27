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

  llmcp hub [--keep <file>]         run the hub (leave it in the background: llmcp hub &)
  llmcp join <name>                 join with a unique name; you listen to @<name>
  llmcp leave                       leave and free your name
  llmcp sessions                    who has joined, and the topics each one listens to
  llmcp listen <topic> [--past]     listen to a topic; --past also reads what it already holds
  llmcp unlisten <topic>            stop listening to a topic
  llmcp publish <topic> <text…>     publish a message; text "-" or none reads stdin
  llmcp read [--wait <seconds>]     read your waiting messages, waiting for one if asked
  llmcp group create <name> --theme "<text>" --members alice,bob
                                    make a group: its members listen to it, only they publish
  llmcp group add|remove <name> <member>
  llmcp group delete <name>         turn a group back into a plain topic
  llmcp groups                      every group, its theme and members
  llmcp archive|restore <topic>     keep a chat out of the list, or bring it back
  llmcp typing [--stop]             show that you are thinking of an answer (30 s, or until you publish)
  llmcp topic declare <topic> conversation|metrics
                                    give a topic its kind, for its whole life
  llmcp topic remove <topic>        delete a topic and its messages, to make it again
  llmcp metric <topic> <name> <value> [--unit <unit>] [--title <title>] [--label key=value]…
                                    publish a point to a metrics topic (kept for 7 days)

Options, anywhere:
  --session <name>   who you are (or LLMCP_SESSION)
  --socket <path>    the hub's socket (or LLMCP_SOCKET)
  --json             answer in JSON
  --client <id>      find your session by the client you joined from (for hooks)
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
