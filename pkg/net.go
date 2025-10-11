package goxash3d_fwgs

/*
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <netdb.h>
#include <unistd.h>
#include <errno.h>

static void set_errno(int err) {
	errno = err;
}
*/
import "C"

import (
	"unsafe"
)

type Packet struct {
	Data []byte
	Addr Addr
}

type Addr struct {
	IP   [4]byte
	Port uint16
}

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

// package-level reusable buffer for batch sends (avoid allocations)
var (
	sendBatch [1024]Packet
)

// parseIPv4Sockaddr attempts to parse an IPv4 sockaddr pointer.
// If sockaddr is nil or not AF_INET it returns zero Addr.
func parseIPv4Sockaddr(sockaddr unsafe.Pointer) Addr {
	var out Addr
	if sockaddr == nil {
		return out
	}

	// Treat the sockaddr pointer as struct sockaddr to inspect family safely
	sa := (*C.struct_sockaddr)(sockaddr)
	if sa.sa_family != C.AF_INET {
		// Not IPv4 -> return zero Addr
		return out
	}

	// Cast to sockaddr_in now that family is AF_INET
	sin := (*C.struct_sockaddr_in)(sockaddr)

	// sin_addr.s_addr is stored in network byte order
	ipN := uint32(C.ntohl(sin.sin_addr.s_addr))
	out.IP = [4]byte{
		byte((ipN >> 24) & 0xFF),
		byte((ipN >> 16) & 0xFF),
		byte((ipN >> 8) & 0xFF),
		byte(ipN & 0xFF),
	}

	// sin_port is network order -> ntohs
	out.Port = uint16(C.ntohs(sin.sin_port))
	return out
}

// writeIPv4Sockaddr writes IPv4 sockaddr_in into the provided sockaddr buffer.
// It copies at most *socklen bytes and updates *socklen to the number of bytes written.
// Returns the number of bytes written.
func writeIPv4Sockaddr(sockaddr unsafe.Pointer, socklen *C.socklen_t, addr Addr) C.socklen_t {
	if sockaddr == nil || socklen == nil {
		return 0
	}

	var sin C.struct_sockaddr_in
	sin.sin_family = C.AF_INET
	// port must be network order
	sin.sin_port = C.htons(C.ushort(addr.Port))

	// compose ip as uint32 then htonl; ip order in Addr is [a,b,c,d] as usual
	ip := (uint32(addr.IP[0]) << 24) | (uint32(addr.IP[1]) << 16) | (uint32(addr.IP[2]) << 8) | uint32(addr.IP[3])
	sin.sin_addr.s_addr = C.htonl(C.uint32_t(ip))

	required := C.socklen_t(unsafe.Sizeof(sin))

	// If caller provided shorter buffer, copy only up to provided size.
	if *socklen < required {
		C.memcpy(sockaddr, unsafe.Pointer(&sin), C.size_t(*socklen))
		written := *socklen
		*socklen = written
		return written
	}

	// Enough space: copy full struct
	C.memcpy(sockaddr, unsafe.Pointer(&sin), C.size_t(required))
	*socklen = required
	return required
}

// ---- exported C functions ----

//export lib_net_socket
func lib_net_socket(domain, typ, proto C.int) C.int {
	if DefaultXash3D.Net == nil {
		return C.socket(domain, typ, proto)
	}
	return C.int(DefaultXash3D.Net.Socket(int(domain), int(typ), int(proto)))
}

//export lib_net_closesocket
func lib_net_closesocket(fd C.int) C.int {
	if DefaultXash3D.Net == nil {
		return C.close(fd)
	}
	return C.int(DefaultXash3D.Net.CloseSocket(int(fd)))
}

