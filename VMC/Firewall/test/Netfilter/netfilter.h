/*
   Copyright 2014 Krishna Raman <kraman@gmail.com>

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

#ifndef _NETFILTER_H
#define _NETFILTER_H

#include <stdio.h>
#include <stdlib.h>
#include <math.h>
#include <unistd.h>
#include <netinet/in.h>
#include <linux/types.h>
#include <linux/socket.h>
#include <linux/netfilter.h>
#include <libnetfilter_queue/libnetfilter_queue.h>

extern uint go_callback(int id, unsigned char* data, int len, void** cb_func, unsigned char** newData, unsigned int* size);

static int nf_callback(struct nfq_q_handle *qh, struct nfgenmsg *nfmsg, struct nfq_data *nfa, void *cb_func){
    uint32_t id = -1;
    struct nfqnl_msg_packet_hdr *ph = NULL;
    unsigned char *buffer = NULL;
	unsigned char *newData = NULL;
	unsigned int size = 0;
    int ret = 0;
    int verdict = 0;
	//int i;

    ph = nfq_get_msg_packet_hdr(nfa);
    id = ntohl(ph->packet_id);

    ret = nfq_get_payload(nfa, &buffer);
    verdict = go_callback(id, buffer, ret, cb_func, &newData, &size);
/* 
	printf("new data size is %d, verdict %d, location %p\n", size, verdict, newData);
	printf("[ ");
	fflush(stdout);

	for (i=0; i<size;i++) {
		printf("%02x ", newData[i] & 0xff);
	}
	printf("]\n");
*/	

    return nfq_set_verdict(qh, id, verdict, size, newData);
}

static inline struct nfq_q_handle* CreateQueue(struct nfq_handle *h, u_int16_t queue, void* cb_func)
{
    return nfq_create_queue(h, queue, &nf_callback, cb_func);
}

static inline void Run(struct nfq_handle *h, int fd)
{
    char buf[4096] __attribute__ ((aligned));
    int rv;

    while ((rv = recv(fd, buf, sizeof(buf), 0)) && rv >= 0) {
        nfq_handle_packet(h, buf, rv);
    }
}

#endif


