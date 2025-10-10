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
	Bind(fd int, add Addr) int
	GetSockName(fd int) *Addr
	GetHostByName(host string) int
	GetHostName() string
	GetAddrInfo(host string) uint8
}

var (
	sendBatch [1024]Packet
)

func parseIPv4Sockaddr(sockaddr unsafe.Pointer) Addr {
	sin := (*C.struct_sockaddr_in)(sockaddr)
	a := uint32(sin.sin_addr.s_addr)
	return Addr{
		IP: [4]byte{
			byte(a >> 24), byte(a >> 16), byte(a >> 8), byte(a),
		},
		Port: uint16(C.ntohs(sin.sin_port)),
	}
}

func writeIPv4Sockaddr(sockaddr unsafe.Pointer, socklen *C.socklen_t, addr Addr) C.socklen_t {
	var sin C.struct_sockaddr_in
	sin.sin_family = C.AF_INET
	sin.sin_port = C.htons(C.ushort(addr.Port))
	sin.sin_addr.s_addr = C.uint32_t(addr.IP[0])<<24 | C.uint32_t(addr.IP[1])<<16 | C.uint32_t(addr.IP[2])<<8 | C.uint32_t(addr.IP[3])
	wlen := C.socklen_t(unsafe.Sizeof(sin))
	if *socklen < wlen {
		C.memcpy(sockaddr, unsafe.Pointer(&sin), C.size_t(*socklen))
		return *socklen
	}
	C.memcpy(sockaddr, unsafe.Pointer(&sin), C.size_t(wlen))
	*socklen = wlen
	return wlen
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
		return C.closesocket(fd)
	}
	return C.int(DefaultXash3D.Net.CloseSocket(int(fd)))
}

//export lib_net_sendto
func lib_net_sendto(fd C.int, buf unsafe.Pointer, length C.size_t, flags C.int, sockaddr unsafe.Pointer, socklen C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.sendto(fd, buf, length, flags, sockaddr, socklen)
	}
	data := unsafe.Slice((*byte)(buf), int(length))
	addr := parseIPv4Sockaddr(sockaddr)
	return C.int(DefaultXash3D.Net.SendTo(int(fd), Packet{Data: data, Addr: addr}, int(flags)))
}

//export lib_net_sendto_batch
func lib_net_sendto_batch(fd C.int, packets **C.char, sizes *C.int, count C.int, flags C.int, to *C.struct_sockaddr_storage, tolen C.int) C.int {
	if DefaultXash3D.Net == nil {
		return C.sendto_batch(fd, packets, sizes, count, flags, to, tolen)
	}

	countInt := int(count)
	if countInt > len(sendBatch) {
		countInt = len(sendBatch)
	}

	addr := parseIPv4Sockaddr(unsafe.Pointer(to))

	pktPtrs := unsafe.Slice(packets, countInt)
	sz := unsafe.Slice(sizes, countInt)

	for i := 0; i < countInt; i++ {
		data := unsafe.Slice((*byte)(unsafe.Pointer(pktPtrs[i])), int(sz[i]))
		sendBatch[i].Data = data
		sendBatch[i].Addr = addr
	}

	return C.int(DefaultXash3D.Net.SendToBatch(int(fd), sendBatch[:countInt], int(flags)))
}

//export lib_net_recvfrom
func lib_net_recvfrom(fd C.int, buf unsafe.Pointer, length C.size_t, flags C.int, sockaddr unsafe.Pointer, socklen *C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.recvfrom(fd, buf, length, flags, sockaddr, socklen)
	}
	goBuf := unsafe.Slice((*byte)(buf), int(length))
	pkt := DefaultXash3D.Net.RecvFrom()
	if pkt == nil {
		C.errno = C.int(C.EAGAIN)
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

//export lib_net_bind
func lib_net_bind(fd C.int, sockaddr unsafe.Pointer, socklen C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.bind(fd, sockaddr, socklen)
	}
	addr := parseIPv4Sockaddr(sockaddr)
	return C.int(DefaultXash3D.Net.Bind(int(fd), addr))
}

//export lib_net_getsockname
func lib_net_getsockname(fd C.int, sockaddr unsafe.Pointer, socklen *C.socklen_t) C.int {
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

//export lib_net_gethostbyname
func lib_net_gethostbyname(hostname *C.char) C.int {
	if DefaultXash3D.Net == nil {
		return C.gethostbyname(hostname)
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
		n = int(namelen) - 1
	}
	C.memcpy(unsafe.Pointer(name), unsafe.Pointer(&hb[0]), C.size_t(n))
	*(*byte)(unsafe.Add(unsafe.Pointer(name), uintptr(n))) = 0
	return C.int(n)
}

//export lib_net_getaddrinfo
func lib_net_getaddrinfo(hostname, service *C.char, hints, result unsafe.Pointer) C.int {
	if DefaultXash3D.Net == nil {
		return C.getaddrinfo(hostname, service, (*C.struct_addrinfo)(hints), (**C.struct_addrinfo)(result))
	}

	// Get host ID from Go
	host := C.GoString(hostname)
	id := DefaultXash3D.Net.GetAddrInfo(host)

	// Allocate sockaddr_in in C memory
	sa := C.malloc(C.size_t(unsafe.Sizeof(C.struct_sockaddr_in{})))
	if sa == nil {
		return C.EAI_MEMORY
	}

	var sin C.struct_sockaddr_in
	sin.sin_family = C.AF_INET
	sin.sin_port = 0

	// Compose IP as 101.101.(id & 0xff).0
	a := id & 0xff
	b := byte(0)
	sin.sin_addr.s_addr = C.uint32_t(a)<<24 | C.uint32_t(b)<<16 | 101<<8 | 101

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

	// Write addrinfo pointer to result
	*(*unsafe.Pointer)(result) = unsafe.Pointer(aiStruct)
	return 0
}
