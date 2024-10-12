package protocol

/*
SolarmanV5 JE Protocol

Just enough (JE) version needed for packet routing.
*/
import (
	"encoding/binary"
	"errors"
)

const (
	V5Start     byte = 0xa5
	V5End       byte = 0x15
	minFrameLen int  = 13
)

type V5Frame struct {
	head      byte
	length    [2]byte
	frameType [2]byte
	seqNo     [2]byte
	serial    [4]byte
	checksum  byte
	tail      byte
	packet    []byte
}

func NewV5Frame(data []byte) (*V5Frame, error) {
	if len(data) < minFrameLen || data[0] != V5Start || data[len(data)-1] != V5End {
		return nil, errors.New("invalid frame")
	}
	frame := &V5Frame{
		head:      data[0],
		length:    [2]byte{data[1], data[2]},
		frameType: [2]byte{data[3], data[4]},
		seqNo:     [2]byte{data[5], data[6]},
		serial:    [4]byte{data[7], data[8], data[9], data[10]},
		checksum:  data[len(data)-2],
		tail:      data[len(data)-1],
		packet:    data,
	}

	return frame, nil
}

// ChecksumOK V5 packet checksum verification
func (f *V5Frame) ChecksumOK() bool {
	var checksum byte
	for i := 1; i < len(f.packet)-2; i++ {
		checksum += f.packet[i] & 0xff
	}
	return checksum == f.checksum
}

func (f *V5Frame) LoggerSN() uint32 {
	return binary.LittleEndian.Uint32(f.serial[:])
}

func (f *V5Frame) PayloadLen() uint16 {
	return binary.LittleEndian.Uint16(f.length[:])
}

func (f *V5Frame) Length() int {
	return len(f.packet)
}
