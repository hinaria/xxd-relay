package relay

import (
    "fmt"
    "net"
    "time"
)

const (
    BufferLength = 4096
)

var (
    ReadNetworkTimeout = 10 * time.Minute
    WriteNetworkTimeout = 10 * time.Second
)

func TcpListen(address string) {
    fmt.Println("tcp listening on:", address)

    listener, err := net.Listen("tcp", address)
    if err != nil {
        fmt.Println("couldn't listen on tcp:", err.Error())
        return
    }

    defer listener.Close()

    for {
        connection, err := listener.Accept()
        if err != nil {
            fmt.Println("couldn't accept incoming tcp connection:", err.Error())
            return
        }

        go tcp(connection)
    }
}

func tcp(client net.Conn) {
    remote := client.RemoteAddr()

    session := tcpGrabSession(client)
    if session == nil {
        fmt.Println(remote, "- client presented invalid secret")
        client.Close()
        return
    }
    
    fmt.Println(remote, "- authenticated. relaying to", session.DestinationAddress)

    server, err := net.Dial("tcp", session.DestinationAddress)
    if err != nil {
        fmt.Println(remote, "- couldn't connect to remote server:", err.Error())
        client.Close()
        return
    }

    fmt.Println(remote, "- connected to both parties. beginning relay.")

    go tcpStreamCopy(client, server)
    go tcpStreamCopy(server, client)
}

func tcpGrabSession(connection net.Conn) *SessionDescription {
    buffer := make([]byte, SecretLength)
    remote := connection.RemoteAddr()

    bytes, err := connection.Read(buffer)
    
    if err != nil {
        fmt.Println(remote, "- couldn't read secret:", err.Error())
        return nil
    }

    if bytes != SecretLength {
        fmt.Printf("%s - expected first tcp segment to a secret (%d bytes). instead, received %d bytes.\n", remote, SecretLength, bytes)
        return nil
    }
    key := string(buffer)
    session, exists := tcpSessions[key]
    
    if exists {
        delete(tcpSessions, key)
        return &session
    }

    return nil
}

func tcpStreamCopy(from net.Conn, to net.Conn) {
    buffer := make([]byte, BufferLength)

    defer from.Close()
    defer to.Close()

    for {
        from.SetReadDeadline(time.Now().Add(ReadNetworkTimeout))

        total, err := from.Read(buffer)
        if (err != nil) {
            fmt.Println(from.RemoteAddr(), "<->", to.RemoteAddr(), "- tcp stream read failed:", err.Error())
            return
        }

        written := 0
        for written < total {
            to.SetWriteDeadline(time.Now().Add(WriteNetworkTimeout))

            bytes, err := to.Write(buffer[written:total])
            if (err != nil) {
                fmt.Println(to.RemoteAddr(), "<->", from.RemoteAddr(), "- tcp stream write failed:", err.Error())
                return
            }

            written += bytes
        }
    }
}