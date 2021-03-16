package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

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

func DialHubUDP() {
	lAddr := net.UDPAddr{}
	lAddr.Port = 9000

	hubAddr := GetRemoteUDPAddress()

	conn, err := net.DialUDP("udp", &lAddr, hubAddr)
	if err != nil {
		util.HandleErr(err)
		return
	}

	defer conn.Close()

	fmt.Printf("UDP: Successfully connected to Hub at %s\n", conn.RemoteAddr().String())
	fmt.Println("Type <STOP> to close the connection")
	fmt.Println("Type <LIST> to list available hosts")
	fmt.Println("Type <ADD [id]> to register host with id")

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

		msgData := []byte(text) // NOTE(antonskwr): flush text down the connection
		_, err = conn.Write(msgData) // TODO(antonskwr): handle number of bytes written

		if err != nil {
			util.HandleErr(err)
			continue
		}

		incomingBuffer := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(incomingBuffer)
		if err != nil {
			util.HandleErr(err)
			continue
		}

		trimmedBuffer := strings.TrimSpace(string(incomingBuffer[0:n]))
		fmt.Printf("HUB reply: %s\n", trimmedBuffer)
	}
}

func HubInvalidOption() {
	fmt.Println("Invalid option")
}

func promptUser() {
	var input string

	fmt.Printf("What would you like to do? dial (t)cp, dial (u)dp: ")
	fmt.Scanf("%s\n", &input)

	switch input {
	case "t":
		DialHubTCP()
	case "u":
		DialHubUDP()
	// NOTE(antonskwr): more parameters prompting
	// case "c":
	// 	fmt.Println("Connect selected, enter server id:")
	// 	var id uint32
	// 	// TODO(antonskwr): handle negative values and bigger than max uint32
	// 	// in golang will be 0 if overflood
	// 	fmt.Scanf("%d\n", &id)
	// 	connectCtx := Context {id}
	// 	return HubConnectToServer, connectCtx
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
