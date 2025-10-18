package goxash3d_fwgs

/*
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <netdb.h>
#include <unistd.h>
#include <errno.h>

// set_errno is a helper to explicitly set errno from Go.
static void set_errno(int err) {
	errno = err;
}
*/
import "C"

import (
	"unsafe"
)

// Packet represents a simple network packet structure
// consisting of a byte payload and a source/destination address.
type Packet struct {
	Data []byte
	Addr Addr
}

// Addr represents an IPv4 address and port combination.
type Addr struct {
	IP   [4]byte
	Port uint16
}

// Xash3DNetwork defines an interface for emulating or overriding
// low-level networking functionality, used by the engine to
// optionally route through Go-based logic.
type Xash3DNetwork interface {
	Socket(domain, typ, proto int) int
	CloseSocket(fd int) int
	SendTo(fd int, pkt Packet, flags int) int
	SendToBatch(fd int, packets []Packet, flags int) int
	RecvFrom() *Packet
	Bind(fd int, addr Addr) int
	GetSockName(fd int) *Addr
	GetHostByName(host string) int
	GetHostName() string
	GetAddrInfo(host string) uint8
}

// sendBatch is a reusable buffer used for batching packet sends
// to avoid repeated allocations in performance-critical paths.
var (
	sendBatch [1024]Packet
)

// parseIPv4Sockaddr converts a C sockaddr pointer to a Go Addr.
// Only AF_INET addresses are supported; others return zero Addr.
func parseIPv4Sockaddr(sockaddr unsafe.Pointer) Addr {
	var out Addr
	if sockaddr == nil {
		return out
	}

	sa := (*C.struct_sockaddr)(sockaddr)
	if sa.sa_family != C.AF_INET {
		return out
	}

	sin := (*C.struct_sockaddr_in)(sockaddr)
	ipN := uint32(C.ntohl(sin.sin_addr.s_addr))
	out.IP = [4]byte{
		byte((ipN >> 24) & 0xFF),
		byte((ipN >> 16) & 0xFF),
		byte((ipN >> 8) & 0xFF),
		byte(ipN & 0xFF),
	}
	out.Port = uint16(C.ntohs(sin.sin_port))
	return out
}

// writeIPv4Sockaddr encodes a Go Addr into a sockaddr_in
// and writes it into a buffer for returning from C APIs.
func writeIPv4Sockaddr(sockaddr unsafe.Pointer, socklen *C.socklen_t, addr Addr) C.socklen_t {
	if sockaddr == nil || socklen == nil {
		return 0
	}

	var sin C.struct_sockaddr_in
	sin.sin_family = C.AF_INET
	sin.sin_port = C.htons(C.ushort(addr.Port))
	ip := (uint32(addr.IP[0]) << 24) | (uint32(addr.IP[1]) << 16) |
		(uint32(addr.IP[2]) << 8) | uint32(addr.IP[3])
	sin.sin_addr.s_addr = C.htonl(C.uint32_t(ip))

	required := C.socklen_t(unsafe.Sizeof(sin))

	if *socklen < required {
		C.memcpy(sockaddr, unsafe.Pointer(&sin), C.size_t(*socklen))
		written := *socklen
		*socklen = written
		return written
	}

	C.memcpy(sockaddr, unsafe.Pointer(&sin), C.size_t(required))
	*socklen = required
	return required
}

// lib_net_socket wraps socket() and optionally delegates to the Go network interface.
//
//export lib_net_socket
func lib_net_socket(domain, typ, proto C.int) C.int {
	if DefaultXash3D.Net == nil {
		return C.socket(domain, typ, proto)
	}
	return C.int(DefaultXash3D.Net.Socket(int(domain), int(typ), int(proto)))
}

// lib_net_closesocket closes a socket via Go or native syscall.
//
//export lib_net_closesocket
func lib_net_closesocket(fd C.int) C.int {
	if DefaultXash3D.Net == nil {
		return C.close(fd)
	}
	return C.int(DefaultXash3D.Net.CloseSocket(int(fd)))
}

