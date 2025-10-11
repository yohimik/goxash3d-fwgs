package goxash3d_fwgs

import (
	"fmt"
	"strconv"
	"strings"
)

type BaseNetOptions struct {
	HostName string
	HostID   int
}

type NetSocket struct {
	id     int
	domain int
	typ    int
	proto  int
	addr   *Addr
}

type BaseNet struct {
	lastSocketID int
	sockets      map[int]*NetSocket
	packets      *PacketQueue
	Options      BaseNetOptions
}

func NewBaseNet(opts BaseNetOptions) *BaseNet {
	return &BaseNet{
		sockets: make(map[int]*NetSocket),
		packets: NewPacketQueue(128),
		Options: opts,
	}
}

func (n *BaseNet) Socket(domain, typ, proto int) int {
	n.lastSocketID += 1
	socket := &NetSocket{
		id:     n.lastSocketID,
		domain: domain,
		typ:    typ,
		proto:  proto,
	}
	n.sockets[socket.id] = socket
	return socket.id
}

func (n *BaseNet) CloseSocket(fd int) int {
	_, ok := n.sockets[fd]
	if !ok {
		return -1
	}
	delete(n.sockets, fd)
	return 0
}

func (n *BaseNet) PushPacket(packet Packet) {
	n.packets.Enqueue(packet)
}

func (n *BaseNet) RecvFrom() *Packet {
	p, ok := n.packets.TryDequeue()
	if !ok {
		return nil
	}
	return &p
}

func (n *BaseNet) Bind(fd int, addr Addr) int {
	s, ok := n.sockets[fd]
	if !ok {
		return -1
	}
	s.addr = &addr
	return 0
}

func (n *BaseNet) GetSockName(fd int) *Addr {
	s, ok := n.sockets[fd]
	if !ok {
		return nil
	}
	return s.addr
}

func (n *BaseNet) GetHostByName(host string) int {
	return 0
}

func (n *BaseNet) GetHostName() string {
	return fmt.Sprintf("%s.%d", n.Options.HostName, n.Options.HostID)
}

func (n *BaseNet) GetAddrInfo(host string) uint8 {
	items := strings.Split(host, ".")
	v, _ := strconv.Atoi(items[1])
	return uint8(v)
}
