package goxash3d_fwgs

import (
	"fmt"
	"github.com/yohimik/goxash3d-fwgs/pkg/platform"
	"strconv"
	"strings"
)

// BaseNetOptions holds configuration for the BaseNet instance.
type BaseNetOptions struct {
	HostName string
	HostID   int
}

// NetSocket represents a simplified network socket.
type NetSocket struct {
	id     int
	domain int
	typ    int
	proto  int
	addr   *Addr
}

// BaseNet provides basic socket management and packet queuing.
// It acts as an abstraction over simplified network operations.
type BaseNet struct {
	lastSocketID int
	sockets      map[int]*NetSocket
	packets      *PacketQueue
	Options      BaseNetOptions
}

// NewBaseNet creates and initializes a new BaseNet instance
// with the given configuration options.
func NewBaseNet(opts BaseNetOptions) *BaseNet {
	return &BaseNet{
		sockets: make(map[int]*NetSocket),
		packets: NewPacketQueue(128),
		Options: opts,
	}
}

// Socket creates a new socket with specified parameters and returns its ID.
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

// CloseSocket closes the socket with the given file descriptor (ID).
// Returns 0 on success or -1 if the socket does not exist.
func (n *BaseNet) CloseSocket(fd int) int {
	_, ok := n.sockets[fd]
	if !ok {
		return -1
	}
	delete(n.sockets, fd)
	return 0
}

// PushPacket adds a packet to the internal packet queue.
func (n *BaseNet) PushPacket(packet Packet) {
	n.packets.Enqueue(packet)
}

// RecvFrom attempts to retrieve a packet from the queue.
//
// It introduces a small delay to emulate timing behavior
// (e.g., simulating blocking behavior similar to recvfrom).
// Returns nil if no packet is available.
func (n *BaseNet) RecvFrom() *Packet {
	platform.Delay()
	p, ok := n.packets.TryDequeue()
	if !ok {
		return nil
	}
	return &p
}

// Bind associates a socket with a given address.
// Returns 0 on success or -1 if the socket doesn't exist.
func (n *BaseNet) Bind(fd int, addr Addr) int {
	s, ok := n.sockets[fd]
	if !ok {
		return -1
	}
	s.addr = &addr
	return 0
}

// GetSockName returns the bound address of the specified socket.
// Returns nil if the socket doesn't exist or isn't bound.
func (n *BaseNet) GetSockName(fd int) *Addr {
	s, ok := n.sockets[fd]
	if !ok {
		return nil
	}
	return s.addr
}

// GetHostByName resolves a host name to an internal ID.
func (n *BaseNet) GetHostByName(host string) int {
	return 0
}

// GetHostName returns the full host identifier as a string,
func (n *BaseNet) GetHostName() string {
	return fmt.Sprintf("%s.%d", n.Options.HostName, n.Options.HostID)
}

// GetAddrInfo extracts an identifier from a host string.
func (n *BaseNet) GetAddrInfo(host string) uint8 {
	items := strings.Split(host, ".")
	v, _ := strconv.Atoi(items[1])
	return uint8(v)
}