//export lib_net_sendto
func lib_net_sendto(fd C.int, buf unsafe.Pointer, length C.size_t, flags C.int, sockaddr unsafe.Pointer, socklen C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		ret := C.sendto(fd, buf, length, flags, (*C.struct_sockaddr)(unsafe.Pointer(sockaddr)), socklen)
		return C.int(ret)
	}

	// If buf is nil or length is zero -> nothing to send (match C behavior)
	if buf == nil || length == 0 {
		return C.int(0)
	}

	// Create a Go slice that points to the C buffer. This is zero-copy: valid as long as the callee doesn't retain it.
	data := unsafe.Slice((*byte)(buf), int(length))
	addr := parseIPv4Sockaddr(sockaddr)
	return C.int(DefaultXash3D.Net.SendTo(int(fd), Packet{Data: data, Addr: addr}, int(flags)))
}

//export lib_net_sendto_batch
func lib_net_sendto_batch(fd C.int, packets **C.char, sizes *C.int, count C.int, flags C.int, to *C.struct_sockaddr_storage, tolen C.int) C.int {
	// If no Go override, implement batch by calling sendto repeatedly (portable)
	if DefaultXash3D.Net == nil {
		if packets == nil || sizes == nil || count <= 0 {
			return C.int(0)
		}
		ci := int(count)

		// Access C arrays safely via pointer-to-array trick
		pktPtrs := (*[1 << 30]*C.char)(unsafe.Pointer(packets))[:ci:ci]
		sz := (*[1 << 30]C.int)(unsafe.Pointer(sizes))[:ci:ci]

		var totalSent int64 = 0
		for i := 0; i < ci; i++ {
			if pktPtrs[i] == nil || sz[i] <= 0 {
				continue
			}
			// Cast sockaddr_storage -> sockaddr via unsafe.Pointer
			ret := C.sendto(fd, unsafe.Pointer(pktPtrs[i]), C.size_t(sz[i]), flags, (*C.struct_sockaddr)(unsafe.Pointer(to)), C.socklen_t(tolen))
			if ret < 0 {
				// return -1 on error (match typical C behavior)
				return C.int(-1)
			}
			// accumulate into Go int64 and avoid direct C-type conversions
			totalSent += int64(ret)
		}
		return C.int(totalSent)
	}

	// Use our package-level buffer to avoid allocations
	countInt := int(count)
	if countInt > len(sendBatch) {
		countInt = len(sendBatch)
	}

	addr := parseIPv4Sockaddr(unsafe.Pointer(to))

	if packets == nil || sizes == nil || countInt == 0 {
		return C.int(0)
	}

	// Convert incoming C arrays to Go slices without allocation
	pktPtrs := (*[1 << 30]*C.char)(unsafe.Pointer(packets))[:countInt:countInt]
	sz := (*[1 << 30]C.int)(unsafe.Pointer(sizes))[:countInt:countInt]

	for i := 0; i < countInt; i++ {
		if pktPtrs[i] == nil || sz[i] <= 0 {
			sendBatch[i].Data = nil
			sendBatch[i].Addr = addr
			continue
		}
		// Zero-copy slice referencing C memory
		data := unsafe.Slice((*byte)(unsafe.Pointer(pktPtrs[i])), int(sz[i]))
		sendBatch[i].Data = data
		sendBatch[i].Addr = addr
	}

	return C.int(DefaultXash3D.Net.SendToBatch(int(fd), sendBatch[:countInt], int(flags)))
}

//export lib_net_recvfrom
func lib_net_recvfrom(fd C.int, buf unsafe.Pointer, length C.size_t, flags C.int, sockaddr unsafe.Pointer, socklen *C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		ret := C.recvfrom(fd, buf, length, flags, (*C.struct_sockaddr)(unsafe.Pointer(sockaddr)), socklen)
		return C.int(ret)
	}

	if buf == nil || length == 0 {
		// nothing to write to user buffer
		return C.int(0)
	}

	goBuf := unsafe.Slice((*byte)(buf), int(length))
	pkt := DefaultXash3D.Net.RecvFrom()
	if pkt == nil {
		// indicate would-block / no data
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

	// write sockaddr if requested
	if sockaddr != nil && socklen != nil {
		writeIPv4Sockaddr(sockaddr, socklen, pkt.Addr)
	}

	return C.int(copyLen)
}

