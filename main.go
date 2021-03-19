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

type CompletionHadler func()

func CompletionHadlerNone() {}

type HubFunc func(ClientContext)

func ReadMsgFromConn(conn net.Conn, c chan []byte, abortChan <-chan int) {
	for {
		select {
		case <-abortChan:
			return
		default:
			incomingBuffer := make([]byte, 1024)
			n, err := conn.Read(incomingBuffer)
			if err != nil {
				util.HandleErrNonFatal(err)
				continue
			}

			c <- incomingBuffer[:n]
		}
	}
}

func PromptUserMsg(conn net.Conn, c chan string, abortChan <-chan int) {
	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-abortChan:
			return
		default:
			fmt.Print(">> ")
			text, err := reader.ReadString('\n')

			if err != nil {
				util.HandleErrNonFatal(err)
				continue
			}

			c <- text
		}
	}
}

func DialHubUDP(hostport string, localPort int, targetName string) CompletionHadler {
	lAddr := net.UDPAddr{}
	lAddr.Port = localPort

	conn, err := reuseport.Dial("udp", lAddr.String(), hostport)
	if err != nil {
		util.HandleErrNonFatal(err)
		return CompletionHadlerNone
	}

	defer conn.Close()

	fmt.Printf("UDP: Successfully dialed %s at %s\n", targetName, conn.RemoteAddr().String())
	fmt.Println("Type <STOP> to close the connection")
	fmt.Println("Type <LIST> to list available hosts")
	fmt.Println("Type <ADD [id]> to register host with id")
	fmt.Println("Type <JOIN [id]> to connect to host with id")

	userPromptChan := make(chan string)
	serverMsgChan := make(chan []byte)
	abortChan := make(chan int)

	go PromptUserMsg(conn, userPromptChan, abortChan)
	go ReadMsgFromConn(conn, serverMsgChan, abortChan)

	for {
		select {
		case receivedBuffer := <-serverMsgChan:
			trimmedBuffer := strings.TrimSpace(string(receivedBuffer))
			fmt.Printf("%s reply: %s\n", targetName, trimmedBuffer)

			resp := handleMsgFromPacket(trimmedBuffer)
			if resp.rType == RespTypeOk || resp.rType == RespTypeReq {
				// TODO(antonskwr) intorduce some kind of timeout for ping procedure
			InnerLoop:
				for {
					err = PingUDP(resp.payload, localPort)
					if err == nil {
						break InnerLoop
					}
					// TODO(antonskwr): sleep
				}

				switch resp.rType {
				case RespTypeOk:
					abortChan <- 0
					return func() {
						DialHubUDP(resp.payload, localPort, "Host Server")
					}
				case RespTypeReq:
				}
			}
		case userMsg := <-userPromptChan:
			if strings.TrimSpace(userMsg) == "STOP" {
				fmt.Println("UDP client exiting...")
				abortChan <- 0
				return CompletionHadlerNone
			}

			msgData := []byte(userMsg)   // NOTE(antonskwr): flush text down the connection
			_, err = conn.Write(msgData) // TODO(antonskwr): handle number of bytes written

			if err != nil {
				util.HandleErrNonFatal(err)
				continue
			}
		}
	}

	return CompletionHadlerNone
}

func PingUDP(hostport string, localPort int) error {
	lAddr := net.UDPAddr{}
	lAddr.Port = localPort

	conn, err := reuseport.Dial("udp", lAddr.String(), hostport)
	if err != nil {
		return err
	}

	defer conn.Close()

	fmt.Println("PingUDP dial successful")

	pingData := []byte("ping")
	_, err = conn.Write(pingData)
	if err != nil {
		return err
	}

	incomingBuffer := make([]byte, 5)
	_, err = conn.Read(incomingBuffer)
	if err != nil {
		return err
	}

	return nil
}

func handleMsgFromPacket(msg string) Resp {
	splittedMsgs := strings.Split(msg, " ")

	if len(splittedMsgs) == 2 {
		if splittedMsgs[0] == "OK" {
			hostport := splittedMsgs[1]
			host, port, err := net.SplitHostPort(hostport)
			if err != nil {
				util.HandleErrNonFatal(err, "failed to parse hostport %s", hostport)
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
				util.HandleErrNonFatal(err, "failed to parse hostport %s", hostport)
				return Resp{RespTypeContinue, ""}
			}

			if len(host) != 0 && len(port) != 0 {
				return Resp{RespTypeReq, hostport}
			}
		}
	}

	return Resp{RespTypeContinue, ""}
}

func HubInvalidOption(ctx ClientContext) {
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

func promptUser() HubFunc {
	var input string

	fmt.Println("What would you like to do?")
	fmt.Println("(s)tart server at [port]")
	fmt.Println("s(t)op server")
	fmt.Println("(d)ial hub at [hostport] from [port]")
	fmt.Printf("-> ")
	fmt.Scanf("%s\n", &input)

	switch input {
	case "s":
		return func(ctx ClientContext) {
			port, err := promptPort()
			if err != nil {
				util.HandleErrNonFatal(err)
				return
			}
			host.StartUDPServer(port, *ctx.ServerQuitChan)
		}
	case "t":
		return func(ctx ClientContext) {
			(*ctx.ServerUpdateChan) <- 0
		}
	case "d":
		return func(ctx ClientContext) {
			hostport, err := promptAddr()
			if err != nil {
				util.HandleErrNonFatal(err)
				return
			}
			port, err := promptPort()
			if err != nil {
				util.HandleErrNonFatal(err)
				return
			}
			completionHandler := DialHubUDP(hostport, port, "Hub")
			completionHandler()
		}
	default:
		return HubInvalidOption
	}
}

type ClientContext struct {
	ServerUpdateChan *chan int
	ServerQuitChan   *chan int
}

func main() {
	serverQuitChan := make(chan int, 1)
	serverUpdateChan := make(chan int, 1)
	clientCtx := ClientContext{
		&serverUpdateChan,
		&serverQuitChan,
	}

	for {
		select {
		case <-(*clientCtx.ServerUpdateChan):
			(*clientCtx.ServerQuitChan) <- 0
		default:
			hubFunc := promptUser()
			hubFunc(clientCtx)
			util.PrintSeparator()
		}
	}
}
