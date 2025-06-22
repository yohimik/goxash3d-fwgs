package goxash3d_fwgs

/*
#include <sys/socket.h>
*/
import "C"

type (
	Int      = C.int
	SizeT    = C.size_t
	SsizeT   = C.ssize_t
	Sockaddr = C.struct_sockaddr
	SocklenT = C.socklen_t
)
