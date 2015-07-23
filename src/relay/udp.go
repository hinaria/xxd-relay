package relay

import (
"fmt"
"net"
"time"
"sync"
)

type LiveSession struct {
    Destination1 string
    Destination2 string
    Created time.Time
}

const (
    MaxPacketLength = 4096
    InitialSessionQueueCapacity = 100
    InitialSessionCapacity = 1000
)

var (
    UdpReadNetworkTimeout = 10 * time.Second
    UdpWriteNetworkTimeout = 10 * time.Second

    UdpAssociationDuration = time.Duration(time.Hour)

    // ip_address string <-> ip_address UDPAddr
    associations = make(map[string]net.UDPAddr, InitialSessionCapacity)

    // the sessions list that our listener adds to
    sessions = make([]LiveSession, 0, InitialSessionQueueCapacity)
    // we move sessions from `sessions` into here, and then walk through this
    // list to limit the amount of time we spend locking `sessions`
    _sessions = make([]LiveSession, 0, InitialSessionCapacity)

    associationsLock = sync.Mutex{}
    sessionsLock = sync.Mutex{}
)

func UdpListen(address string) {
    go invalidator()
    listener(address)
}

func invalidator() {
    const TrimDurationMinutes = 10

    fmt.Println("beginning udp route invaldation loop")

    for {
        // wait until `TrimDurationMinutes` before trimming sessions, but
        // moving `sessions` to `_sessions`
        for i := 0; i < TrimDurationMinutes; i++ {
            time.Sleep(time.Minute)

            sessionsLock.Lock()
            if len(sessions) > 0 {
                fmt.Println("moving", len(sessions), "to the invalidation queue")
                _sessions = append(_sessions, sessions...)
                sessions = sessions[:0]
            }
            sessionsLock.Unlock()
        }

        // `TrimDurationMinutes` have passed. trim sessions slowly.

        previous := len(_sessions)
        count := 0
        for i, session := range _sessions {
            if (time.Since(session.Created) < UdpAssociationDuration) {
                _sessions[count] = session
                count++
            }

            if (i % 1000 == 0) {
                time.Sleep(10 * time.Millisecond)
            }
        }

        fmt.Println("trimmed session list. we now have", count, "sessions. previously", previous)
        _sessions = _sessions[:count]
    }
}

func listener(address string) {
    fmt.Println("udp listening on:", address)

    listenAddress, err := net.ResolveUDPAddr("udp", address)
    if err != nil {
        fmt.Println("couldn't parse address", address, "-", err.Error())
    }

    listener, err := net.ListenUDP("udp", listenAddress)
    if err != nil {
        fmt.Println("couldn't listen on udp:", err.Error())
        return
    }

    defer listener.Close()

    buffer := make([]byte, BufferLength)
    for {
        listener.SetReadDeadline(time.Now().Add(UdpReadNetworkTimeout))
        bytes, from, err := listener.ReadFromUDP(buffer)
        if err != nil {
            if netError, ok := err.(net.Error); ok && netError.Timeout() {
                continue
            }

            fmt.Println("couldn't read from listener:", err.Error())
            continue
        }

        associationsLock.Lock()
        to, hasAssociation := associations[from.String()]
        associationsLock.Unlock()

        if hasAssociation {
            listener.SetWriteDeadline(time.Now().Add(UdpWriteNetworkTimeout))
            go listener.WriteToUDP(buffer[:bytes], &to)
        } else if bytes == SecretLength {
            fmt.Println(from, "- no existing association, but received a potential secret")
            
            secret := string(buffer[:bytes])
            udpSessionsLock.Lock()
            session, exists := udpSessions[secret]
            udpSessionsLock.Unlock()
            
            if exists {
                fmt.Println(from, "- secret matched pending session")
                to, err := net.ResolveUDPAddr("udp", session.Destination)
                if err != nil {
                    fmt.Println("couldn't parse udp destination address", session.Destination, "-", err.Error())
                    continue
                }

                fromString := from.String()
                toString := to.String()

                associationsLock.Lock()
                associations[fromString] = *to
                associations[toString] = *from
                associationsLock.Unlock()

                sessionsLock.Lock()
                sessions = append(sessions, LiveSession { fromString, toString, time.Now() })
                sessionsLock.Unlock()

                // echo back the secret
                listener.SetWriteDeadline(time.Now().Add(UdpWriteNetworkTimeout))
                go listener.WriteToUDP(buffer[:bytes], from)
            } else {
                fmt.Println(from, "- active session found for this address")
            }
        } else {
            fmt.Println(from, "- active session found for this address")
        }
    }
}