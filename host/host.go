package host

import (
	"fmt"
	"net"
	"strings"

	"github.com/antonskwr/nat-punch-through-client/reuseport"
	"github.com/antonskwr/nat-punch-through-client/util"
)

func connCloseHandler(conn net.PacketConn, quit <-chan int) {
	<-quit
	conn.Close()
}

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
	go connCloseHandler(conn, quit)
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

			if err != nil {
				util.HandleErrNonFatal(err, "Server will stop listening")
				return
			}

			trimmedMsg := strings.TrimSpace(string(msgBuffer[0:n]))
			fmt.Printf("Server: %s -> %s\n", addr.String(), trimmedMsg)

			resp := handleMsgFromPacket(trimmedMsg, &addr)
			_, err = conn.WriteTo([]byte(resp), addr) // TODO(antonskwr): handle the number of bytes

			if err != nil {
				util.HandleErrNonFatal(err)
				continue // TODO(antonskwr): consider returning here
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
