package goxash3d_fwgs

// Provides custom implementations of low-level network I/O functions
// by wrapping standard C socket functions `recvfrom` and `sendto`. These replacements
// integrate with a user-defined packet handling system to simulate
// network behavior for use in a controlled or virtualized environment.

/*
#include <errno.h>
#include <stdint.h>
#include <arpa/inet.h>
#include <netinet/in.h>

static void set_errno(int err) {
	errno = err;
}
*/
import "C"
import (
	"unsafe"
)

// Packet Represents a UDP network message
type Packet struct {
	IP   [4]byte
	Data []byte
}

// Xash3DNetwork Represents network interface of Xash3D-FWGS engine.
type Xash3DNetwork struct {
	Incoming chan Packet
	Outgoing chan Packet
}

const ChannelSize = 128

func NewXash3DNetwork() *Xash3DNetwork {
	return &Xash3DNetwork{
		Incoming: make(chan Packet, ChannelSize),
		Outgoing: make(chan Packet, ChannelSize),
	}
}

// Recvfrom Receives packets from a custom Go channel (`Incoming`),
// simulating non-blocking socket reads and populating sockaddr structures as needed.
// i386 requires 10ms timeout.
func (x *Xash3DNetwork) Recvfrom(
	sockfd Int,
	buf unsafe.Pointer,
	length Int,
	flags Int,
	src_addr *Sockaddr,
	addrlen *SocklenT,
) Int {
	var pkt Packet

	select {
	case pkt = <-x.Incoming:
	default:
		C.set_errno(C.EAGAIN)
		return -1
	}

	n := len(pkt.Data)
	if n > int(length) {
		n = int(length) // truncate
	}
	dst := unsafe.Slice((*byte)(buf), n)
	copy(dst, pkt.Data)

	if src_addr != nil && addrlen != nil {
		csa := (*C.struct_sockaddr_in)(unsafe.Pointer(src_addr))
		csa.sin_family = C.AF_INET
		csa.sin_port = C.htons(12345) // dummy port
		ip := uint32(pkt.IP[0])<<24 | uint32(pkt.IP[1])<<16 | uint32(pkt.IP[2])<<8 | uint32(pkt.IP[3])
		csa.sin_addr.s_addr = C.uint32_t(C.htonl(C.uint32_t(ip)))
		*addrlen = SocklenT(unsafe.Sizeof(*csa))
	}

	return Int(n)
}

// Sendto Sends packet data to a custom Go channel (`Outgoing`),
// simulating outgoing UDP traffic by extracting destination IP and payload.
func (x *Xash3DNetwork) Sendto(
	sockfd Int,
	buf unsafe.Pointer,
	length Int,
	flags Int,
	dest unsafe.Pointer,
	addrlen SocklenT,
) Int {
	if buf == nil || dest == nil || length <= 0 {
		return 0
	}
	sa := (*C.struct_sockaddr_in)(dest)
	ipBytes := *(*[4]byte)(unsafe.Pointer(&sa.sin_addr))
	byteView := unsafe.Slice((*byte)(buf), int(length))
	packetBuf := make([]byte, length)
	copy(packetBuf, byteView)

	x.Outgoing <- Packet{IP: ipBytes, Data: packetBuf}

	return Int(length)
}
