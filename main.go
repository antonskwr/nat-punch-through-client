package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/antonskwr/nat-punch-through-client/util"
)

func GetRemoteAddress() *net.TCPAddr {
	addr := net.TCPAddr{}
	addr.IP = net.ParseIP("127.0.0.1")
	addr.Port = 8080
	return &addr
}

func DialHubTCP() {
	lAddr := net.TCPAddr{}
	lAddr.Port = 9000

	hubAddr := GetRemoteAddress()

	conn, err := net.DialTCP("tcp", &lAddr, hubAddr)
	if err != nil {
		util.HandleErr(err)
		return
	}

	fmt.Printf("Successfully connected to Hub at: %s\n", hubAddr.String())
	fmt.Println("Type <STOP> to close the connection")

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(">> ")
		text, _ := reader.ReadString('\n') // NOTE(antonskwr): this line is blocking
		fmt.Fprintf(conn, text+"\n") // NOTE(antonskwr): flust text down the connection

		// NOTE(antonskwr): server response handling
		message, _ := bufio.NewReader(conn).ReadString('\n') // NOTE(antonskwr): blocking
		fmt.Print("->: " + message)
		if strings.TrimSpace(string(text)) == "STOP" {
			fmt.Println("TCP client exiting...")
			break
		}
	}

	conn.SetLinger(0) // NOTE(antonskwr): close connection immediately
	conn.Close()
}

func HubInvalidOption() {
	fmt.Println("Invalid option")
}

func promptUser() {
	var input string

	fmt.Println("What would you like to do? dial (t)cp")
	fmt.Scanf("%s\n", &input)

	switch input {
	case "t":
		DialHubTCP()
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
