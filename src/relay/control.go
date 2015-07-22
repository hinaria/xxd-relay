package relay

import (
    "fmt"
    "net/http"
    "encoding/json"
    "encoding/base64"
)

const (
    ProtocolUdp = 1
    ProtocolTcp = 2

    ActionAdd = 1
    ActionRemove = 2
)

type HttpControlRequest struct {
    Base64Secret string `json:"secret"`
    Destination string  `json:"destination"`
    Protocol int        `json:"protocol"`
    Action int          `json:"action"`
}

func HttpControlListen(address string) {
    fmt.Println("http listening on:", address)
    
    http.HandleFunc("/route", handle)
    http.ListenAndServe(address, nil)
}

func getSessionsForProtocol(protocol int) map[string]SessionDescription {
    switch protocol {
        case ProtocolUdp:
            return udpSessions
        case ProtocolTcp:
            return tcpSessions
        default:
            return nil
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
    error := decoder.Decode(&params)

    if error != nil {
        http.Error(writer, "failed to decode json payload", 400)
        return
    }

    secretData, error := base64.StdEncoding.DecodeString(params.Base64Secret)
    if error != nil {
        http.Error(writer, "invalid secret encoding", 400)
        return
    }

    if len(secretData) != SecretLength {
        http.Error(writer, "invalid secret length", 400)
        return
    }

    sessions := getSessionsForProtocol(params.Protocol)
    if sessions == nil {
        http.Error(writer, "invalid protocol", 400)
        return   
    }

    secret := string(secretData)

    switch params.Action {
        case ActionAdd:
            fmt.Println("adding route to", params.Destination, "with secret", secret)
            sessions[secret] = SessionDescription { secret, params.Destination }
        case ActionRemove:
            fmt.Println("removing route with the secret", secret)
            delete(sessions, secret)
        default:
            http.Error(writer, "invalid action", 400)
    }

    fmt.Fprintln(writer, `{ "status": "foxies are awesome :3" }`)
}  