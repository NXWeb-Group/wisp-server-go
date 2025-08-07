package utils

import (
	"encoding/binary"
	"errors"

	"github.com/NXWeb-Group/wisp-server-go/types"
)

func SerializeFrame(frame types.WispFrame) []byte {
	data := make([]byte, 0, 5+len(frame.Payload))
	data = append(data, byte(frame.Type))

	// Convert uint32 to 4 bytes using little endian encoding
	streamIDBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(streamIDBytes, frame.StreamID)
	data = append(data, streamIDBytes...)

	data = append(data, frame.Payload...)
	return data
}

func DeserializeFrame(data []byte) (types.WispFrame, error) {
	if len(data) < 5 { // Now requires at least 5 bytes (1 + 4 + 0)
		return types.WispFrame{}, errors.New("invalid frame")
	}

	frame := types.WispFrame{
		Type:     types.PACKET_TYPE(data[0]),            // First byte is the type (uint8)
		StreamID: binary.LittleEndian.Uint32(data[1:5]), // Next 4 bytes are stream ID (little endian)
		Payload:  data[5:],                              // Remaining bytes are payload
	}

	return frame, nil
}

func ParseConnect(data []byte) (types.Connect, error) {
	if len(data) < 3 {
		return types.Connect{}, errors.New("invalid connect frame")
	}

	connect := types.Connect{
		Type: types.STREAM_TYPE(data[0]),
		Port: binary.LittleEndian.Uint16(data[1:3]),
		Host: data[3:],
	}

	return connect, nil
}
