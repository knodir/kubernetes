
inp <- scan("packet_latency_sample.txt", list(0,0,0))
server_time<-inp[[1]]; latency<-inp[[3]]

to_seconds = 1000000000
to_ms = 1000000
to_us = 1000

server_time = server_time/to_ms
latency = latency/to_ms

png('latency.png', width = 6, height=5, units="in", res=300)

inp <- scan("nat_map_capacity_sample.txt", list(0,0))
nat_time<-inp[[1]]; cap<-inp[[2]]

nat_time = round(nat_time/to_ms)

# get unique time for each nat capacity
array_size = cap[length(cap)]

# this array's index is equal to capacity of the nat and value is equal to the latency of the packet when nat had such capacity 
arr = NULL
for (i in 1:array_size) arr[i] = 0;

for (item in 2:length(cap)) { # skip the first item since it is always zero
	if (arr[cap[item]] == 0) {
		temp_delta = 0
		for (latency_index in 1:length(server_time)) {
			# if difference between nat_time and server time is less than 1 millisecond, we assume those times to be equal and therefore take packet latency to be equal to that value
			if (abs(nat_time[item] - server_time[latency_index]) < 1) {
				arr[cap[item]] = latency[latency_index]
				break
			}
		}
	}		
}

sprintf("%f", arr)

plot(arr, type="o", col="blue", xlab="NAT capacity", ylab="Packet delay (ms)")
# # lines(fw2time, fw2rampct, type="o", col="red")
# # lines(fw3time, fw3rampct, type="o", col="green")
# # lines(fw4time, fw4rampct, type="o", col="black")

title(main="Latency")
# # legend(0,100, c("middlebox 1", "middlebox 2",  "middlebox 3", "middlebox 4"), col=c("blue","red","green", "black"), lty=1:1)

grid(col="gray")
dev.off()
