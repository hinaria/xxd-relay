package relay

type SessionDescription struct {
    Secret string
    DestinationAddress string
}

var udpSessions = make(map[string]SessionDescription)
var tcpSessions = make(map[string]SessionDescription)
