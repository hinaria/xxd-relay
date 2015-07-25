package main

import (
	"relay"
	"strconv"
)

var (
	HttpListenAddress = "0.0.0.0:9000"

	TcpListenPort = 9005
	UdpListenPort = 9006

	TcpListenAddress = "0.0.0.0:" + strconv.Itoa(TcpListenPort)
	UdpListenAddress = "0.0.0.0:" + strconv.Itoa(UdpListenPort)

	SecretLength = 32
)

func sleep() {
	select {}
}

func main() {
	go relay.HttpControlListen(HttpListenAddress)
	go relay.TcpListen(TcpListenAddress)
	go relay.UdpListen(UdpListenAddress)
	sleep()
}
