package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/bytedance/sonic"
	"nhooyr.io/websocket"
)

var (
	ctx   context.Context = context.Background()
	mutex                 = &sync.Mutex{}
	msg   input
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

	set := portgrab()

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

	if _, err := os.Stat("databases"); os.IsNotExist(err) {
		err = os.Mkdir("databases", 0755)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("Folder 'databases' created successfully.")
	}

	var set settings
	sonic.Unmarshal(getFilecontent("settings.json"), &set)
	return set
}

func setup() {
	defaultset := settings{
		Addr: "0.0.0.0",
		Port: "25565",
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
	return

}
