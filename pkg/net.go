package goxash3d_fwgs

/*
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <netdb.h>
#include <unistd.h>
#include <errno.h>
*/
import "C"

import (
	"unsafe"
)

// Packet is the concrete payload.
type Packet struct {
	Data []byte
	Addr
}

type Addr struct {
	IP   [4]byte
	Port uint16
}

type Xash3DNetwork interface {
	Socket(domain, typ, proto int) int
	CloseSocket(fd int) int
	// SendTo should send provided packet (fd, flags, and sockaddr bytes)
	// Return number of bytes sent (>=0) or negative on error.
	SendTo(fd int, pkt Packet, flags int) int
	// Send multiple packets to the same destination. Return total bytes sent.
	SendToBatch(fd int, packets []Packet, flags int) int
	// RecvFrom should receive and return a Packet (or nil on error).
	// It may allocate a Packet whose Data points to a Go buffer.
	RecvFrom() *Packet
	// Bind receives fd and raw sockaddr bytes (as passed from C).
	Bind(fd int, add Addr) int
	// GetSockName returns ok, domain, type, proto, and raw sockaddr bytes to be copied to C.
	GetSockName(fd int) *Addr
	// GetHostByName returns an int (legacy); you can return 0 on success or errno-like code.
	GetHostByName(host string) int
	// GetHostName writes into provided buffer and returns number of bytes written or negative on error.
	GetHostName() string
	// GetAddrInfo should allocate/write addrinfo structures into C memory (hintsPtr & resultPtr are C pointers).
	// Return 0 on success (like getaddrinfo), otherwise non-zero.
	GetAddrInfo(host string) uint8
}

// parseIPv4Sockaddr attempts to parse an IPv4 sockaddr (sa pointer with given socklen).
// Returns slice of bytes representing the sockaddr (length = socklen) and also fills ip and port if IPv4.
// This is a minimal, fast parser for common AF_INET usage.
func parseIPv4Sockaddr(sockaddr unsafe.Pointer, socklen C.socklen_t) (addr Addr) {
	// cast to sockaddr_in
	sin := (*C.struct_sockaddr_in)(sockaddr)
	// extract IPv4 and port (network byte order)
	a := uint32(sin.sin_addr.s_addr)
	addr.IP[0] = byte(a & 0xff)
	addr.IP[1] = byte((a >> 8) & 0xff)
	addr.IP[2] = byte((a >> 16) & 0xff)
	addr.IP[3] = byte((a >> 24) & 0xff)
	addr.Port = uint16(C.ntohs(sin.sin_port))
	return
}

// writeIPv4Sockaddr writes an IPv4 sockaddr_in into provided sockaddr pointer and returns written length.
func writeIPv4Sockaddr(sockaddr unsafe.Pointer, socklen *C.socklen_t, addr Addr) C.socklen_t {
	var sin C.struct_sockaddr_in
	sin.sin_family = C.AF_INET
	sin.sin_port = C.htons(C.ushort(addr.Port))
	// compose s_addr as network-order uint32
	s_addr := uint32(addr.IP[0]) | uint32(addr.IP[1])<<8 | uint32(addr.IP[2])<<16 | uint32(addr.IP[3])<<24
	sin.sin_addr.s_addr = C.uint32_t(s_addr)
	wlen := C.socklen_t(unsafe.Sizeof(sin))
	if *socklen < wlen {
		// user provided smaller buffer: copy up to *socklen
		C.memcpy(sockaddr, unsafe.Pointer(&sin), C.size_t(*socklen))
		return *socklen
	}
	C.memcpy(sockaddr, unsafe.Pointer(&sin), C.size_t(wlen))
	*socklen = wlen
	return wlen
}

// ---- exported cgo functions ----

//export go_net_socket
func go_net_socket(domain, typ, proto C.int) C.int {
	if DefaultXash3D.Net == nil {
		return C.socket(domain, typ, proto)
	}
	return C.int(DefaultXash3D.Net.Socket(int(domain), int(typ), int(proto)))
}

//export go_net_closesocket
func go_net_closesocket(fd C.int) C.int {
	if DefaultXash3D.Net == nil {
		return C.closesocket(fd)
	}
	return C.int(DefaultXash3D.Net.CloseSocket(int(fd)))
}

//export go_net_sendto
func go_net_sendto(fd C.int, buf unsafe.Pointer, length C.size_t, flags C.int, sockaddr unsafe.Pointer, socklen C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.sendto(fd, buf, length, flags, sockaddr, socklen)
	}

	// Build Packet that points into C buffer (zero-copy)
	data := unsafe.Slice((*byte)(buf), int(length))
	addr := parseIPv4Sockaddr(sockaddr, socklen)

	// Pass raw sockaddr bytes to Net as well (no copy) — safe because slice header references C memory
	// but Go must not retain beyond this call.
	n := DefaultXash3D.Net.SendTo(int(fd), Packet{
		Data: data,
		Addr: addr,
	}, int(flags))
	return C.int(n)
}

//export go_net_sendto_batch
func go_net_sendto_batch(
	fd C.int,
	packets **C.char,
	sizes *C.int,
	count C.int,
	flags C.int,
	to *C.struct_sockaddr_storage,
	tolen C.int,
) C.int {
	if DefaultXash3D.Net == nil {
		// assume C has sendto_batch symbol
		return C.sendto_batch(fd, packets, sizes, count, flags, to, tolen)
	}
	countInt := int(count)
	addr := parseIPv4Sockaddr(*to, *sizes)

	// Build slices of packet pointers and sizes without allocations.
	pktPtrs := unsafe.Slice(packets, countInt)
	sz := unsafe.Slice(sizes, countInt)

	// Prepare Go []*Packet and sizes slice that reference C buffers.
	pktList := make([]Packet, 0, countInt)
	for i := 0; i < countInt; i += 1 {
		p := pktPtrs[i]
		if p == nil {
			continue
		}
		bytes := unsafe.Slice((*byte)(unsafe.Pointer(p)), int(sz[i]))
		pktList[i] = Packet{
			Data: bytes,
			Addr: addr,
		}
	}

	n := DefaultXash3D.Net.SendToBatch(int(fd), pktList, int(flags))
	return C.int(n)
}

