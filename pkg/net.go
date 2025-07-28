package goxash3d_fwgs

// Provides custom implementations of low-level network I/O functions
// by wrapping standard C socket functions `recvfrom` and `sendto`. These replacements
// integrate with a user-defined packet handling system to simulate
// network behavior for use in a controlled or virtualized environment.

/*
#include "net.h"
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

type RecvfromCallback func() *Packet
type SendtoCallback func(p Packet)

// Xash3DNetwork Represents network interface of Xash3D-FWGS engine.
type Xash3DNetwork struct {
	recvfrom RecvfromCallback
	sendto   SendtoCallback
}

func NewXash3DNetwork() *Xash3DNetwork {
	return &Xash3DNetwork{}
}

func (x *Xash3DNetwork) RegisterRecvfromCallback(cb RecvfromCallback) {
	x.recvfrom = cb
}

func (x *Xash3DNetwork) RegisterSendtoCallback(cb SendtoCallback) {
	x.sendto = cb
}

func (x *Xash3DNetwork) RegisterNetCallbacks() {
	C.RegisterRecvFromCallback((C.recvfrom_func_t)(C.Recvfrom))
	C.RegisterSendToCallback((C.sendto_func_t)(C.Sendto))
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
	pkt := x.recvfrom()

	if pkt == nil {
		C.set_errno(C.EAGAIN)
		return Int(-1)
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
	sock Int,
	packets **C.char,
	sizes *C.size_t,
	packet_count Int,
	seq_num Int,
	to *C.struct_sockaddr_storage,
	tolen SizeT,
) Int {
	count := int(packet_count)
	packetArray := unsafe.Pointer(packets)
	sizeArray := unsafe.Pointer(sizes)

	// --- Extract IP address ---
	ipBytes := extractIP(to)

	// --- Iterate packets ---
	for i := 0; i < count; i++ {
		packetPtr := *(**C.char)(unsafe.Pointer(uintptr(packetArray) + uintptr(i)*unsafe.Sizeof(uintptr(0))))
		packetSize := *(*C.size_t)(unsafe.Pointer(uintptr(sizeArray) + uintptr(i)*unsafe.Sizeof(C.size_t(0))))

		// Use unsafe.Slice for faster copy
		byteView := unsafe.Slice((*byte)(unsafe.Pointer(packetPtr)), int(packetSize))
		x.sendto(Packet{
			IP:   ipBytes,
			Data: byteView,
		})
	}

	return 10
}

func extractIP(to *C.struct_sockaddr_storage) [4]byte {
	family := to.ss_family
	switch family {
	case C.AF_INET:
		sa := (*C.struct_sockaddr_in)(unsafe.Pointer(to))
		ip := (*[4]byte)(unsafe.Pointer(&sa.sin_addr))
		return *ip
	default:
		return [4]byte{0, 0, 0, 0}
	}
}

//export Recvfrom
func Recvfrom(
	sockfd Int,
	buf unsafe.Pointer,
	length Int,
	flags Int,
	src_addr *Sockaddr,
	addrlen *SocklenT,
) Int {
	return DefaultXash3D.Recvfrom(sockfd, buf, length, flags, src_addr, addrlen)
}

//export Sendto
func Sendto(
	sock Int,
	packets **C.char,
	sizes *SizeT,
	packet_count Int,
	seq_num Int,
	to *C.struct_sockaddr_storage,
	tolen SizeT,
) Int {
	return DefaultXash3D.Sendto(sock, packets, sizes, packet_count, seq_num, to, tolen)
}
