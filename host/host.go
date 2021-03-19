package host

import (
	"fmt"
	"net"
	"strings"

	"github.com/antonskwr/nat-punch-through-client/reuseport"
	"github.com/antonskwr/nat-punch-through-client/util"
)

func StartUDPServer(port int, quit <-chan int) {
	udpAddr := net.UDPAddr{}
	udpAddr.Port = port

	conn, connErr := reuseport.ListenPacket("udp", udpAddr.String())
	if connErr != nil {
		util.HandleErrFatal(connErr)
		return
	}

	fmt.Printf("Server: Started UDP server at %s\n", conn.LocalAddr().String())

	go handleConnUDP(conn, quit)
}

func handleConnUDP(conn net.PacketConn, quit <-chan int) {
	defer conn.Close()

	msgBuffer := make([]byte, 32)

	for {
		select {
		case <-quit:
			fmt.Println("Server: Exiting UDP server!")
			return
		default:
			n, addr, err := conn.ReadFrom(msgBuffer)
			trimmedMsg := strings.TrimSpace(string(msgBuffer[0:n]))
			fmt.Printf("Server: %s -> %s\n", addr.String(), trimmedMsg)

			if trimmedMsg == "STOP" {
				fmt.Println("Server: Exiting UDP server!")
				return
			}

			if err != nil {
				util.HandleErrNonFatal(err)
				continue
			}

			resp := handleMsgFromPacket(trimmedMsg, &addr)
			_, err = conn.WriteTo([]byte(resp), addr) // TODO(antonskwr): handle the number of bytes

			if err != nil {
				util.HandleErrNonFatal(err)
				continue
			}
		}
	}
}

func handleMsgFromPacket(msg string, addr *net.Addr) string {
	resp := "Unknown msg"

	if msg == "TEST" {
		resp = "test 1, 2"
	}

	return resp
}