//export go_net_recvfrom
func go_net_recvfrom(fd C.int, buf unsafe.Pointer, length C.size_t, flags C.int, sockaddr unsafe.Pointer, socklen *C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.recvfrom(fd, buf, length, flags, sockaddr, socklen)
	}
	goBuf := unsafe.Slice((*byte)(buf), int(length))

	pkt := DefaultXash3D.Net.RecvFrom()
	if pkt == nil {
		// set errno to EAGAIN (or other; adjust in your Net impl)
		C.errno = C.int(C.EAGAIN)
		return C.int(-1)
	}
	// copy packet bytes into provided buffer (if pkt.Data points to Go memory we copy)
	copyLen := len(pkt.Data)
	if copyLen > int(length) {
		copyLen = int(length)
	}
	if copyLen > 0 {
		// pkt.Data may already reference goBuf (zero-copy); still copy to be safe
		copy(goBuf[:copyLen], pkt.Data[:copyLen])
	}

	// If sockaddr pointer provided, write IPv4 sockaddr (or raw bytes if pkt has them in Data)
	if sockaddr != nil && socklen != nil {
		// write IPv4 sockaddr into provided buffer (updates *socklen)
		writeIPv4Sockaddr(sockaddr, socklen, pkt.Addr)
	}
	return C.int(copyLen)
}

//export go_net_bind
func go_net_bind(fd C.int, sockaddr unsafe.Pointer, socklen C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.bind(fd, sockaddr, socklen)
	}
	// Convert sockaddr pointer to Go []byte (no copy)
	addr := parseIPv4Sockaddr(sockaddr, socklen)
	return C.int(DefaultXash3D.Net.Bind(int(fd), addr))
}

//export go_net_getsockname
func go_net_getsockname(fd C.int, sockaddr unsafe.Pointer, socklen *C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.getsockname(fd, sockaddr, socklen)
	}
	addr := DefaultXash3D.Net.GetSockName(int(fd))
	if addr == nil {
		// indicate error to C
		return C.int(-1)
	}
	writeIPv4Sockaddr(sockaddr, socklen, *addr)
	return C.int(0)
}

//export go_net_gethostbyname
func go_net_gethostbyname(hostname *C.char) C.int {
	if DefaultXash3D.Net == nil {
		return C.gethostbyname(hostname)
	}
	goHost := C.GoString(hostname)
	return C.int(DefaultXash3D.Net.GetHostByName(goHost))
}

//export go_net_gethostname
func go_net_gethostname(name *C.char, namelen C.size_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.gethostname(name, namelen)
	}
	// create Go buffer and let Net fill it (zero-alloc if possible)
	hostname := DefaultXash3D.Net.GetHostName()
	hb := []byte(hostname)
	n := len(hb)
	maxLen := int(namelen) - 1 // leave space for '\0'
	if n > maxLen {
		n = maxLen
	}
	// Copy hostname bytes into C buffer
	C.memcpy(unsafe.Pointer(name), unsafe.Pointer(&hb[0]), C.size_t(n))
	// Add null terminator
	*(*byte)(unsafe.Add(unsafe.Pointer(name), uintptr(n))) = 0
	return C.int(0)
}

//export go_net_getaddrinfo
func go_net_getaddrinfo(hostname, service *C.char, hints, result unsafe.Pointer) C.int {
	if DefaultXash3D.Net == nil {
		return C.getaddrinfo(hostname, service, (*C.struct_addrinfo)(hints), (**C.struct_addrinfo)(result))
	}
	host := C.GoString(hostname)
	id := DefaultXash3D.Net.GetAddrInfo(host)

	// Allocate sockaddr (AF_INET)
	sa := C.malloc(C.size_t(unsafe.Sizeof(C.struct_sockaddr_in{})))
	if sa == nil {
		return C.EAI_MEMORY
	}

	var sin C.struct_sockaddr_in
	sin.sin_family = C.AF_INET
	sin.sin_port = 0

	// Compose IP as 101.101.(id & 0xff).((id >> 8) & 0xff)
	a := id & 0xff
	b := byte(0)
	sin.sin_addr.s_addr = C.uint32_t(a)<<24 | C.uint32_t(b)<<16 | 101<<8 | 101

	C.memcpy(sa, unsafe.Pointer(&sin), C.size_t(unsafe.Sizeof(sin)))

	// Allocate addrinfo
	ai := C.malloc(unsafe.Sizeof(C.struct_addrinfo{}))
	if ai == nil {
		C.free(sa)
		return C.EAI_MEMORY
	}

	aiStruct := (*C.struct_addrinfo)(ai)
	aiStruct.ai_flags = 0
	aiStruct.ai_family = C.AF_INET
	aiStruct.ai_socktype = C.SOCK_DGRAM
	aiStruct.ai_protocol = C.IPPROTO_UDP
	aiStruct.ai_addrlen = C.socklen_t(unsafe.Sizeof(sin))
	aiStruct.ai_addr = (*C.struct_sockaddr)(sa)
	aiStruct.ai_canonname = nil
	aiStruct.ai_next = nil

	// Write addrinfo pointer into result
	*(*unsafe.Pointer)(result) = ai

	return C.int(0)
}
