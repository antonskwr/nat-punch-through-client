package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

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

type CompletionHadler func(<-chan string)

func CompletionHadlerNone(stdinChan <-chan string) {}

type HubFunc func(ClientContext, <-chan string)

func ReadMsgFromConn(conn net.Conn, c chan []byte, abortChan <-chan int) {
	for {
		select {
		case <-abortChan:
			return
		default:
			incomingBuffer := make([]byte, 1024)
			n, err := conn.Read(incomingBuffer)
			if err != nil {
				util.HandleErrNonFatal(err, "Closing UDP Conn")
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

func DialHubUDP(hostport string, localPort int, targetName string, stdinChan <-chan string) CompletionHadler {
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

	serverMsgChan := make(chan []byte)
	abortChan := make(chan int, 1)

	go ReadMsgFromConn(conn, serverMsgChan, abortChan)

	for {
		select {
		case receivedBuffer := <-serverMsgChan:
			trimmedBuffer := strings.TrimSpace(string(receivedBuffer))
			fmt.Printf("%s reply: %s\n", targetName, trimmedBuffer)

			resp := handleMsgFromPacket(trimmedBuffer)
			if resp.rType == RespTypeOk || resp.rType == RespTypeReq {
				// TODO(antonskwr) intorduce some kind of timeout for ping procedure
				peerConn, err := PingUDP(resp.payload, localPort)
				if err != nil {

				}

				switch resp.rType {
				case RespTypeOk:
					abortChan <- 0
					return func(stdinCh <-chan string) {
						host.StartChatOnConnection(peerConn, stdinCh)
					}
				case RespTypeReq:
					abortChan <- 0
					return func(stdinCh <-chan string) {
						host.StartChatOnConnection(peerConn, stdinCh)
					}
				}
			}
		case userMsg := <-stdinChan:
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
}

func PingUDP(hostport string, localPort int) (net.Conn, error) {
	lAddr := net.UDPAddr{}
	lAddr.Port = localPort

	conn, err := reuseport.Dial("udp", lAddr.String(), hostport)
	if err != nil {
		return nil, err
	}

	// defer conn.Close()

	fmt.Println("PingUDP dial successful")

	pingData := []byte("ping")
	pingChan := make(chan int)

	go ReadFromPingConn(conn, pingChan)

	// TODO(antonskwr): implement timeout (close connection, return custom error)
InnerLoop:
	for {
		select {
		case respCode := <-pingChan:
			if respCode == 1 {
				// connection successful
				break InnerLoop
			}
		default:
			_, err = conn.Write(pingData)
			if err != nil {
				util.HandleErrNonFatal(err, "PingUDP write error")
			}
			fmt.Println("PingUDP will retry write...")
			time.Sleep(500 * time.Millisecond)
		}
	}

	return conn, nil
}

func ReadFromPingConn(conn net.Conn, c chan int) {
	for {
		incomingBuffer := make([]byte, 5)
		_, err := conn.Read(incomingBuffer) // NOTE(antonskwr): might be blocking
		if err != nil {
			util.HandleErrNonFatal(err, "ReadFromPingConn error")
			continue
		}

		c <- 1
		return
	}
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

func HubInvalidOption(ctx ClientContext, stdinChan <-chan string) {
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

func promptUser(stdinChan <-chan string) HubFunc {
	fmt.Println("What would you like to do?")
	fmt.Println("(s)tart server at [port]")
	fmt.Println("s(t)op server")
	fmt.Println("(d)ial hub at [hostport] from [port]")
	fmt.Printf("-> ")
	input := <-stdinChan
	trimedInput := strings.TrimSpace(input)

	switch trimedInput {
	case "s":
		return func(ctx ClientContext, stdinChan <-chan string) {
			port, err := promptPort(stdinChan)
			if err != nil {
				util.HandleErrNonFatal(err)
				return
			}
			host.StartUDPServer(port, *ctx.ServerQuitChan)
		}
	case "t":
		return func(ctx ClientContext, stdinChan <-chan string) {
			(*ctx.ServerQuitChan) <- 0
		}
	case "d":
		return func(ctx ClientContext, stdinChan <-chan string) {
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
			completionHandler := DialHubUDP(hostport, port, "Hub", stdinChan)
			completionHandler(stdinChan)
		}
	default:
		return HubInvalidOption
	}
}

type ClientContext struct {
	ServerQuitChan *chan int
}

func main() {
	serverQuitChan := make(chan int)
	stdinChan := make(chan string) // NOTE(antonskwr): don't put into goroutines, other than ReadFromStdin
	stdinAbortChan := make(chan int)
	go ReadFromStdin(stdinChan, stdinAbortChan)

	clientCtx := ClientContext{
		&serverQuitChan,
	}

	for {
		hubFunc := promptUser(stdinChan)
		hubFunc(clientCtx, stdinChan)
		util.PrintSeparator()
	}
}
