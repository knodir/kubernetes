inp <- scan("load_1.txt", list("",0,0))
fw1time<-inp[[2]]; fw1rampct<-inp[[3]]

inp <- scan("load_2.txt", list("",0,0))
fw2time<-inp[[2]]; fw2rampct<-inp[[3]]

inp <- scan("load_3.txt", list("",0,0))
fw3time<-inp[[2]]; fw3rampct<-inp[[3]]

inp <- scan("load_4.txt", list("",0,0))
fw4time<-inp[[2]]; fw4rampct<-inp[[3]]

first = fw1time[1]

fw1time = (fw1time - first)/1000000000
fw2time = (fw2time - first)/1000000000
fw3time = (fw3time - first)/1000000000
fw4time = (fw4time - first)/1000000000

png('scaling.png', width = 6, height=5, units="in", res=300)

plot(fw1time, fw1rampct, type="o", col="blue", xlab="Time (s)", ylab="RAM Utilization %", ylim=c(45,100), xlim=c(0,152))
lines(fw2time, fw2rampct, type="o", col="red")
lines(fw3time, fw3rampct, type="o", col="green")
lines(fw4time, fw4rampct, type="o", col="black")

title(main="Dynamic scaling under load")
legend(0,100, c("middlebox 1", "middlebox 2",  "middlebox 3", "middlebox 4"), col=c("blue","red","green", "black"), lty=1:1)

grid(col="gray")
dev.off()
