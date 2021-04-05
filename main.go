package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/antonskwr/nat-punch-through-client/host"
	"github.com/antonskwr/nat-punch-through-client/hubclient"
	"github.com/antonskwr/nat-punch-through-client/util"
)

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

func HubInvalidOption(ctx hubclient.ClientContext, stdinChan <-chan string) {
	fmt.Println("Invalid option")
}

func promptAddr(stdinChan <-chan string) string {
	fmt.Printf("enter address: ")
	sAddr := <-stdinChan
	trimmedAddr := strings.TrimSpace(sAddr)
	return trimmedAddr
}

func promptPort(stdinChan <-chan string) (int, error) {
	fmt.Printf("enter port: ")
	sPort := <-stdinChan
	port, err := strconv.Atoi(strings.TrimSpace(sPort))
	if err != nil {
		return 0, err
	}
	return port, nil
}

func promptUser(stdinChan <-chan string) hubclient.HubFunc {
	fmt.Println("What would you like to do?")
	fmt.Println("(s)tart server at [port]")
	fmt.Println("s(t)op server")
	fmt.Println("(d)ial hub at [hostport] from [port]")
	fmt.Printf("-> ")
	input := <-stdinChan
	trimedInput := strings.TrimSpace(input)

	switch trimedInput {
	case "s":
		return func(ctx hubclient.ClientContext, stdinChan <-chan string) {
			port, err := promptPort(stdinChan)
			if err != nil {
				util.HandleErrNonFatal(err)
				return
			}
			host.StartUDPServer(port, *ctx.ServerQuitChan)
		}
	case "t":
		return func(ctx hubclient.ClientContext, stdinChan <-chan string) {
			(*ctx.ServerQuitChan) <- 0
		}
	case "d":
		return func(ctx hubclient.ClientContext, stdinChan <-chan string) {
			hostport := promptAddr(stdinChan)
			if hostport == "" {
				err := fmt.Errorf("hostport is empty")
				util.HandleErrNonFatal(err)
				return
			}
			port, err := promptPort(stdinChan)
			if err != nil {
				util.HandleErrNonFatal(err)
				return
			}
			completionHandler := hubclient.DialHubUDP(hostport, port, "Hub", stdinChan)
			completionHandler(stdinChan)
		}
	default:
		return HubInvalidOption
	}
}

func main() {
	serverQuitChan := make(chan int)
	stdinChan := make(chan string) // NOTE(antonskwr): don't put into goroutines, other than ReadFromStdin
	stdinAbortChan := make(chan int)
	go ReadFromStdin(stdinChan, stdinAbortChan)

	clientCtx := hubclient.ClientContext{
		ServerQuitChan: &serverQuitChan,
	}

	for {
		hubFunc := promptUser(stdinChan)
		hubFunc(clientCtx, stdinChan)
		util.PrintSeparator()
	}
}
