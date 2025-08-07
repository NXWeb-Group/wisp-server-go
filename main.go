package main

import (
	"encoding/binary"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/NXWeb-Group/wisp-server-go/types"
	"github.com/NXWeb-Group/wisp-server-go/utils"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin for development
		return true
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	wsmutex := sync.Mutex{}

	log.Println("Client connected")

	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload, 127)
	wsmutex.Lock()
	conn.WriteMessage(websocket.BinaryMessage, utils.SerializeFrame(types.WispFrame{
		Type:     types.CONTINUE,
		StreamID: 0,
		Payload:  payload,
	}))
	wsmutex.Unlock()

	streams := types.Streams{
		Sockets: make(map[uint32]types.Socket),
		Mutex:   sync.RWMutex{},
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		frame, err := utils.DeserializeFrame(message)
		if err != nil {
			log.Printf("Error deserializing frame: %v", err)
			break
		}

		switch frame.Type {
		case types.CONNECT:
			connect, err := utils.ParseConnect(frame.Payload)
			if err != nil {
				log.Printf("Error parsing CONNECT frame: %v", err)
				break
			}

			switch connect.Type {
			case types.TCP:
				utils.TCPSocket(conn, connect, frame, &streams, &wsmutex)
			case types.UDP:
				log.Printf("UDP not supported yet")
			default:
				log.Printf("Unknown stream type: %v", connect.Type)
			}

		case types.DATA:
			streams.Mutex.RLock()
			socket, exists := streams.Sockets[frame.StreamID]
			streams.Mutex.RUnlock()
			if !exists {
				log.Printf("No socket found for StreamID: %d", frame.StreamID)
				break
			}

			if socket.TCP != nil {
				_, err := socket.TCP.Write(frame.Payload)
				if err != nil {
					log.Printf("Error writing to TCP socket: %v", err)
				}
				socket.Buffer -= 1
				streams.Mutex.Lock()
				if socket.Buffer == 0 {
					socket.Buffer = 127
					go func() {
						wsmutex.Lock()
						conn.WriteMessage(websocket.BinaryMessage, utils.SerializeFrame(types.WispFrame{
							Type:     types.CONTINUE,
							StreamID: frame.StreamID,
							Payload:  payload,
						}))
						wsmutex.Unlock()
					}()
				}
				streams.Sockets[frame.StreamID] = socket
				streams.Mutex.Unlock()

			} else if socket.UDP != nil {
				_, err := socket.UDP.Write(frame.Payload)
				if err != nil {
					log.Printf("Error writing to UDP socket: %v", err)
				}
			} else {
				log.Printf("No active socket for StreamID: %d", frame.StreamID)
			}
		case types.CLOSE:
			log.Printf("Received CLOSE frame with StreamID: %d", frame.StreamID)
			streams.Mutex.Lock()
			if socket, exists := streams.Sockets[frame.StreamID]; exists {
				if socket.TCP != nil {
					socket.TCP.Close()
				}
				if socket.UDP != nil {
					socket.UDP.Close()
				}
				delete(streams.Sockets, frame.StreamID)
			}
			streams.Mutex.Unlock()
		default:
			log.Printf("Received unknown frame type: %d", frame.Type)
		}
	}

	streams.Mutex.Lock()
	for _, socket := range streams.Sockets {
		if socket.TCP != nil {
			socket.TCP.Close()
		}
		if socket.UDP != nil {
			socket.UDP.Close()
		}
	}
	streams.Mutex.Unlock()

	log.Println("Client disconnected")
}

func main() {
	http.HandleFunc("/wisp/", handleWebSocket)

	port := ":8080"
	if len(os.Args) > 1 {
		port = ":" + os.Args[1]
	}

	log.Printf("Starting WebSocket server on %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
