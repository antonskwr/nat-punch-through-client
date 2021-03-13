package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type HubFunc func(Context)

type Context struct {
	id uint32
}

func GetBasePath() string {
	return "http://127.0.0.1:8080/api/servers/"
}

func HubAddServer(ctx Context) {
	fmt.Println("Add server")
}

func HubListServers(ctx Context) {
	fmt.Println("List servers")
	resp, err := http.Get(GetBasePath() + "list")

	handleErr(err)
	printRespBody(resp)
}

func HubConnectToServer(ctx Context) {
	fmt.Printf("Connect to server [id:%v]\n", ctx.id)
	resp, err := http.Get(GetBasePath() + "connect")

	handleErr(err)
	printRespBody(resp)
}

func HubInvalidOption(ctx Context) {
	fmt.Println("Invalid option")
}

func promptUser() (HubFunc, Context) {
	var input string

	fmt.Println("What would you like to do? (l)ist, (a)dd, (c)onnect [id]")
	emptyContext := Context{}

	fmt.Scanf("%s\n", &input)
	switch input {
	case "l":
		return HubListServers, emptyContext
	case "a":
		return HubAddServer, emptyContext
	case "c":
		fmt.Println("Connect selected, enter server id:")
		var id uint32
		// TODO(antonskwr): handle negative values and bigger than max uint32
		// in golang will be 0 if overflood
		fmt.Scanf("%d\n", &id)
		connectCtx := Context {id}
		return HubConnectToServer, connectCtx
	default:
		return HubInvalidOption, emptyContext
	}
}

func printRespBody(resp *http.Response) {
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	handleErr(err)
	fmt.Println(string(body))
}

func handleErr(err error, message ...string) {
	if err != nil {
		if len(message) > 0 {
			err = fmt.Errorf("[%s] -- %w --", message[0], err)
		}
		log.Fatal(err)
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
