package relay

import (
	"bytes"
	"net"
	"sync"
	"time"
)

type LiveSession struct {
	ClientString string
	Client       *net.UDPAddr
	Peer         *net.UDPAddr
	PeerSocket   *net.UDPConn
	Created      time.Time
}

const (
	MaxPacketLength             = 4096
	InitialSessionQueueCapacity = 100
	InitialSessionCapacity      = 1000
)

var (
	UdpReadNetworkTimeout  = 10 * time.Second
	UdpWriteNetworkTimeout = 10 * time.Second

	UdpAssociationDuration = time.Duration(2 * time.Hour)

	// ip_address string <-> ip_address UDPAddr
	associations     = make(map[string]LiveSession, InitialSessionCapacity)
	associationsLock = sync.Mutex{}

	// the sessions list that our listener adds to
	sessionListByTime     = make([]LiveSession, 0, InitialSessionQueueCapacity)
	sessionListByTimeLock = sync.Mutex{}

	// we move sessions from `sessions` into here, and then walk through this
	// list to limit the amount of time we spend locking `sessions`
	_sessionListByTime = make([]LiveSession, 0, InitialSessionCapacity)

	noUdpSession = []byte{125}
)

func UdpListen(address string) {
	go invalidator()
	clientListen(address)
}

func invalidator() {
	const TrimDurationMinutes = 10

	println("beginning udp route invaldation loop")

	for {
		// wait until `TrimDurationMinutes` before trimming sessions, but
		// moving `sessions` to `_sessions`
		for i := 0; i < TrimDurationMinutes; i++ {
			time.Sleep(time.Minute)

			sessionListByTimeLock.Lock()
			if len(sessionListByTime) > 0 {
				println("moving", len(sessionListByTime), "to the invalidation queue")
				_sessionListByTime = append(_sessionListByTime, sessionListByTime...)
				sessionListByTime = sessionListByTime[:0]
			}
			sessionListByTimeLock.Unlock()
		}

		// `TrimDurationMinutes` have passed. trim sessions slowly.

		previous := len(_sessionListByTime)
		count := 0
		for i, session := range _sessionListByTime {
			if time.Since(session.Created) >= UdpAssociationDuration {
				associationsLock.Lock()
				delete(associations, session.ClientString)
				associationsLock.Unlock()
			} else {
				_sessionListByTime[count] = session
				count++
			}

			if i%1000 == 0 {
				time.Sleep(10 * time.Millisecond)
			}
		}

		println("trimmed udp session list. we now have", count, "sessions. previously", previous)
		_sessionListByTime = _sessionListByTime[:count]
	}
}

func clientListen(address string) {
	buffer := make([]byte, BufferLength)

	println("udp listening on:", address)

	listenAddress, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		println("couldn't parse address:", address, "-", err.Error())
	}

	listener, err := net.ListenUDP("udp", listenAddress)
	if err != nil {
		println("couldn't listen on udp:", err.Error())
		return
	}

	for {
		listener.SetReadDeadline(time.Now().Add(UdpReadNetworkTimeout))
		bytes, from, err := listener.ReadFromUDP(buffer)

		if err != nil {
			if netError, ok := err.(net.Error); ok && netError.Timeout() {
				continue
			}

			println("couldn't read from main client listener:", err.Error())
			continue
		}

		fromString := from.String()

		associationsLock.Lock()
		session, ok := associations[fromString]
		associationsLock.Unlock()

		if ok {
			socket := session.PeerSocket
			socket.SetWriteDeadline(time.Now().Add(UdpWriteNetworkTimeout))
			socket.WriteToUDP(buffer[:bytes], session.Peer)
		} else if bytes == SecretLength {
			println(from, "- no existing association, but received a potential secret")

			secret := string(buffer[:bytes])

			pendingSessions.UdpLock.Lock()
			pending, exists := pendingSessions.Udp[secret]
			pendingSessions.UdpLock.Unlock()

			if exists {
				println(from, "- secret matched pending session")

				to, err := net.ResolveUDPAddr("udp", pending.Destination)
				if err != nil {
					println("couldn't parse udp destination address", pending.Destination, "-", err.Error())
					continue
				}

				newSocket, err := createUdpSocket()
				if err != nil {
					println("couldn't create new socket -", err.Error())
					continue
				}

				session := LiveSession{
					ClientString: fromString,
					Client:       from,
					Peer:         to,
					PeerSocket:   newSocket,
					Created:      time.Now()}

				associationsLock.Lock()
				associations[session.ClientString] = session
				associationsLock.Unlock()

				trackSession(session)

				println(from, "- session authenticated. now relaying packets with", to.String())

				go copyServerToClient(session, listener)
			} else {
				println(from, "- potential secret did not match any pending sessions. no active session found for this address")
				listener.WriteToUDP(noUdpSession, from)
			}
		} else {
			println(from, "- no active session found for this address")
			listener.WriteToUDP(noUdpSession, from)
		}
	}
}

func trackSession(session LiveSession) {
	sessionListByTimeLock.Lock()
	sessionListByTime = append(_sessionListByTime, session)
	sessionListByTimeLock.Unlock()
}

func createUdpSocket() (*net.UDPConn, error) {
	address, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err != nil {
		return nil, err
	}

	socket, err := net.ListenUDP("udp", address)
	if err != nil {
		return nil, err
	}

	return socket, nil
}

func copyServerToClient(session LiveSession, listener *net.UDPConn) {
	buffer := make([]byte, BufferLength)
	server := session.PeerSocket
	client := listener

	for time.Since(session.Created) < UdpAssociationDuration {
		server.SetReadDeadline(time.Now().Add(UdpReadNetworkTimeout))
		data, from, err := server.ReadFromUDP(buffer)

		if err != nil {
			if netError, ok := err.(net.Error); ok && netError.Timeout() {
				continue
			}

			println("couldn't read from server socket.", err.Error())
			continue
		}

		if !bytes.Equal([]byte(from.IP), []byte(session.Peer.IP)) || from.Port != session.Peer.Port {
			println("socket received data from non-peer, ignoring. received from", from, "but only allowing from", session.Peer)
			continue
		}

		client.SetWriteDeadline(time.Now().Add(UdpWriteNetworkTimeout))
		client.WriteToUDP(buffer[:data], session.Client)
	}
}
