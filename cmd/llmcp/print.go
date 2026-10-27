package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/akrck02/LLM-CP/protocol"
)

// printer writes answers as short plain text, which costs a reader few tokens, or as JSON for
// programs.
type printer struct {
	output io.Writer
	json   bool
}

// newPrinter makes a printer to an output, in JSON or plain text.
func newPrinter(output io.Writer, asJSON bool) printer {
	return printer{output: output, json: asJSON}
}

// done says a command worked.
func (answer printer) done(text string) error {
	if answer.json {
		return answer.encode(map[string]string{"done": text})
	}
	_, err := fmt.Fprintln(answer.output, text)
	return err
}

// sessions prints one line per session: its name and the topics it listens to.
func (answer printer) sessions(sessions []*protocol.Session) error {
	if answer.json {
		return answer.encode(sessions)
	}
	for _, session := range sessions {
		if _, err := fmt.Fprintf(answer.output, "%s: %s\n", session.Name, strings.Join(session.Topics, " ")); err != nil {
			return err
		}
	}
	return nil
}

// published says where a message went and its number.
func (answer printer) published(message *protocol.Message) error {
	if answer.json {
		return answer.encode(message)
	}
	_, err := fmt.Fprintf(answer.output, "#%d published to %s\n", message.Number, message.Topic)
	return err
}

// messages prints one line per message: its number, topic, sender and text, with the text's own
// line breaks written as \n so a message never spills over several lines.
func (answer printer) messages(messages []*protocol.Message) error {
	if answer.json {
		if messages == nil {
			messages = []*protocol.Message{}
		}
		return answer.encode(messages)
	}
	if len(messages) == 0 {
		_, err := fmt.Fprintln(answer.output, "no messages")
		return err
	}
	for _, message := range messages {
		text := strings.ReplaceAll(message.Text, "\n", `\n`)
		if _, err := fmt.Fprintf(answer.output, "#%d %s %s: %s\n", message.Number, message.Topic, message.From, text); err != nil {
			return err
		}
	}
	return nil
}

// groups prints one line per group: its name, members and theme.
func (answer printer) groups(groups []*protocol.Group) error {
	if answer.json {
		if groups == nil {
			groups = []*protocol.Group{}
		}
		return answer.encode(groups)
	}
	if len(groups) == 0 {
		_, err := fmt.Fprintln(answer.output, "no groups")
		return err
	}
	for _, group := range groups {
		line := group.Name + " (" + strings.Join(group.Members, ", ") + ")"
		if group.Theme != "" {
			line += ": " + group.Theme
		}
		if _, err := fmt.Fprintln(answer.output, line); err != nil {
			return err
		}
	}
	return nil
}

// encode writes a value as one line of JSON.
func (answer printer) encode(value any) error {
	return json.NewEncoder(answer.output).Encode(value)
}
