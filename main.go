package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

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

func main2() {

	set = portgrab()

	http.HandleFunc("/ws", datahandler)
	fmt.Print("Server started on -> "+set.Addr+":"+set.Port, "\n")

	http.ListenAndServe(set.Addr+":"+set.Port, nil)
}

func datahandler(w http.ResponseWriter, r *http.Request) {

	ws, _ := websocket.Accept(w, r, nil)
	defer ws.Close(websocket.StatusNormalClosure, "")

	for {
		if !takein(ws, r) {
			ws.Close(websocket.StatusInternalError, "")
			break
		}
	}
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
		wsjson.Write(ctx, ws, err.Error())
		return false
	}

	message, _ := io.ReadAll(reader)

	mutex.Lock()
	if err = sonic.Unmarshal(message, &msg); err != nil {
		wsjson.Write(ctx, ws, err.Error())
		return false
	}

	//add message to the queue
	process(&msg, ws)
	mutex.Unlock()

	return true
}

func process(msg *input, ws *websocket.Conn) {
	fmt.Print(sonic.MarshalString(msg))
	currentDir, _ := os.Getwd()
	cmd := exec.Command(filepath.Join(currentDir, set.Name))

	stdin, err := cmd.StdinPipe()
	if err != nil {
		wsjson.Write(ctx, ws, err.Error())
		return
	}
	defer stdin.Close()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		wsjson.Write(ctx, ws, err.Error())
		return
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		wsjson.Write(ctx, ws, err.Error())
		return
	}

	message, _ := sonic.ConfigFastest.MarshalToString(msg)
	_, err = io.WriteString(stdin, message)
	if err != nil {
		fmt.Println("Error sending message to the executable:", err)
		return
	}

	// Read the response from the executable
	response, err := io.ReadAll(stdout)
	if err != nil {
		fmt.Println("Error reading from the executable:", err)
		return
	}

	// Print the response

	wsjson.Write(ctx, ws, string(response))
}

func main() {

	os.Getwd()
	cmd := exec.Command("./PolygonDB.exe")
	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting the executable:", err)
		return
	} else {
		fmt.Print("works")
		time.Sleep(200)
	}

	cmd.Process.Kill()

	fmt.Print(cmd.Output())
}