// lib_net_sendto sends data to a remote address, optionally via Go logic.
//
//export lib_net_sendto
func lib_net_sendto(fd C.int, buf unsafe.Pointer, length C.size_t, flags C.int, sockaddr unsafe.Pointer, socklen C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		ret := C.sendto(fd, buf, length, flags, (*C.struct_sockaddr)(unsafe.Pointer(sockaddr)), socklen)
		return C.int(ret)
	}

	if buf == nil || length == 0 {
		return C.int(0)
	}

	data := unsafe.Slice((*byte)(buf), int(length))
	addr := parseIPv4Sockaddr(sockaddr)
	return C.int(DefaultXash3D.Net.SendTo(int(fd), Packet{Data: data, Addr: addr}, int(flags)))
}

// lib_net_sendto_batch sends multiple packets in one call.
// It uses a zero-copy mechanism to avoid memory overhead.
//
//export lib_net_sendto_batch
func lib_net_sendto_batch(
	fd C.int,
	packets **C.char,
	sizes *C.int,
	count C.int,
	flags C.int,
	to *C.struct_sockaddr_storage,
	tolen C.int,
) C.int {
	if packets == nil || sizes == nil || count <= 0 {
		return C.int(0)
	}

	ci := int(count)

	getPktPtr := func(base unsafe.Pointer, i int) *C.char {
		ptr := (**C.char)(unsafe.Pointer(uintptr(base) + uintptr(i)*unsafe.Sizeof(uintptr(0))))
		return *ptr
	}

	getSize := func(base unsafe.Pointer, i int) C.int {
		ptr := (*C.int)(unsafe.Pointer(uintptr(base) + uintptr(i)*unsafe.Sizeof(C.int(0))))
		return *ptr
	}

	if DefaultXash3D.Net == nil {
		var totalSent int64 = 0
		for i := 0; i < ci; i++ {
			p := getPktPtr(unsafe.Pointer(packets), i)
			sz := getSize(unsafe.Pointer(sizes), i)
			if p == nil || sz <= 0 {
				continue
			}
			ret := C.sendto(fd, unsafe.Pointer(p), C.size_t(sz), flags, (*C.struct_sockaddr)(unsafe.Pointer(to)), C.socklen_t(tolen))
			if ret < 0 {
				return C.int(-1)
			}
			totalSent += int64(ret)
		}
		if totalSent > int64(^C.int(0)) {
			return C.int(^C.int(0))
		}
		return C.int(totalSent)
	}

	countInt := ci
	if countInt > len(sendBatch) {
		countInt = len(sendBatch)
	}

	addr := parseIPv4Sockaddr(unsafe.Pointer(to))

	for i := 0; i < countInt; i++ {
		p := getPktPtr(unsafe.Pointer(packets), i)
		sz := getSize(unsafe.Pointer(sizes), i)

		if p == nil || sz <= 0 {
			sendBatch[i].Data = nil
			sendBatch[i].Addr = addr
			continue
		}

		data := unsafe.Slice((*byte)(unsafe.Pointer(p)), int(sz))
		sendBatch[i].Data = data
		sendBatch[i].Addr = addr
	}

	return C.int(DefaultXash3D.Net.SendToBatch(int(fd), sendBatch[:countInt], int(flags)))
}

// lib_net_recvfrom receives data into a provided buffer and writes the sender address.
//
//export lib_net_recvfrom
func lib_net_recvfrom(fd C.int, buf unsafe.Pointer, length C.size_t, flags C.int, sockaddr unsafe.Pointer, socklen *C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		ret := C.recvfrom(fd, buf, length, flags, (*C.struct_sockaddr)(unsafe.Pointer(sockaddr)), socklen)
		return C.int(ret)
	}

	if buf == nil || length == 0 {
		return C.int(0)
	}

	goBuf := unsafe.Slice((*byte)(buf), int(length))
	pkt := DefaultXash3D.Net.RecvFrom()
	if pkt == nil {
		C.set_errno(C.EAGAIN)
		return C.int(-1)
	}

	copyLen := len(pkt.Data)
	if copyLen > int(length) {
		copyLen = int(length)
	}
	if copyLen > 0 {
		copy(goBuf[:copyLen], pkt.Data[:copyLen])
	}

	if sockaddr != nil && socklen != nil {
		writeIPv4Sockaddr(sockaddr, socklen, pkt.Addr)
	}

	return C.int(copyLen)
}

