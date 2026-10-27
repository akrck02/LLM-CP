package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/akrck02/LLM-CP/internal/failure"
)

// options is what a command was told, from its flags and the environment.
type options struct {
	session     string
	socket      string
	json        bool
	past        bool
	waitSeconds uint
	theme       string   // group create
	members     string   // group create, by commas
	stop        bool     // typing
	client      string   // an assistant's client id, to find its session (see clients.go)
	unit        string   // metric
	title       string   // metric: the name as people read it
	keep        string   // hub: the file of the declared topics and groups
	labels      []string // metric, each key=value
	arguments   []string // what is left once the flags are taken out
}

// defaultKeepPath is where a hub keeps its declared topics and groups unless told otherwise; with
// no home folder, nothing is kept.
func defaultKeepPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "llmcp", "kept.json")
}

// maximumWait keeps a waiting read shorter than the silence limit, so the hub does not drop a
// session while it waits.
const maximumWait = silenceLimit - time.Minute

// parseOptions reads the flags wherever they are among the arguments, up to a "--" after which
// everything is an argument, and fills what is missing from LLMCP_SESSION and LLMCP_SOCKET.
func parseOptions(arguments []string) (parsed options, err error) {
	arguments, literal, _ := cutAtDoubleDash(arguments)
	flags := flag.NewFlagSet("llmcp", flag.ContinueOnError)
	flags.SetOutput(io.Discard) // errors are returned with their code instead
	flags.StringVar(&parsed.session, "session", os.Getenv("LLMCP_SESSION"), "")
	flags.StringVar(&parsed.socket, "socket", os.Getenv("LLMCP_SOCKET"), "")
	flags.BoolVar(&parsed.json, "json", false, "")
	flags.BoolVar(&parsed.past, "past", false, "")
	flags.UintVar(&parsed.waitSeconds, "wait", 0, "")
	flags.StringVar(&parsed.theme, "theme", "", "")
	flags.StringVar(&parsed.members, "members", "", "")
	flags.BoolVar(&parsed.stop, "stop", false, "")
	flags.StringVar(&parsed.client, "client", "", "")
	flags.StringVar(&parsed.unit, "unit", "", "")
	flags.StringVar(&parsed.title, "title", "", "")
	flags.StringVar(&parsed.keep, "keep", defaultKeepPath(), "")
	flags.Func("label", "", func(label string) error {
		parsed.labels = append(parsed.labels, label)
		return nil
	})
	for {
		if err = flags.Parse(arguments); err != nil {
			return options{}, failure.Wrap(BadOption, strings.Join(arguments, " "), err)
		}
		if flags.NArg() == 0 {
			break
		}
		parsed.arguments = append(parsed.arguments, flags.Arg(0))
		arguments = flags.Args()[1:]
	}
	parsed.arguments = append(parsed.arguments, literal...)
	if parsed.waitSeconds > uint(maximumWait/time.Second) {
		return options{}, failure.Wrap(BadOption, "--wait is at most "+maximumWait.String(), nil)
	}
	if parsed.socket == "" {
		parsed.socket = defaultSocket()
	}
	if parsed.session == "" && parsed.client != "" {
		parsed.session, err = clientSession(parsed.socket, parsed.client)
	}
	return parsed, err
}

// cutAtDoubleDash splits the arguments at the first "--", which it drops.
func cutAtDoubleDash(arguments []string) (before, after []string, found bool) {
	index := slices.Index(arguments, "--")
	if index < 0 {
		return arguments, nil, false
	}
	return arguments[:index], arguments[index+1:], true
}

// defaultSocket is the user's runtime folder, where only they can write, or else a name with
// their user id in the temporary folder.
func defaultSocket() string {
	if runtime := os.Getenv("XDG_RUNTIME_DIR"); runtime != "" {
		return filepath.Join(runtime, "llmcp.sock")
	}
	return filepath.Join(os.TempDir(), "llmcp-"+strconv.Itoa(os.Getuid())+".sock")
}

// requireSession is the session's name, or an error saying how to give it.
func (given options) requireSession() (string, error) {
	if given.session == "" {
		return "", MissingSession
	}
	return given.session, nil
}

// requireArguments checks that at least that many arguments were given.
func (given options) requireArguments(count int) error {
	if len(given.arguments) < count {
		return MissingArgument
	}
	return nil
}
