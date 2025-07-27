#ifndef NET_H
#define NET_H

#ifdef __cplusplus
extern "C" {
#endif

#include <sys/types.h>
#include <sys/socket.h>

typedef int (*recvfrom_func_t)(
    int sockfd,
    void *buf,
    int len,
    int flags,
    struct sockaddr *src_addr,
    socklen_t *addrlen
);

void RegisterRecvFromCallback(recvfrom_func_t fn);

typedef int (*sendto_func_t)(
    int sockfd,
    char **packets,
    size_t *sizes,
    int packet_count,
    int seq_num,
	struct sockaddr_storage *to,
	size_t to_len
);

void RegisterSendToCallback(sendto_func_t fn);

// callbacks.c or other C file
extern int Recvfrom(int sockfd, void* buf, int len, int flags, struct sockaddr* src_addr, socklen_t* addrlen);
extern int Sendto(int sockfd, char **packets, size_t *sizes, int packet_count, int seq_num, struct sockaddr_storage *to, size_t to_len);


#ifdef __cplusplus
}
#endif

#endif // NET_H