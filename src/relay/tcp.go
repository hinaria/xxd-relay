package relay

import (
	"net"
	"time"
)

const (
	BufferLength = 4096
)

var (
	TcpReadNetworkTimeout  = 10 * time.Minute
	TcpWriteNetworkTimeout = 10 * time.Second
)

func TcpListen(address string) {
	println("tcp listening on:", address)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		println("couldn't listen on tcp:", err.Error())
		return
	}

	defer listener.Close()

	for {
		connection, err := listener.Accept()
		if err != nil {
			println("couldn't accept incoming tcp connection:", err.Error())
			return
		}

		go tcp(connection)
	}
}

func tcp(client net.Conn) {
	remote := client.RemoteAddr()

	session := tcpGrabSession(client)
	if session == nil {
		println(remote, "- client presented invalid secret")
		client.Close()
		return
	}

	println(remote, "- authenticated. relaying to", session.Destination)

	server, err := net.Dial("tcp", session.Destination)
	if err != nil {
		println(remote, "- couldn't connect to remote server:", err.Error())
		client.Close()
		return
	}

	println(remote, "- connected to both parties. beginning relay.")

	go tcpStreamCopy(client, server)
	go tcpStreamCopy(server, client)
}

func tcpGrabSession(connection net.Conn) *PendingSessionDescription {
	buffer := make([]byte, SecretLength)
	remote := connection.RemoteAddr()

	bytes, err := connection.Read(buffer)

	if err != nil {
		println(remote, "- couldn't read secret:", err.Error())
		return nil
	}

	if bytes != SecretLength {
		printf("%s - expected first tcp segment to a secret (%d bytes). instead, received %d bytes.\n", remote, SecretLength, bytes)
		return nil
	}

	key := string(buffer)

	pendingSessions.TcpLock.Lock()
	session, exists := pendingSessions.Tcp[key]
	if exists {
		delete(pendingSessions.Tcp, key)
	}
	pendingSessions.TcpLock.Unlock()

	if exists {
		return &session
	}

	return nil
}

func tcpStreamCopy(from net.Conn, to net.Conn) {
	buffer := make([]byte, BufferLength)

	defer from.Close()
	defer to.Close()

	for {
		from.SetReadDeadline(time.Now().Add(TcpReadNetworkTimeout))

		total, err := from.Read(buffer)
		if err != nil {
			println(from.RemoteAddr(), "<->", to.RemoteAddr(), "- tcp stream read failed:", err.Error())
			return
		}

		written := 0
		for written < total {
			to.SetWriteDeadline(time.Now().Add(TcpWriteNetworkTimeout))

			bytes, err := to.Write(buffer[written:total])
			if err != nil {
				println(to.RemoteAddr(), "<->", from.RemoteAddr(), "- tcp stream write failed:", err.Error())
				return
			}

			written += bytes
		}
	}
}
