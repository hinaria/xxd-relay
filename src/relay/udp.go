package relay

import (
    "fmt"
    "net"
    "time"
)


const (
    MaxPacketLength = 4096
)

func UdpListen(address string) {
    fmt.Println("udp listening on:", address)

    // ip_address <-> ip_address
    associations := make(map[net.UDPAddr]net.UDPAddr)

    listenAddress, error := net.ResolveUDPAddr("udp", address)
    if error != nil {
        fmt.Println("couldn't parse address", address, "-", error.Error())
    }

    listener, error := net.ListenUDP("udp", listenAddress)
    if error != nil {
        fmt.Println("couldn't listen on udp:", error.Error())
        return
    }

    defer listener.Close()

    buffer := make([]byte, BufferLength)
    for {
        bytes, from, error := listener.ReadFromUDP(buffer)
        if error != nil {
            fmt.Println("couldn't read from listener:", error.Error())
            continue
        }

        to, hasAssociation := associations[from]
        if hasAssociation {
            // todo: add last write time
            listener.SetWriteDeadline(time.Now().Add(WriteNetworkTimeout))
            go listener.WriteToUDP(buffer[:bytes], to)
        } else if bytes == SecretLength {
            fmt.Println(from, "- existing association, but received a secret")
            
            secret := string(buffer)
            session, exists := udpSessions[secret]
            
            if exists {
                fmt.Println(from, "- matched session", secret)
                destination, error := net.ResolveUDPAddr("udp", session.Destination)
                if error != nil {
                    fmt.Println("couldn't parse udp destination address", session.Destination, "-", error.Error())
                    continue
                }

                associations[from] = destination
                associations[destination] = from

                // echo back the secret
                listener.SetWriteDeadline(time.Now().Add(WriteNetworkTimeout))
                go listener.WriteToUDP(buffer[:bytes], from)
            } else {
                fmt.Println(from, "- no session found for this address", secret)
            }
        }
    }
}