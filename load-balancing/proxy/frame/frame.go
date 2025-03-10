package frame

import (
	"encoding/binary"
	"fmt"
	"io"
)

func ReadUint32(rd io.Reader) (uint32, error) {
	var i uint32
	if err := binary.Read(rd, binary.LittleEndian, &i); err != nil {
		return 0, fmt.Errorf("Error read uint32: %v", err)
	}
	return i, nil
}

func WriteUint32(wr io.Writer, i uint32) error {
	if err := binary.Write(wr, binary.LittleEndian, i); err != nil {
		return fmt.Errorf("Err write uint32: %v", err)
	}
	return nil
}

// Read a frame into an existing buffer
func ReadFrameInto(rd io.Reader, b []byte) (int, error) {
	var nbyte uint32
	nbyte, err := ReadUint32(rd)
	if err != nil {
		return 0, fmt.Errorf("Error read nbyte: %v", err)
	}
	if int(nbyte) > len(b) {
		return 0, fmt.Errorf("Output buf too short: %v < %v", len(b), nbyte)
	}
	// Only read the first nbyte bytes
	b = b[:nbyte]
	n, err := io.ReadFull(rd, b)
	if n != int(nbyte) {
		return 0, fmt.Errorf("ReadFull short: %v < %v", n, nbyte)
	}
	if err != nil {
		return 0, fmt.Errorf("ReadFull err: %v", err)
	}
	return int(nbyte), nil
}

func WriteFrame(wr io.Writer, b []byte) error {
	nbyte := uint32(len(b))
	if err := WriteUint32(wr, nbyte); err != nil {
		return fmt.Errorf("Err write nbyte: %v", err)
	}
	if n, err := wr.Write(b); err != nil {
		return fmt.Errorf("Err write buf: %v", err)
	} else if n < len(b) {
		return fmt.Errorf("Write short: %v < %v", nbyte, len(b))
	}
	return nil
}
