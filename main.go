package main

import (
	"fmt"
	"net"

	"github.com/antonskwr/nat-punch-through-client/util"
)

type HubFunc func(Context)

type Context struct {
	Id uint32
}

func GetAddress() string {
	return "127.0.0.1:8080"
}

func HubPing(ctx Context) {
	conn, err := net.Dial("tcp", GetAddress())
	if err != nil {
		util.HandleErr(err)
		return
	}

	defer conn.Close()

	localAddr := conn.LocalAddr().String()

	fmt.Println("localAddr:", localAddr)
}

func HubInvalidOption(ctx Context) {
	fmt.Println("Invalid option")
}

func promptUser() (HubFunc, Context) {
	var input string

	fmt.Println("What would you like to do? (p)ing")
	emptyContext := Context{}

	fmt.Scanf("%s\n", &input)
	switch input {
	case "p":
		return HubPing, emptyContext
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
		return HubInvalidOption, emptyContext
	}
}

func printSeparator() {
	fmt.Printf("=========\n\n")
}

func main() {
	for {
		hubFunc, ctx := promptUser()
		hubFunc(ctx)
		printSeparator()
	}
}
