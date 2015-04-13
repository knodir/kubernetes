inp <- scan("serverlog_base1000msgps.txt", list(0,0,0,0))
latency_base<-inp[[3]]; throughput_base<-inp[[4]]

inp <- scan("serverlog_poll1000_1000msgps.txt", list(0,0,0,0))
latency_poll1000<-inp[[3]]; throughput_poll1000<-inp[[4]]

inp <- scan("serverlog_poll100_1000msgps.txt", list(0,0,0,0))
latency_poll100<-inp[[3]]; throughput_poll100<-inp[[4]]

inp <- scan("serverlog_cons_1000msgps.txt", list(0,0,0,0))
latency_cons<-inp[[3]]; throughput_cons<-inp[[4]]

png('latency.png', width = 5, height=5, units="in", res=300)
boxplot(latency_base/1000000, latency_poll1000/1000000, latency_poll100/1000000, latency_cons/1000000, ylab="latency (ms)", main="Latency Benchmark", col = c("red", "sienna", "palevioletred1", "royalblue2"), names = c("base", "poll1000", "poll100", "consistent"), outline=FALSE)
grid(col = "gray")
dev.off()


png('throughput.png', width = 5, height=5, units="in", res=300)
boxplot(throughput_base, throughput_poll1000, throughput_poll100, throughput_cons, ylab="throughput (msg/second)", main="Latency Benchmark", col = c("red", "sienna", "palevioletred1", "royalblue2"), names = c("base", "poll1000", "poll100", "consistent"), outline=FALSE)
grid(col = "gray")
dev.off()
