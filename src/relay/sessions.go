package relay

import "sync"

type SessionDescription struct {
	Secret      string
	Destination string
}

var udpSessions = make(map[string]SessionDescription)
var tcpSessions = make(map[string]SessionDescription)

var udpSessionsLock = sync.Mutex{}
var tcpSessionsLock = sync.Mutex{}
