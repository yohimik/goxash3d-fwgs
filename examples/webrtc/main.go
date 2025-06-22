package main

/*
#include <sys/socket.h>
*/
import "C"
import (
	goxash3d_fwgs "github.com/yohimik/goxash3d-fwgs/pkg"
	"unsafe"
)

//export __wrap_recvfrom
func __wrap_recvfrom(
	sockfd C.int,
	buf unsafe.Pointer,
	length C.size_t,
	flags C.int,
	src_addr unsafe.Pointer,
	addrlen unsafe.Pointer,
) C.ssize_t {
	return C.ssize_t(goxash3d_fwgs.DefaultXash3D.Recvfrom(
		goxash3d_fwgs.Int(sockfd),
		buf,
		goxash3d_fwgs.SizeT(length),
		goxash3d_fwgs.Int(flags),
		(*goxash3d_fwgs.Sockaddr)(src_addr),
		(*goxash3d_fwgs.SocklenT)(addrlen),
	))
}

//export __wrap_sendto
func __wrap_sendto(
	sockfd C.int,
	buf unsafe.Pointer,
	length C.size_t,
	flags C.int,
	dest unsafe.Pointer,
	addrlen C.socklen_t,
) C.ssize_t {
	return C.ssize_t(goxash3d_fwgs.DefaultXash3D.Sendto(
		goxash3d_fwgs.Int(sockfd),
		buf,
		goxash3d_fwgs.SizeT(length),
		goxash3d_fwgs.Int(flags),
		dest,
		goxash3d_fwgs.SocklenT(addrlen),
	))
}

func main() {
	go runSFU()

	goxash3d_fwgs.DefaultXash3D.SysStart()
}