// lib_net_bind binds a socket to a given address.
//
//export lib_net_bind
func lib_net_bind(fd C.int, sockaddr unsafe.Pointer, socklen C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.bind(fd, (*C.struct_sockaddr)(unsafe.Pointer(sockaddr)), socklen)
	}
	addr := parseIPv4Sockaddr(sockaddr)
	return C.int(DefaultXash3D.Net.Bind(int(fd), addr))
}

// lib_net_getsockname retrieves the local address of a socket.
//
//export lib_net_getsockname
func lib_net_getsockname(fd C.int, sockaddr unsafe.Pointer, socklen *C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.getsockname(fd, (*C.struct_sockaddr)(unsafe.Pointer(sockaddr)), socklen)
	}
	addr := DefaultXash3D.Net.GetSockName(int(fd))
	if addr == nil {
		return C.int(-1)
	}
	writeIPv4Sockaddr(sockaddr, socklen, *addr)
	return C.int(0)
}

// lib_net_gethostbyname resolves a hostname via Go or falls back to C.
//
//export lib_net_gethostbyname
func lib_net_gethostbyname(hostname *C.char) C.int {
	if DefaultXash3D.Net == nil {
		h := C.gethostbyname(hostname)
		if h == nil {
			return C.int(-1)
		}
		return C.int(0)
	}
	goHost := C.GoString(hostname)
	return C.int(DefaultXash3D.Net.GetHostByName(goHost))
}

// lib_net_gethostname returns the configured hostname as a string.
//
//export lib_net_gethostname
func lib_net_gethostname(name *C.char, namelen C.size_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.gethostname(name, namelen)
	}
	hostname := DefaultXash3D.Net.GetHostName()
	hb := []byte(hostname)
	n := len(hb)
	if n >= int(namelen) {
		if namelen == 0 {
			return C.int(-1)
		}
		n = int(namelen) - 1
	}
	if n > 0 {
		C.memcpy(unsafe.Pointer(name), unsafe.Pointer(&hb[0]), C.size_t(n))
	}
	*(*byte)(unsafe.Add(unsafe.Pointer(name), uintptr(n))) = 0
	return C.int(n)
}

// lib_net_getaddrinfo fills a struct addrinfo with a synthesized address based on host ID.
//
//export lib_net_getaddrinfo
func lib_net_getaddrinfo(hostname, service *C.char, hints, result unsafe.Pointer) C.int {
	if DefaultXash3D.Net == nil {
		return C.getaddrinfo(hostname, service, (*C.struct_addrinfo)(hints), (**C.struct_addrinfo)(result))
	}

	host := ""
	if hostname != nil {
		host = C.GoString(hostname)
	}
	id := DefaultXash3D.Net.GetAddrInfo(host)

	sa := C.malloc(C.size_t(unsafe.Sizeof(C.struct_sockaddr_in{})))
	if sa == nil {
		return C.EAI_MEMORY
	}

	var sin C.struct_sockaddr_in
	sin.sin_family = C.AF_INET
	sin.sin_port = 0

	o1, o2, o3, o4 := uint32(101), uint32(101), uint32(id&0xff), uint32(0)
	ipHostOrder := (o1 << 24) | (o2 << 16) | (o3 << 8) | o4
	sin.sin_addr.s_addr = C.htonl(C.uint32_t(ipHostOrder))

	C.memcpy(sa, unsafe.Pointer(&sin), C.size_t(unsafe.Sizeof(sin)))

	ai := C.malloc(C.size_t(unsafe.Sizeof(C.struct_addrinfo{})))
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

	*(*unsafe.Pointer)(result) = unsafe.Pointer(aiStruct)
	return C.int(0)
}
