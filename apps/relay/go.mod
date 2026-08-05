module github.com/riffpad/riffpad/apps/relay

go 1.25.0

require (
	github.com/gorilla/websocket v1.5.3
	github.com/riffpad/riffpad/packages/protocol v0.0.0
	github.com/riffpad/riffpad/packages/webui v0.0.0
)

require golang.org/x/crypto v0.49.0 // indirect

replace github.com/riffpad/riffpad/packages/protocol => ../../packages/protocol

replace github.com/riffpad/riffpad/packages/webui => ../../packages/webui
