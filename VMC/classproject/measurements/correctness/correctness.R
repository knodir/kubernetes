inp <- scan("dataset1.txt", list(0,0,0,0))
send_rate_1<-inp[[3]]; receive_rate_1<-inp[[4]]

inp <- scan("dataset2.txt", list(0,0,0,0))
send_rate_2<-inp[[3]]; receive_rate_2<-inp[[4]]

inp <- scan("dataset4.txt", list(0,0,0,0))
send_rate_4<-inp[[3]]; receive_rate_4<-inp[[4]]

png('correctness.png', width = 6, height=5, units="in", res=300)

plot(send_rate_1, receive_rate_1, type="o", col="blue", xlab="Send Rate (echo msgs/sec)", ylab="Receive Rate (echo msgs/sec)", xlim=c(0,200))
lines(send_rate_2, receive_rate_2, type="o", col="red")
lines(send_rate_4, receive_rate_4, type="o", col="green")
title(main="Middlebox correctness with shared etcd state")
legend(100,50,c("1 middlebox", "2 middleboxes", "4 middleboxes"), col=c("blue", "red", "green"), lty=1:1)

grid(col = "gray")
dev.off()

