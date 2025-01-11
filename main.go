package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Room struct {
	Clients map[*websocket.Conn]bool
	Mutex   sync.Mutex
}

var rooms = make(map[string]*Room) // Map of roomId to Room

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Parse roomId from query parameters
	roomID := r.URL.Query().Get("roomId")
	if roomID == "" {
		http.Error(w, "roomId is required", http.StatusBadRequest)
		return
	}

	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading connection:", err)
		return
	}
	defer func() {
		leaveRoom(roomID, conn)
		conn.Close()
	}()

	// Join the room
	joinRoom(roomID, conn)

	for {
		// Read message from client
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message, closing connection:", err)
			break
		}

		// Broadcast the message to the room
		broadcastToRoom(roomID, string(message))
	}
}

func joinRoom(roomID string, conn *websocket.Conn) {
	// Ensure the room exists
	if _, ok := rooms[roomID]; !ok {
		rooms[roomID] = &Room{Clients: make(map[*websocket.Conn]bool)}
	}

	room := rooms[roomID]
	room.Mutex.Lock()
	room.Clients[conn] = true
	room.Mutex.Unlock()

	fmt.Printf("Client joined room: %s\n", roomID)
}

func leaveRoom(roomID string, conn *websocket.Conn) {
	if room, ok := rooms[roomID]; ok {
		room.Mutex.Lock()
		delete(room.Clients, conn)
		room.Mutex.Unlock()
		fmt.Printf("Client left room: %s\n", roomID)

		// Clean up the room if empty
		if len(room.Clients) == 0 {
			delete(rooms, roomID)
		}
	}
}

func broadcastToRoom(roomID string, message string) {
	if room, ok := rooms[roomID]; ok {
		room.Mutex.Lock()
		defer room.Mutex.Unlock()

		for client := range room.Clients {
			err := client.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				fmt.Println("Error broadcasting message:", err)
				client.Close()
				delete(room.Clients, client)
			}
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)

	port := "8080"
	fmt.Printf("WebSocket server is listening on ws://localhost:%s/ws\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
