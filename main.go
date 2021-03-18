package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/antonskwr/nat-punch-through-client/host"
	"github.com/antonskwr/nat-punch-through-client/reuseport"
	"github.com/antonskwr/nat-punch-through-client/util"
)

func GetRemoteTCPAddress() *net.TCPAddr {
	addr := net.TCPAddr{}
	addr.IP = net.ParseIP("127.0.0.1")
	addr.Port = 8080
	return &addr
}

func GetRemoteUDPAddress() *net.UDPAddr {
	addr := net.UDPAddr{}
	addr.IP = net.ParseIP("127.0.0.1")
	addr.Port = 8080
	return &addr
}

func DialHubTCP() {
	lAddr := net.TCPAddr{}
	lAddr.Port = 9000

	hubAddr := GetRemoteTCPAddress()

	conn, err := net.DialTCP("tcp", &lAddr, hubAddr)
	if err != nil {
		util.HandleErr(err)
		return
	}

	conn.SetLinger(0) // NOTE(antonskwr): close connection immediately after Close() called
	defer conn.Close()

	fmt.Printf("TCP: Successfully connected to Hub at %s\n", conn.RemoteAddr().String())
	fmt.Println("Type <STOP> to close the connection")

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(">> ")
		text, err := reader.ReadString('\n') // NOTE(antonskwr): this line is blocking

		if err != nil {
			util.HandleErr(err)
			continue
		}

		fmt.Fprintf(conn, text+"\n") // NOTE(antonskwr): flust text down the connection

		// NOTE(antonskwr): server response handling
		message, err := bufio.NewReader(conn).ReadString('\n') // NOTE(antonskwr): blocking

		if err != nil {
			util.HandleErr(err)
			continue
		}

		fmt.Print("->: " + message)
		if strings.TrimSpace(string(text)) == "STOP" {
			fmt.Println("TCP client exiting...")
			break
		}
	}
}

func DialHubUDP(hostport string, localPort int, name string) {
	lAddr := net.UDPAddr{}
	lAddr.Port = localPort

	conn, err := reuseport.Dial("udp", lAddr.String(), hostport)
	if err != nil {
		util.HandleErr(err)
		return
	}

	defer conn.Close()

	fmt.Printf("UDP: Successfully connected to %s at %s\n", name, conn.RemoteAddr().String())
	fmt.Println("Type <STOP> to close the connection")
	fmt.Println("Type <LIST> to list available hosts")
	fmt.Println("Type <ADD [id]> to register host with id")
	fmt.Println("Type <CONN [id]> to connect to host with id")
	fmt.Println("Type <PULL> to pull msgs from connection (debug only)")

	var interruptResp Resp

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(">> ")
		text, err := reader.ReadString('\n') // NOTE(antonskwr): this line is blocking
		// NOTE(antonskwr): ReadString() returns string with the delimeter included

		if err != nil {
			util.HandleErr(err)
			continue
		}

		if strings.TrimSpace(text) == "STOP" {
			fmt.Println("UDP client exiting...")
			return
		}

		if strings.TrimSpace(text) != "PULL" {
			msgData := []byte(text)      // NOTE(antonskwr): flush text down the connection
			_, err = conn.Write(msgData) // TODO(antonskwr): handle number of bytes written

			if err != nil {
				util.HandleErr(err)
				continue
			}
		}

		incomingBuffer := make([]byte, 1024)
		n, err := conn.Read(incomingBuffer)
		if err != nil {
			util.HandleErr(err)
			continue
		}

		trimmedBuffer := strings.TrimSpace(string(incomingBuffer[0:n]))
		fmt.Printf("HUB reply: %s\n", trimmedBuffer)

		resp := handleMsgFromPacket(trimmedBuffer)
		if resp.rType != RespTypeContinue {
			interruptResp = resp
			break
		}
	}

	switch interruptResp.rType {
	case RespTypeOk:
		for !PingUDP(interruptResp.payload, localPort) {
		}
		DialHubUDP(interruptResp.payload, localPort, "host")
	case RespTypeReq:
		for !PingUDP(interruptResp.payload, localPort) {
		}
		fmt.Println("Was able to connect to client")
	}
}

func PingUDP(hostport string, localPort int) bool {
	lAddr := net.UDPAddr{}
	lAddr.Port = localPort

	conn, err := reuseport.Dial("udp", lAddr.String(), hostport)
	if err != nil {
		util.HandleErr(err)
		return false
	}

	defer conn.Close()

	fmt.Println("PingUDP connection successful")

	return true
}

type RespType int

const (
	RespTypeOk RespType = iota
	RespTypeReq
	RespTypeContinue
)

type Resp struct {
	rType   RespType
	payload string
}

func handleMsgFromPacket(msg string) Resp {
	splittedMsgs := strings.Split(msg, " ")

	if len(splittedMsgs) == 2 {
		if splittedMsgs[0] == "OK" {
			hostport := splittedMsgs[1]
			host, port, err := net.SplitHostPort(hostport)
			if err != nil {
				util.HandleErr(err, "failed to parse hostport %s", hostport)
				return Resp{RespTypeContinue, ""}
			}

			if len(host) != 0 && len(port) != 0 {
				return Resp{RespTypeOk, hostport}
			}
		}
		if splittedMsgs[0] == "REQ" {
			hostport := splittedMsgs[1]
			host, port, err := net.SplitHostPort(hostport)
			if err != nil {
				util.HandleErr(err, "failed to parse hostport %s", hostport)
				return Resp{RespTypeContinue, ""}
			}

			if len(host) != 0 && len(port) != 0 {
				return Resp{RespTypeReq, hostport}
			}
		}
	}

	return Resp{RespTypeContinue, ""}
}

func HubInvalidOption() {
	fmt.Println("Invalid option")
}

func promptAddr() (string, error) {
	fmt.Printf("enter address: ")
	var addr string
	_, err := fmt.Scanf("%s\n", &addr)
	if err != nil {
		return "", err
	}
	return addr, nil
}

func promptPort() (int, error) {
	fmt.Printf("enter port: ")
	var port int
	_, err := fmt.Scanf("%d\n", &port)
	if err != nil {
		return 0, err
	}
	return port, nil
}

func promptUser() {
	var input string

	fmt.Println("What would you like to do?")
	fmt.Println("(s)tart server at [port]")
	fmt.Println("(d)ial hub at [hostport] from [port]")
	// fmt.Println("(di)al hub at [hostname] from [port]")
	fmt.Printf("-> ")
	fmt.Scanf("%s\n", &input)

	switch input {
	case "s":
		port, err := promptPort()
		if err != nil {
			util.HandleErr(err)
			return
		}
		host.StartUDPServer(port)
	case "d":
		hostport, err := promptAddr()
		if err != nil {
			util.HandleErr(err)
			return
		}
		port, err := promptPort()
		if err != nil {
			util.HandleErr(err)
			return
		}
		DialHubUDP(hostport, port, "Hub")
		// case "di":
		// hostport, err := promptAddr()
		// if err != nil {
		// 	util.HandleErr(err)
		// 	return
		// }
		// // TODO: lookup ip
		// port, err := promptPort()
		// if err != nil {
		// 	util.HandleErr(err)
		// 	return
		// }
		// DialHubUDP(hostport, port)
	default:
		fmt.Println("Invalid option")
	}
}

func main() {
	for {
		promptUser()
		util.PrintSeparator()
	}
}
