package goxash3d_fwgs

/*
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <netdb.h>
#include <unistd.h>
*/
import "C"

import (
	"unsafe"
)

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
func go_net_sendto(fd C.int, buf unsafe.Pointer, len C.size_t, flags C.int, sockaddr unsafe.Pointer, socklen C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.sendto(fd, buf, len, flags, sockaddr, socklen)
	}
	data := unsafe.Slice((*byte)(buf), len) // Directly use C buffer as Go slice
	addr := unsafe.Slice((*byte)(sockaddr), socklen)
	return C.int(DefaultXash3D.Net.SendTo(int(fd), data, int(flags), addr))
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
		return C.sendto_batch(fd, packets, sizes, count, flags, to, tolen)
	}
	// Build Go slice headers for packet pointers and sizes (no allocations)
	pktPtrs := unsafe.Slice(packets, count)
	sz := unsafe.Slice(sizes, count)

	// Use sockaddr directly (no copy)
	sa := unsafe.Slice((*byte)(unsafe.Pointer(to)), tolen)

	flagsInt := int(flags)
	fdInt := int(fd)
	n := 0
	for i := 0; i < int(count); i++ {
		n += DefaultXash3D.Net.SendTo(fdInt, unsafe.Slice((*byte)(unsafe.Pointer(pktPtrs[i])), int(sz[i])), flagsInt, sa)
	}

	return C.int(n)
}

//export go_net_recvfrom
func go_net_recvfrom(fd C.int, buf unsafe.Pointer, len C.size_t, flags C.int, sockaddr unsafe.Pointer, socklen *C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.recvfrom(fd, buf, len, flags, sockaddr, socklen)
	}
	goBuf := unsafe.Slice((*byte)(buf), len) // use C buffer directly
	n, sa := DefaultXash3D.Net.RecvFrom(int(fd), goBuf, int(flags))
	if n < 0 {
		return -1
	}
	if sockaddr != nil && socklen != nil && len(sa) > 0 {
		copyLen := int(*socklen)
		if copyLen > len(sa) {
			copyLen = len(sa)
		}
		C.memcpy(sockaddr, unsafe.Pointer(&sa[0]), C.size_t(copyLen))
		*socklen = C.socklen_t(copyLen)
	}
	return C.int(n)
}

//export go_net_bind
func go_net_bind(fd C.int, sockaddr unsafe.Pointer, socklen C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.bind(fd, sockaddr, socklen)
	}
	sa := C.GoBytes(sockaddr, C.int(socklen))
	return C.int(DefaultXash3D.Net.Bind(int(fd), sa))
}

//export go_net_getsockname
func go_net_getsockname(fd C.int, sockaddr unsafe.Pointer, socklen *C.socklen_t) C.int {
	if DefaultXash3D.Net == nil {
		return C.getsockname(fd, sockaddr, socklen)
	}
	sa, n := DefaultXash3D.Net.GetSockName(int(fd))
	if sockaddr != nil && socklen != nil && n > 0 {
		copyLen := int(*socklen)
		if copyLen > len(sa) {
			copyLen = len(sa)
		}
		C.memcpy(sockaddr, unsafe.Pointer(&sa[0]), C.size_t(copyLen))
		*socklen = C.socklen_t(copyLen)
	}
	return 0
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
	buf := make([]byte, int(namelen))
	n := DefaultXash3D.Net.GetHostName(buf)
	if n < 0 {
		return -1
	}
	if name != nil {
		C.memcpy(unsafe.Pointer(name), unsafe.Pointer(&buf[0]), C.size_t(n))
	}
	return C.int(n)
}

//export go_net_getaddrinfo
func go_net_getaddrinfo(hostname, service *C.char, hints, result unsafe.Pointer) C.int {
	if DefaultXash3D.Net == nil {
		return C.getaddrinfo(hostname, service, hints, result)
	}
	host := C.GoString(hostname)
	svc := C.GoString(service)
	return C.int(DefaultXash3D.Net.GetAddrInfo(host, svc, uintptr(hints), uintptr(result)))
}
