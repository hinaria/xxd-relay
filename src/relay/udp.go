package relay

// import (
//     "fmt"
//     "net"
// )

// const (
//     MaxPacketLength = 4096
// )

// func UdpListen(address string) {
//     fmt.Println("udp listening on:", address)

//     listener, err = net.ListenUDP("udp", address)
//     if err != nil {
//         fmt.Println("couldn't listen on udp:", err.Error())
//         return
//     }

//     defer listener.Close()

//     buffer := make([]byte, BufferLength)
//     for {
//         bytes, from, error = listener.ReadFrom(buffer)
//         if err != nil {
//             fmt.Println("couldn't read from listener:", err.Error())
//             continue
//         }

        

//     }
// }

// func udpGrabSession(conn net.Conn) *SessionDescription {
//     buffer := make([]byte, SecretLength)

//     bytes, err := conn.Read(buffer)
    
//     if err != nil {
//         fmt.Println("couldn't read secret:", err.Error())
//         return nil
//     }

//     if bytes != SecretLength {
//         fmt.Printf("expected first udp segment to a secret (%d bytes). instead, received %d bytes.\n")
//         return nil
//     }

//     key := string(buffer)
//     session, exists := udpSessions[key]
//     if exists {
//         delete(tcpSessions, key)
//     }

//     return session
// }
// func udpStreamCopy(from net.Conn, to net.Conn) {
//     buffer := make([]byte, 4096)

//     defer from.Close()
//     defer to.Close()


//     for {
//         from.SetReadDeadline(time.Now().Add(NetworkTimeoutSeconds * time.Second))

//         total, err := from.Read(buffer)
//         if (err != nil) {
//             fmt.Println("udp stream read failed:", err.Error())
//             return
//         }

//         written := 0
//         for written < total {
//             to.SetWriteDeadline(time.Now().Add(NetworkTimeoutSeconds * time.Second))

//             bytes, err := to.Write(buffer[written:total])
//             if (err != nil) {
//                 fmt.Println("udp stream wriite failed:", err.Error())
//                 return
//             }

//             written += bytes
//         }
//     }
// }