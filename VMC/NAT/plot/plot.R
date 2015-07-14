# run as Rscript plot.R native/plotdata.txt coloc/plotdata.txt local/plotdata.txt lan/plotdata.txt plot.png
# read CLI parameters
args <- commandArgs(trailingOnly = TRUE)

native <- args[1] 
coloc <- args[2]
local <- args[3]
lan <- args[4]

outFile <- args[5]
#sprintf("args = %s", args)

plotTitle <- "NAT capacity vs. latency"

# define the output file and plot it
png(outFile, width = 6, height=5, units="in", res=300)
	
# plot(0, 0, type="p", xlab="NAT capacity", ylab="Packet latency (ms)", 
# 	xlim=c(0,10000), ylim=c(0,100))

linetype <- c(1:4)
plotchar <- c(1:4)
 
colors <- c("blue", "red", "purple", "black")
titles <- c("native", "co-etcd",  "local-etcd", "lan-etcd")

# read data for native and plot the line
inp <- read.table(native, header=TRUE)
natCapacity <- inp[[1]]
latency <- inp[[2]]
# sprintf("%s", natCapacity)
# sprintf("%s", latency)
plot(natCapacity, latency, type="l", col=colors[1], lty=linetype[1], xlab="NAT capacity", ylab="Packet latency (ms)", xlim=c(0,10000), ylim=c(0,100))

# read data for coloc and plot the line
inp <- read.table(coloc, header=TRUE)
natCapacity <- inp[[1]]
latency <- inp[[2]]
lines(natCapacity, latency, type="l", lty=linetype[2], col=colors[2], pch=plotchar[2])

# read data for local and plot the line
inp <- read.table(local, header=TRUE)
natCapacity <- inp[[1]]
latency <- inp[[2]]
lines(natCapacity, latency, type="l", lty=linetype[3], col=colors[3], pch=plotchar[3])

# read data for local and plot the line
inp <- read.table(lan, header=TRUE)
natCapacity <- inp[[1]]
latency <- inp[[2]]
lines(natCapacity, latency, type="l", lty=linetype[4], col=colors[4], pch=plotchar[4])

# put the plot title, legend and finalize plotting
title(main=plotTitle)
legend("topleft", titles, col=colors, lty=linetype)
# grid(col="gray")
dev.off()