//export lib_net_bind
func lib_net_bind(fd C.int, sockaddr unsafe.Pointer, socklen C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.bind(fd, (*C.struct_sockaddr)(unsafe.Pointer(sockaddr)), socklen)
	}
	addr := parseIPv4Sockaddr(sockaddr)
	return C.int(DefaultXash3D.Net.Bind(int(fd), addr))
}

//export lib_net_getsockname
func lib_net_getsockname(fd C.int, sockaddr unsafe.Pointer, socklen *C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.getsockname(fd, (*C.struct_sockaddr)(unsafe.Pointer(sockaddr)), socklen)
	}
	addr := DefaultXash3D.Net.GetSockName(int(fd))
	if addr == nil {
		// indicate error to C
		return C.int(-1)
	}
	writeIPv4Sockaddr(sockaddr, socklen, *addr)
	return C.int(0)
}

//export lib_net_gethostbyname
func lib_net_gethostbyname(hostname *C.char) C.int {
	if DefaultXash3D.Net == nil {
		// C.gethostbyname returns a pointer; return success (0) if non-nil, -1 on failure
		h := C.gethostbyname(hostname)
		if h == nil {
			return C.int(-1)
		}
		return C.int(0)
	}
	goHost := C.GoString(hostname)
	return C.int(DefaultXash3D.Net.GetHostByName(goHost))
}

//export lib_net_gethostname
func lib_net_gethostname(name *C.char, namelen C.size_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.gethostname(name, namelen)
	}
	hostname := DefaultXash3D.Net.GetHostName()
	hb := []byte(hostname)
	n := len(hb)
	if n >= int(namelen) {
		// keep space for terminating NUL
		if namelen == 0 {
			return C.int(-1)
		}
		n = int(namelen) - 1
	}
	if n > 0 {
		C.memcpy(unsafe.Pointer(name), unsafe.Pointer(&hb[0]), C.size_t(n))
	}
	// terminate
	*(*byte)(unsafe.Add(unsafe.Pointer(name), uintptr(n))) = 0
	return C.int(n)
}

//export lib_net_getaddrinfo
func lib_net_getaddrinfo(hostname, service *C.char, hints, result unsafe.Pointer) C.int {
	if DefaultXash3D.Net == nil {
		return C.getaddrinfo(hostname, service, (*C.struct_addrinfo)(hints), (**C.struct_addrinfo)(result))
	}

	// Get host ID from Go
	host := ""
	if hostname != nil {
		host = C.GoString(hostname)
	}
	id := DefaultXash3D.Net.GetAddrInfo(host)

	// Allocate sockaddr_in in C memory
	sa := C.malloc(C.size_t(unsafe.Sizeof(C.struct_sockaddr_in{})))
	if sa == nil {
		return C.EAI_MEMORY
	}

	// Compose sockaddr_in (IPv4)
	var sin C.struct_sockaddr_in
	sin.sin_family = C.AF_INET
	sin.sin_port = 0

	// Compose IP address e.g., 101.101.(id & 0xff).0  (as the original intended pattern)
	// Build in host-order then convert to network order via htonl
	o1 := uint32(101)
	o2 := uint32(101)
	o3 := uint32(id & 0xff)
	o4 := uint32(0)
	ipHostOrder := (o1 << 24) | (o2 << 16) | (o3 << 8) | o4
	sin.sin_addr.s_addr = C.htonl(C.uint32_t(ipHostOrder))

	// Copy sin into the allocated memory
	C.memcpy(sa, unsafe.Pointer(&sin), C.size_t(unsafe.Sizeof(sin)))

	// Allocate addrinfo in C memory
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

	// Write addrinfo pointer to result (result is **addrinfo)
	*(*unsafe.Pointer)(result) = unsafe.Pointer(aiStruct)
	return C.int(0)
}
