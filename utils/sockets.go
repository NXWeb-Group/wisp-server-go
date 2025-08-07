package utils

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/NXWeb-Group/wisp-server-go/types"
	"github.com/gorilla/websocket"
)

func TCPSocket(conn *websocket.Conn, connect types.Connect, frame types.WispFrame, streams *types.Streams, wsmutex *sync.Mutex) {
	socket, err := net.Dial("tcp", net.JoinHostPort(string(connect.Host), fmt.Sprintf("%d", connect.Port)))
	if err != nil {
		log.Printf("Error connecting to TCP socket: %v", err)
		wsmutex.Lock()
		conn.WriteMessage(websocket.BinaryMessage, SerializeFrame(types.WispFrame{
			Type:     types.CLOSE,
			StreamID: frame.StreamID,
			Payload:  []byte{0x01},
		}))
		wsmutex.Unlock()
		return
	}

	streams.Mutex.Lock()
	socketEntry := streams.Sockets[frame.StreamID]
	socketEntry.TCP = socket.(*net.TCPConn)
	socketEntry.Buffer = 127
	streams.Sockets[frame.StreamID] = socketEntry
	streams.Mutex.Unlock()

	go func() {
		defer func() {
			wsmutex.Lock()
			conn.WriteMessage(websocket.BinaryMessage, SerializeFrame(types.WispFrame{
				Type:     types.CLOSE,
				StreamID: frame.StreamID,
				Payload:  []byte{0x02},
			}))
			wsmutex.Unlock()
			streams.Mutex.Lock()
			delete(streams.Sockets, frame.StreamID)
			streams.Mutex.Unlock()
			socket.Close()

		}()
		buffer := make([]byte, 4096)
		for {
			n, err := socket.Read(buffer)
			if err != nil {
				// log.Printf("Error reading from TCP socket: %v", err)
				break
			}
			if n > 0 {
				// log.Printf("Received %d bytes from TCP socket", n)
				wsmutex.Lock()
				conn.WriteMessage(websocket.BinaryMessage, SerializeFrame(types.WispFrame{
					Type:     types.DATA,
					StreamID: frame.StreamID,
					Payload:  buffer[:n],
				}))
				wsmutex.Unlock()
			}
		}
	}()
}
