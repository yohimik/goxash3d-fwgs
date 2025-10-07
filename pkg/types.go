package goxash3d_fwgs

type Xash3DNetwork interface {
	Socket(domain, typ, proto int) int
	CloseSocket(fd int) int
	SendTo(fd int, buf []byte, flags int, sockaddr []byte) int
	SendToBatch(fd int, buf [][]byte, flags int, sockaddr []byte) int
	RecvFrom(fd int, buf []byte, flags int) (int, []byte)
	Bind(fd int, sockaddr []byte) int
	GetSockName(fd int) ([]byte, int)
	GetHostByName(host string) int
	GetHostName(name []byte) int
	GetAddrInfo(hostname, service string, hints, result uintptr) int
}
