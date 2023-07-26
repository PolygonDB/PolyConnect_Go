package main

import (
	"fmt"
	"net/http"

	"nhooyr.io/websocket"
)

func main() {

	set := "0.0.0.0:25565"

	http.HandleFunc("/ws", datahandler)
	fmt.Print("Server started on -> "+set, "\n")

	http.ListenAndServe(set, nil)
}

func datahandler(w http.ResponseWriter, r *http.Request) {

	ws, _ := websocket.Accept(w, r, nil)
	defer ws.Close(websocket.StatusNormalClosure, "")

}
