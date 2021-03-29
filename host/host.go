package host

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/antonskwr/nat-punch-through-client/reuseport"
	"github.com/antonskwr/nat-punch-through-client/util"
)

func ReadMsgFromConn(conn net.Conn, c chan []byte, abortChan <-chan int) {
	for {
		select {
		case <-abortChan:
			return
		default:
			incomingBuffer := make([]byte, 1024)
			n, err := conn.Read(incomingBuffer)
			if err != nil {
				util.HandleErrNonFatal(err, "Closing UDP Chat due to error")
				return
			}

			c <- incomingBuffer[:n]
		}
	}
}

func ReadFromStdin(c chan string, abortChan <-chan int) {
	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-abortChan:
			return
		default:
			text, err := reader.ReadString('\n')

			if err != nil {
				util.HandleErrNonFatal(err)
				continue
			}

			c <- text
		}
	}
}

func StartChatOnConnection(conn net.Conn, stdinChan <-chan string) {
	fmt.Println("Starting chat with peer")
	peerMsgChan := make(chan []byte)
	abortChan := make(chan int, 1)

	defer conn.Close()

	go ReadMsgFromConn(conn, peerMsgChan, abortChan)

	for {
		select {
		case receivedBuffer := <-peerMsgChan:
			trimmedBuffer := strings.TrimSpace(string(receivedBuffer))
			fmt.Printf("%s reply: %s\n", conn.RemoteAddr().String(), trimmedBuffer)
		case userMsg := <-stdinChan:
			if strings.TrimSpace(userMsg) == "STOP" {
				fmt.Println("UDP chat exiting...")
				abortChan <- 0
				return
			}

			msgData := []byte(userMsg)    // NOTE(antonskwr): flush text down the connection
			_, err := conn.Write(msgData) // TODO(antonskwr): handle number of bytes written

			if err != nil {
				util.HandleErrNonFatal(err)
				continue
			}
		}
	}
}

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
