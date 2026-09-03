package gsocket

import (
	"shopble/common/comsocket"
	"time"
)

const (
	WriteWait  = 10 * time.Second
	PongWait   = 60 * time.Second
	PingPeriod = (PongWait * 9) / 10
)

var (
	PublicNs  = comsocket.NewHub("public")
	PrivateNs = comsocket.NewHub("private")
)

var (
	PublicRoomVideoQueue = "video_queue"
)

func Broadcast(hub *comsocket.Hub, room string, env comsocket.Envelope) {
	hub.Broadcast(room, env)
}

type (
	NsHub map[string]*comsocket.Hub
)
