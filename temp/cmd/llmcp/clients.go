package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/akrck02/LLM-CP/internal/failure"
)

// clientVariable is where an assistant's client tells its own id; Claude Code sets it in every
// command it runs, and gives the same id to its hooks. A hook does not know the name the session
// chose on the hub, so join writes it down under the client's id for the hook to find.
const clientVariable = "CLAUDE_CODE_SESSION_ID"

// clientsFolder is where joined names are written down, one file per client, owner only.
func clientsFolder(socket string) string {
	return filepath.Join(filepath.Dir(socket), "llmcp-clients")
}

// rememberClient writes down which name a client joined with.
func rememberClient(socket, client, name string) error {
	folder := clientsFolder(socket)
	if err := os.MkdirAll(folder, 0o700); err != nil {
		return failure.Wrap(ClientNotRemembered, client, err)
	}
	if err := os.WriteFile(filepath.Join(folder, filepath.Base(client)), []byte(name), 0o600); err != nil {
		return failure.Wrap(ClientNotRemembered, client, err)
	}
	return nil
}

// forgetClient removes a client's note when it names the session that left, so its hook never
// finds a later session that took the same name.
func forgetClient(socket, client, name string) {
	written, err := clientSession(socket, client)
	if err != nil || written != name {
		return // nothing written, or another name: not this session's note
	}
	// a note left behind only costs a hook that finds no session, never a wrong one: the name is
	// checked above, so a failed removal is not worth failing the leave
	_ = os.Remove(filepath.Join(clientsFolder(socket), filepath.Base(client)))
}

// clientSession is the name a client last joined with.
func clientSession(socket, client string) (string, error) {
	written, err := os.ReadFile(filepath.Join(clientsFolder(socket), filepath.Base(client)))
	if errors.Is(err, os.ErrNotExist) {
		return "", failure.Wrap(UnknownClient, client, nil)
	}
	if err != nil {
		return "", failure.Wrap(UnknownClient, client, err)
	}
	return strings.TrimSpace(string(written)), nil
}
