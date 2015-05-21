to_seconds = 1000000000
to_ms = 1000000
to_us = 1000

eachRun <- function(latencyFile, capacityFile) {
	# return array which has the natCapacityVSlatency data. Its index is equal to the capacity of the nat and value is equal to the latency of the packet (when nat had such capacity)
	retArray = NULL

	inp <- scan(latencyFile, list(0,0,0))
	server_time<-inp[[1]]; latency<-inp[[3]]


	server_time = server_time/to_ms
	latency = latency/to_ms


	inp <- scan(capacityFile, list(0,0))
	nat_time<-inp[[1]]; cap<-inp[[2]]

	nat_time = round(nat_time/to_ms)

	# get unique time for each nat capacity
	array_size = cap[length(cap)]

	for (i in 1:array_size) retArray[i] = 0;

	for (item in 2:length(cap)) { # skip the first item since it is always zero
		if (retArray[cap[item]] == 0) {
			for (latency_index in 1:length(server_time)) {
				# if difference between nat_time and server time is less than 1 millisecond, we assume those times to be equal and therefore take packet latency to be equal to that value
				if (abs(nat_time[item] - server_time[latency_index]) < 1) {
					retArray[cap[item]] = latency[latency_index]
					break
				}
			}
		}		
	}
	return(retArray)
}

arrFromFirstRun = eachRun("real_data/packet_latency1.txt", "real_data/nat_map_capacity1.txt")
arrFromSecondRun = eachRun("real_data/packet_latency2.txt", "real_data/nat_map_capacity2.txt")
arrFromThirdRun = eachRun("real_data/packet_latency3.txt", "real_data/nat_map_capacity3.txt")

# get the min number of NAT entries for each run and trim array size to be the same, i.e., [0:min]
minLength = min(length(arrFromFirstRun), length(arrFromSecondRun), length(arrFromThirdRun))

#sprintf("length(arrFromFirstRun) = %f", length(arrFromFirstRun))
#sprintf("length(arrFromSecondRun) = %f", length(arrFromSecondRun))
#sprintf("length(arrFromThirdRun) = %f", length(arrFromThirdRun))

# sprintf("minLength = %f", minLength)

# select point we would like to draw on the graph
step = 100
stepIndex = 0
avgVal = 0
cycles = 0
plotPoints = minLength / step

natCap = NULL
laten = NULL 
finalArr = NULL

trimmedFirst = NULL
trimmedSecond = NULL
trimmedThird = NULL

for (i in 1:length(arrFromFirstRun)) {
	avgVal = avgVal + arrFromFirstRun[i]
	cycles = cycles + 1
	if ((i %% step) == 0) { 
		stepIndex = stepIndex + 1
		trimmedFirst[stepIndex] = avgVal/cycles
		avgVal = 0
		cycles = 0
	}
}

# sprintf("trimmedFirst = %f", trimmedFirst)

avgVal = 0
stepIndex = 0
cycles = 0
for (i in 1:length(arrFromSecondRun)) {
	avgVal = avgVal + arrFromSecondRun[i]
	cycles = cycles + 1
	if ((i %% step) == 0) { 
		stepIndex = stepIndex + 1
		trimmedSecond[stepIndex] = avgVal/cycles
		avgVal = 0
		cycles = 0
	}
}

# sprintf("trimmedSecond = %f", trimmedSecond)

avgVal = 0
stepIndex = 0
cycles = 0
for (i in 1:length(arrFromThirdRun)) {
	avgVal = avgVal + arrFromThirdRun[i]
	cycles = cycles + 1
	if ((i %% step) == 0) { 
		stepIndex = stepIndex + 1
		trimmedThird[stepIndex] = avgVal/cycles
		avgVal = 0
		cycles = 0
	}
}

# sprintf("trimmedThird = %f", trimmedThird)

for (i in 1:plotPoints) {
    natCap[i] = i*step
    laten[i] = (trimmedFirst[i] + trimmedSecond[i] + trimmedThird[i]) / 3
}

png('latency.png', width = 6, height=5, units="in", res=300)

plot(natCap, laten, type="o", col="blue", xlab="NAT capacity", ylab="Packet delay (ms)")

# lines(fw2time, fw2rampct, type="o", col="red")
# lines(fw3time, fw3rampct, type="o", col="green")
# lines(fw4time, fw4rampct, type="o", col="black")

title(main="Latency")
# # legend(0,100, c("middlebox 1", "middlebox 2",  "middlebox 3", "middlebox 4"), col=c("blue","red","green", "black"), lty=1:1)

grid(col="gray")
dev.off()
