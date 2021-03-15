package host

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/antonskwr/nat-punch-through-client/util"
)

func StartServer(lAddr net.TCPAddr) {
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