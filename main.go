package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/bytedance/sonic"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

var (
	ctx   context.Context = context.Background()
	mutex                 = &sync.Mutex{}
	msg   input
	set   settings
)

// Settings.json parsing
type settings struct {
	Addr string `json:"addr"`
	Port string `json:"port"`
	Name string `json:"name-of-exe"`
}

type input struct {
	Dbname string      `json:"dbname"`
	Loc    string      `json:"location"`
	Act    string      `json:"action"`
	Val    interface{} `json:"value"`
}

func main() {

	set = portgrab()

	http.HandleFunc("/ws", datahandler)
	fmt.Print("Server started on -> "+set.Addr+":"+set.Port, "\n")

	http.ListenAndServe(set.Addr+":"+set.Port, nil)
}

func datahandler(w http.ResponseWriter, r *http.Request) {

	ws, _ := websocket.Accept(w, r, nil)
	defer ws.Close(websocket.StatusNormalClosure, "")

}

func portgrab() settings {
	if _, err := os.Stat("settings.json"); os.IsNotExist(err) {
		setup()
	}

	var set settings
	sonic.Unmarshal(getFilecontent("settings.json"), &set)
	return set
}

// Makes PolygonDB settings.json
func setup() {
	defaultset := settings{
		Addr: "0.0.0.0",
		Port: "25565",
		Name: "PolygonDB.exe",
	}
	data, _ := sonic.ConfigFastest.MarshalIndent(&defaultset, "", "    ")
	os.WriteFile("settings.json", data, 0644)
	fmt.Print("Settings.json has been setup. \n")
}

func getFilecontent(filename string) []byte {
	file, _ := os.ReadFile("settings.json")
	return file
}

func takein(ws *websocket.Conn, r *http.Request) bool {

	//Reads input
	_, reader, err := ws.Reader(ctx)
	if err != nil {
		return false
	}

	message, _ := io.ReadAll(reader)

	mutex.Lock()
	if err = sonic.Unmarshal(message, &msg); err != nil {
		return false
	}

	//add message to the queue
	process(&msg, ws)
	mutex.Unlock()

	return true
}

func process(msg *input, ws *websocket.Conn) {
	fmt.Print(sonic.MarshalString(msg))
	cmd := exec.Command(set.Name)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("Error creating stdin pipe:", err)
		return
	}
	defer stdin.Close()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error creating stdout pipe:", err)
		return
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting the executable:", err)
		return
	}

	response := make([]byte, 4096)

	message := msg
	_, err = fmt.Fprintln(stdin, message)
	if err != nil {
		fmt.Println("Error sending message to the executable:", err)
		return
	}

	// Read the response from the executable
	n, err := io.ReadFull(stdout, response)
	if err != nil {
		fmt.Println("Error reading from the executable:", err)
		return
	}

	// Print the response
	wsjson.Write(ctx, ws, string(response[:n]))

	return
}
