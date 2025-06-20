package main

/*
#cgo LDFLAGS: -L. -lxash -lpublic -lbuild_vcs -lm -lbacktrace
#include "xash.h"
#include <stdlib.h>
#include <errno.h>
#include <stdint.h>
#include <string.h>
#include <arpa/inet.h>
#include <sys/socket.h>
#include <netinet/in.h>

static void set_errno(int err) {
	errno = err;
}
*/
import "C"
import (
	"fmt"
	"github.com/pion/webrtc/v4"
	"os"
	"unsafe"
)

// Packet represents a received message
type Packet struct {
	IP   []byte
	Data []byte
}

//export __wrap_recvfrom
func __wrap_recvfrom(sockfd C.int, buf unsafe.Pointer, length C.size_t, flags C.int, src_addr *C.struct_sockaddr, addrlen *C.socklen_t) C.ssize_t {
	var pkt Packet

	select {
	case pkt = <-incoming:
		// Received a packet
	default:
		// No data available: simulate EWOULDBLOCK
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
		csa.sin_port = C.htons(12345) // use dummy port or real one
		ip := uint32(pkt.IP[0])<<24 | uint32(pkt.IP[1])<<16 | uint32(pkt.IP[2])<<8 | uint32(pkt.IP[3])
		csa.sin_addr.s_addr = C.uint32_t(C.htonl(C.uint32_t(ip)))
		*addrlen = C.socklen_t(unsafe.Sizeof(*csa))
	}

	return C.ssize_t(n)
}

//export __wrap_sendto
func __wrap_sendto(sockfd C.int, buf unsafe.Pointer, length C.size_t, flags C.int, dest unsafe.Pointer, addrlen C.socklen_t) C.ssize_t {
	if buf == nil || dest == nil || length <= 0 {
		return 0
	}
	if length <= 0 {
		return 0
	}
	sa := (*C.struct_sockaddr_in)(dest)
	ipBytes := (*[4]byte)(unsafe.Pointer(&sa.sin_addr))[:4:4]
	ip := fmt.Sprintf("%d.%d.%d.%d", ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3])
	channel, ok := connections.Load(ip)
	if !ok || channel == nil {
		return 0
	}
	c, ok := channel.(*webrtc.DataChannel)
	if !ok {
		return 0
	}
	data := C.GoBytes(buf, C.int(length))
	c.Send(data)
	return C.ssize_t(length)
}

func runEngine() {
	args := os.Args

	argc := C.int(len(args))
	argv := make([]*C.char, len(args)+1)

	for i, s := range args {
		argv[i] = C.CString(s)
		defer C.free(unsafe.Pointer(argv[i]))
	}
	argv[len(args)] = nil

	C.Launcher_Main(argc, &argv[0])
}
