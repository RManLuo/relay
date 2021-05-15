package relay

import (
	"errors"
	"net"
	"time"
)

type UDPDistribute struct {
	Connected bool
	Conn      *(net.UDPConn)
	Cache     chan []byte
	RAddr     net.Addr
	LAddr     net.Addr
}

func NewUDPDistribute(conn *(net.UDPConn), addr net.Addr) *UDPDistribute {
	return &UDPDistribute{
		Connected: true,
		Conn:      conn,
		Cache:     make(chan []byte, 16),
		RAddr:     addr,
		LAddr:     conn.LocalAddr(),
	}
}

func (this *UDPDistribute) Close() error {
	this.Connected = false
	return this.Conn.Close()
	// return nil
}

func (this *UDPDistribute) Read(b []byte) (n int, err error) {
	if !this.Connected {
		return 0, errors.New("udp conn has been closed")
	}
	select {
	case <-time.After(16 * time.Second):
		return 0, errors.New("i/o read timeout")
	case data := <-this.Cache:
		n := len(data)
		copy(b, data)
		return n, nil
	}
}

func (this *UDPDistribute) Write(b []byte) (int, error) {
	if !this.Connected {
		return 0, errors.New("udp conn has been closed")
	}
	return this.Conn.WriteTo(b, this.RAddr)
}

func (this *UDPDistribute) RemoteAddr() net.Addr {
	return this.RAddr
}
func (this *UDPDistribute) LocalAddr() net.Addr {
	return this.LAddr
}
func (this *UDPDistribute) SetDeadline(t time.Time) error {
	return this.Conn.SetDeadline(t)
}
func (this *UDPDistribute) SetReadDeadline(t time.Time) error {
	return this.Conn.SetReadDeadline(t)
}
func (this *UDPDistribute) SetWriteDeadline(t time.Time) error {
	return this.Conn.SetWriteDeadline(t)
}
