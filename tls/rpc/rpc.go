package rpc

import (
	"fmt"
	"net"

	"proxy/frame"
)

// Tell the server the number of RPCs to expect
func InformNRPC(conn net.Conn, nrpc int) error {
	return frame.WriteUint32(conn, uint32(nrpc))
}

// Get the number of RPCs to expect from the client
func GetNRPC(conn net.Conn) (int, error) {
	nrpc, err := frame.ReadUint32(conn)
	return int(nrpc), err
}

func ClientRPC(addr []byte, conn net.Conn, b []byte) error {
	if err := frame.WriteFrame(conn, addr); err != nil {
		return err
	}
	if err := frame.WriteFrame(conn, b); err != nil {
		return err
	}
	nbyte, err := frame.ReadFrameInto(conn, b)
	if err != nil {
		return err
	}
	if nbyte != len(b) {
		return fmt.Errorf("Err recvd wrong num bytes: %v != %v", nbyte, len(b))
	}
	return nil
}

func ProxyRPC(addr []byte, cliConn net.Conn, srvConn net.Conn, b []byte) error {
	// Read the destination address from the client
	addrNByte, err := frame.ReadFrameInto(cliConn, addr)
	if err != nil {
		return err
	}
	// Read an RPC from the client
	nbyte, err := frame.ReadFrameInto(cliConn, b)
	if err != nil {
		return err
	}
	// Forward the RPC to the server
	b2 := b[:nbyte]
	if err := ClientRPC(addr[:addrNByte], srvConn, b2); err != nil {
		return err
	}
	// Respond to the client
	err = frame.WriteFrame(cliConn, b2)
	if err != nil {
		return err
	}
	return nil
}

func ServeRPC(addr []byte, conn net.Conn, b []byte) (int, int, error) {
	addrNByte, err := frame.ReadFrameInto(conn, addr)
	if err != nil {
		return 0, 0, err
	}
	nbyte, err := frame.ReadFrameInto(conn, b)
	if err != nil {
		return 0, 0, err
	}
	b2 := b[:nbyte]
	if err := frame.WriteFrame(conn, b2); err != nil {
		return 0, 0, err
	}
	return addrNByte, nbyte, nil
}
