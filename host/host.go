package host

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/antonskwr/nat-punch-through-client/util"

	reuse "github.com/libp2p/go-reuseport"
)

func StartUDPServer(port int) {
	udpAddr := net.UDPAddr{}
	udpAddr.Port = port

	conn, connErr := reuse.ListenPacket("udp", udpAddr.String())
	if connErr != nil {
		util.HandleErrFatal(connErr)
		return
	}

	fmt.Printf("Server: Started UDP server at %s\n", conn.LocalAddr().String())

	go handleConnUDP(conn)
}

func handleConnUDP(conn net.PacketConn) {
	defer conn.Close()

	msgBuffer := make([]byte, 7)

	for {
		n, addr, err := conn.ReadFrom(msgBuffer)
		trimmedMsg := strings.TrimSpace(string(msgBuffer[0:n]))
		fmt.Printf("Server: %s -> %s\n", addr.String(), trimmedMsg)

		if trimmedMsg == "STOP" {
			fmt.Println("Server: Exiting UDP server!")
			return
		}

		if err != nil {
			util.HandleErr(err)
			continue
		}

		resp := handleMsgFromPacket(trimmedMsg, &addr)
		_, err = conn.WriteTo([]byte(resp), addr) // TODO(antonskwr): handle the number of bytes

		if err != nil {
			util.HandleErr(err)
			continue
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

func StartTCPServer(lAddr net.TCPAddr) {
	tcpListener, err := net.ListenTCP("tcp", &lAddr)

	if err != nil {
		util.HandleErr(err)
		return
	}

	defer tcpListener.Close()

	for {
		conn, connErr := tcpListener.AcceptTCP()

		if connErr != nil {
			util.HandleErr(connErr)
			continue
		}

		hAddr := conn.LocalAddr().String()
		gAddr := conn.RemoteAddr().String()

		fmt.Printf("Host: New connection.\nhost address: %s\nguest address: %s\n", hAddr, gAddr)
		util.PrintSeparator()

		go handleTCPConn(conn)
	}
}

func handleTCPConn(conn *net.TCPConn) {
	for {
		// NOTE (antonskwr): in next line host waits for guest to write something to connection
		data, err := bufio.NewReader(conn).ReadString('\n') // NOTE (antonskwr): blocking
		if err != nil {
			if err == io.EOF {
				fmt.Printf("HOST: peer at %s disconnected\n", conn.RemoteAddr().String())
				util.PrintSeparator()
				break
			}
			util.HandleErr(err)
			continue
		}

		if strings.TrimSpace(string(data)) == "STOP" {
			fmt.Printf("HOST: closing connection for peer at %s\n", conn.RemoteAddr().String())
			util.PrintSeparator()
			break
		}

		fmt.Print("-> ", string(data))
		t := time.Now()
		hostTime := t.Format(time.RFC3339) + "\n"

		conn.Write([]byte(hostTime))
	}

	conn.Close()
}
