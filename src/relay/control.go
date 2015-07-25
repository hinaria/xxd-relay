package relay

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
)

const (
	ProtocolUdp = 1
	ProtocolTcp = 2

	ActionAdd    = 1
	ActionRemove = 2
)

type HttpControlRequest struct {
	Base64Secret string `json:"secret"`
	Destination  string `json:"destination"`
	Protocol     int    `json:"protocol"`
	Action       int    `json:"action"`
}

var success = []byte{123, 32, 34, 115, 116, 97, 116, 117, 115, 34, 58, 32, 34, 102, 111, 120, 105, 101, 115, 32, 97, 114, 101, 32, 97, 119, 101, 115, 111, 109, 101, 32, 58, 51, 34, 32, 125}

func HttpControlListen(address string) {
	fmt.Println("http listening on:", address)

	http.HandleFunc("/route", handle)
	http.ListenAndServe(address, nil)
}

func getSessionsForProtocol(protocol int) (map[string]SessionDescription, *sync.Mutex) {
	switch protocol {
	case ProtocolUdp:
		return udpSessions, &udpSessionsLock
	case ProtocolTcp:
		return tcpSessions, &tcpSessionsLock
	default:
		return nil, nil
	}
}

func handle(writer http.ResponseWriter, request *http.Request) {
	fmt.Println("http:", request.URL)

	if request.Body == nil {
		http.Error(writer, "no content body", 400)
		return
	}

	var params HttpControlRequest
	decoder := json.NewDecoder(request.Body)
	err := decoder.Decode(&params)

	if err != nil {
		http.Error(writer, "failed to decode json payload", 400)
		return
	}

	secretData, err := base64.StdEncoding.DecodeString(params.Base64Secret)
	if err != nil {
		http.Error(writer, "invalid secret encoding", 400)
		return
	}

	if len(secretData) != SecretLength {
		http.Error(writer, "invalid secret length", 400)
		return
	}

	sessions, sessionsLock := getSessionsForProtocol(params.Protocol)
	if sessions == nil {
		http.Error(writer, "invalid protocol", 400)
		return
	}

	addr, err := net.ResolveTCPAddr("tcp", params.Destination)
	if err != nil {
		http.Error(writer, "invalid destination", 400)
		return
	}

	secret := string(secretData)

	sessionsLock.Lock()
	switch params.Action {
	case ActionAdd:
		fmt.Println("adding route to", params.Destination, "with secret", secret)
		sessions[secret] = SessionDescription{secret, addr.String()}
	case ActionRemove:
		fmt.Println("removing route with the secret", secret)
		delete(sessions, secret)
	default:
		http.Error(writer, "invalid action", 400)
	}
	sessionsLock.Unlock()

	header := writer.Header()
	header.Set("Content-Type", "application/json")
	writer.Write(success)
}
