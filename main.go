package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
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

type PayloadMessage struct {
	Message       string `json:"message"`
	Authorization string `json:"authorization"`
}

type BroadcastMessage struct {
	Email   string `json:"email"`
	Message string `json:"message"`
}

type ChatMessage struct {
	RoomID    string    `json:"room_id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	Message   string    `json:"message"`
}

// Env variable

var rooms = make(map[string]*Room) // Map of roomId to Room

func publishChatMessageToRabbitMQ(chatMessage ChatMessage) error {
	var rabbitHost = os.Getenv("RABBIT_HOST")
	logrus.Info("RabbitMQ Host:", rabbitHost)
	conn, err := amqp.Dial(rabbitHost)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %w", err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		"chat_histories", // name
		"direct",         // type
		true,             // durable
		false,            // auto-deleted
		false,            // internal
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare an exchange: %w", err)
	}

	body, err := json.Marshal(chatMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal chat message: %w", err)
	}

	err = ch.Publish(
		"chat_histories", // exchange
		"message",        // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		return fmt.Errorf("failed to publish a message: %w", err)
	}

	return nil
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get Header Bearer Token
	token := r.URL.Query().Get("authorization")
	if token == "" {
		logrus.Error("Authorization is required")
		http.Error(w, "Authorization is required", http.StatusBadRequest)
		return
	}

	//Get User Info
	response, err := GetUserInfoByToken(token)
	logrus.Info("User Info:", response)
	if err != nil {
		logrus.Error("Error getting user info:", err)
		http.Error(w, "Error getting user info", http.StatusInternalServerError)
		return
	}
	if response.Data.ID == 0 {
		logrus.Error("Authorization not valid")
		http.Error(w, "User info is empty", http.StatusInternalServerError)
		return
	}
	logrus.Info("User Info:", response)
	logrus.Info("User ID:", response.Data.ID)

	// Parse roomId from query parameters
	roomID := r.URL.Query().Get("roomId")
	if roomID == "" {
		logrus.Error("roomId is required")
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
		fmt.Println("Received message from roomid:", roomID)
		fmt.Printf("Received message from client: %s\n", message)
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

func broadcastToRoom(roomID string, payloadMessage string) {
	//decode payload message
	var payload PayloadMessage
	err := json.Unmarshal([]byte(payloadMessage), &payload)
	if err != nil {
		fmt.Println("Error decoding payload message:", err)
		return
	}

	fmt.Println("Broadcasting message to room:", roomID)
	fmt.Println("Message:", payload.Message)
	fmt.Println("Authorization:", payload.Authorization)

	//Get User Info
	response, err := GetUserInfoByToken(payload.Authorization)
	logrus.Info("User Info:", response)
	if err != nil {
		logrus.Error("Error getting user info:", err)
		return
	}

	var broadcastMessage BroadcastMessage
	broadcastMessage.Email = response.Data.Email
	broadcastMessage.Message = payload.Message

	//encode broadcast message
	broadcastMessageBytes, _ := json.Marshal(broadcastMessage)

	//store to rabbitmq using goroutine
	chatMessage := ChatMessage{
		RoomID:    roomID,
		Email:     response.Data.Email,
		CreatedAt: time.Now(),
		Message:   payload.Message,
	}
	err = publishChatMessageToRabbitMQ(chatMessage)
	if err != nil {
		fmt.Println("Error publishing chat message to RabbitMQ:", err)
		return
	}

	if room, ok := rooms[roomID]; ok {
		room.Mutex.Lock()
		defer room.Mutex.Unlock()

		for client := range room.Clients {
			// err := client.WriteMessage(websocket.TextMessage, []byte(payload.Message))
			err := client.WriteMessage(websocket.TextMessage, broadcastMessageBytes)
			if err != nil {
				fmt.Println("Error broadcasting message:", err)
				client.Close()
				delete(room.Clients, client)
			}
		}
	}
}

func main() {
	//godotenv initialisation
	err := godotenv.Load()
	if err != nil {
		logrus.Fatalf("Error loading .env file")
		return
	}

	http.HandleFunc("/ws", handleWebSocket)

	port := "80"
	fmt.Printf("WebSocket server is listening on ws://localhost:%s/ws\n", port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
