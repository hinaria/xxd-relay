package relay

import "sync"

type PendingSessionDescription struct {
	Secret      string
	Destination string
}

type PendingSessionGroups struct {
	Udp     map[string]PendingSessionDescription
	Tcp     map[string]PendingSessionDescription
	UdpLock sync.Mutex
	TcpLock sync.Mutex
}

var pendingSessions = PendingSessionGroups{make(map[string]PendingSessionDescription), make(map[string]PendingSessionDescription), sync.Mutex{}, sync.Mutex{}}
