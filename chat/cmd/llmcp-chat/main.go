// Command llmcp-chat shows what the sessions on a hub say, as a chat in a web page, and lets a
// person write to them. It watches the hub and passes each message on to every open page as a
// server-sent event; the page, a Go UI program built for WebAssembly (cmd/chat-web), draws the
// chats, and what the person writes comes back through /publish, sent as a session of their name.
//
//	llmcp-chat -address :8090 -web web
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

func main() {
	address := flag.String("address", "127.0.0.1:8090", "where the page is served; :8090 to reach it from other machines")
	webDirectory := flag.String("web", "web", "the folder with index.html, chat.wasm and wasm_exec.js")
	socket := flag.String("socket", os.Getenv("LLMCP_SOCKET"), "the hub's socket (default as llmcp's)")
	flag.Parse()
	if *socket == "" {
		*socket = defaultSocket()
	}

	relay := newRelay()
	go relay.followHub(*socket)

	routes := http.NewServeMux()
	routes.Handle("/", http.FileServer(http.Dir(*webDirectory)))
	routes.HandleFunc("/events", relay.serveEvents)
	sender := publisher{socket: *socket}
	routes.Handle("/publish", fromOwnPage(http.HandlerFunc(sender.servePublish)))
	routes.Handle("/groups", fromOwnPage(sender.serveAction(createGroup)))
	routes.Handle("/members/add", fromOwnPage(sender.serveAction(addMembers)))
	routes.Handle("/members/remove", fromOwnPage(sender.serveAction(removeMembers)))
	routes.Handle("/archive", fromOwnPage(sender.serveAction(archiveTopic)))
	routes.Handle("/restore", fromOwnPage(sender.serveAction(restoreTopic)))
	slog.Info("serving the chat", "address", *address, "socket", *socket)
	if err := http.ListenAndServe(*address, logRequests(routes)); err != nil {
		fmt.Fprintln(os.Stderr, "llmcp-chat:", err)
		os.Exit(1)
	}
}

// defaultSocket is where llmcp puts the hub's socket when it is not told.
func defaultSocket() string {
	if runtime := os.Getenv("XDG_RUNTIME_DIR"); runtime != "" {
		return filepath.Join(runtime, "llmcp.sock")
	}
	return filepath.Join(os.TempDir(), "llmcp-"+strconv.Itoa(os.Getuid())+".sock")
}

// logRequests writes each request's method, path and who made it, to see which pages and apps
// reach the server.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		slog.Info("request", "method", request.Method, "path", request.URL.Path, "from", request.RemoteAddr, "agent", request.UserAgent())
		next.ServeHTTP(response, request)
	})
}
