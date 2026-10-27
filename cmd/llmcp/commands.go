package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/akrck02/LLM-CP/hub"
	"github.com/akrck02/LLM-CP/internal/failure"
	"github.com/akrck02/LLM-CP/protocol"
	"github.com/akrck02/LLM-CP/server"
)

// silenceLimit is how long a session can go without any command before the hub drops it.
const silenceLimit = 30 * time.Minute

// call is one command sent to the hub: who sends it, what it was given, the hub and the printer.
type call struct {
	session string
	given   options
	client  protocol.HubClient
	answer  printer
	request context.Context
}

// clientCommands are the commands that talk to the hub, by name.
var clientCommands = map[string]func(call) error{
	"join":     join,
	"leave":    leave,
	"sessions": sessions,
	"listen":   listen,
	"unlisten": unlisten,
	"publish":  publish,
	"read":     read,
	"group":    group,
	"groups":   groups,
	"archive":  archive,
	"restore":  restore,
	"typing":   typing,
	"topic":    topic,
	"metric":   metric,
}

// run carries out the command named first among the arguments, the options being anywhere.
func run(arguments []string) error {
	given, err := parseOptions(arguments)
	if err != nil {
		return err
	}
	if err := given.requireArguments(1); err != nil {
		return err
	}
	name := given.arguments[0]
	given.arguments = given.arguments[1:]
	if name == "hub" {
		return runHub(given.socket, given.keep)
	}
	command, found := clientCommands[name]
	if !found {
		return failure.Wrap(UnknownCommand, name, nil)
	}
	if name == "join" && len(given.arguments) > 0 {
		given.session = given.arguments[0]
	}
	return runClientCommand(command, given)
}

// runClientCommand connects to the hub, runs one command and prints its answer.
func runClientCommand(command func(call) error, given options) error {
	session, err := given.requireSession()
	if err != nil {
		return err
	}
	client, connection, err := connect(given.socket)
	if err != nil {
		return err
	}
	defer connection.Close()
	sent := call{
		session: session,
		given:   given,
		client:  client,
		answer:  newPrinter(os.Stdout, given.json),
		request: context.Background(),
	}
	return fromStatus(command(sent), given.socket)
}

// runHub serves a hub on the socket until the process is interrupted or terminated, keeping its
// declared topics and groups in a file (none for "").
func runHub(socket, keepPath string) error {
	meeting := hub.New(silenceLimit, time.Now)
	if keepPath != "" {
		if err := meeting.KeepIn(keepPath); err != nil {
			return err
		}
	}
	listener, err := server.Listen(socket)
	if err != nil {
		return err
	}
	served := server.New(meeting)
	stop, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-stop.Done()
		served.Stop()
	}()
	return served.Serve(listener)
}

// join joins the hub with the session's name.
func join(sent call) error {
	if _, err := sent.client.Join(sent.request, &protocol.SessionRequest{Session: sent.session}); err != nil {
		return err
	}
	if client := os.Getenv(clientVariable); client != "" {
		if err := rememberClient(sent.given.socket, client, sent.session); err != nil {
			return err
		}
	}
	return sent.answer.done("joined as " + sent.session + ", listening to " + hub.OwnTopic(sent.session))
}

// leave leaves the hub and frees the name.
func leave(sent call) error {
	if _, err := sent.client.Leave(sent.request, &protocol.SessionRequest{Session: sent.session}); err != nil {
		return err
	}
	if client := os.Getenv(clientVariable); client != "" {
		forgetClient(sent.given.socket, client, sent.session)
	}
	return sent.answer.done("left; " + sent.session + " is free")
}

// sessions lists the sessions joined and their topics.
func sessions(sent call) error {
	list, err := sent.client.Sessions(sent.request, &protocol.SessionRequest{Session: sent.session})
	if err != nil {
		return err
	}
	return sent.answer.sessions(list.Sessions)
}

// listen listens to the topic in the arguments.
func listen(sent call) error {
	if err := sent.given.requireArguments(1); err != nil {
		return err
	}
	topic := sent.given.arguments[0]
	asked := &protocol.ListenRequest{Session: sent.session, Topic: topic, Past: sent.given.past}
	if _, err := sent.client.Listen(sent.request, asked); err != nil {
		return err
	}
	return sent.answer.done("listening to " + topic)
}

// unlisten stops listening to the topic in the arguments.
func unlisten(sent call) error {
	if err := sent.given.requireArguments(1); err != nil {
		return err
	}
	topic := sent.given.arguments[0]
	if _, err := sent.client.Unlisten(sent.request, &protocol.TopicRequest{Session: sent.session, Topic: topic}); err != nil {
		return err
	}
	return sent.answer.done("not listening to " + topic)
}

// publish sends the text in the arguments, or on stdin when it is "-" or missing.
func publish(sent call) error {
	if err := sent.given.requireArguments(1); err != nil {
		return err
	}
	topic := sent.given.arguments[0]
	text := strings.Join(sent.given.arguments[1:], " ")
	if text == "" || text == "-" {
		read, err := io.ReadAll(os.Stdin)
		if err != nil {
			return failure.Wrap(StdinNotRead, topic, err)
		}
		text = strings.TrimRight(string(read), "\n")
	}
	asked := &protocol.PublishRequest{Session: sent.session, Topic: topic, Text: text}
	message, err := sent.client.Publish(sent.request, asked)
	if err != nil {
		return err
	}
	return sent.answer.published(message)
}

// read prints the waiting messages, waiting for one if asked to.
func read(sent call) error {
	asked := &protocol.ReadRequest{Session: sent.session, WaitSeconds: uint32(sent.given.waitSeconds)}
	list, err := sent.client.Read(sent.request, asked)
	if err != nil {
		return err
	}
	return sent.answer.messages(list.Messages)
}
