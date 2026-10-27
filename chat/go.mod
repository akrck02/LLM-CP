module github.com/akrck02/LLM-CP/chat

go 1.26.8

replace github.com/akrck02/LLM-CP => ../

require (
	github.com/akrck02/LLM-CP v0.0.0
	github.com/akrck02/bubble v0.0.0
	github.com/akrck02/bubble/goui v0.0.0-00010101000000-000000000000
	github.com/akrck02/go-renderer v0.2.1-0.20261022213249-8ecc924ab3e5
	github.com/akrck02/go-ui v0.0.0
	google.golang.org/grpc v1.84.0
)

require (
	github.com/go-gl/gl v0.0.0-20260331235117-4566fea9a276 // indirect
	github.com/go-gl/glfw/v3.4/glfw v0.1.0-pre.1 // indirect
	golang.org/x/image v0.45.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

// Go UI, Go Renderer and Bubble are cloned next to LLM-CP, as Bubble's goui module expects them.
replace (
	github.com/akrck02/bubble => ../../bubble
	github.com/akrck02/bubble/goui => ../../bubble/goui
	github.com/akrck02/go-renderer => ../../go-renderer-browser-emoji
	github.com/akrck02/go-ui => ../../go-ui-scroll-from-end
)
