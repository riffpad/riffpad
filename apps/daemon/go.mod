module github.com/riffpad/riffpad/apps/daemon

go 1.25.0

require (
	github.com/gorilla/websocket v1.5.3
	github.com/mdp/qrterminal/v3 v3.0.0
	github.com/riffpad/riffpad/packages/protocol v0.0.0
	github.com/riffpad/riffpad/packages/webui v0.0.0
	golang.org/x/term v0.41.0
)

require (
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	rsc.io/qr v0.2.0 // indirect
)

replace github.com/riffpad/riffpad/packages/protocol => ../../packages/protocol

replace github.com/riffpad/riffpad/packages/webui => ../../packages/webui
