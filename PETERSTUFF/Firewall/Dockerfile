FROM quantumlight/ubuntu-pcap-netfilter-ssh-iptable:latest 

MAINTAINER Peter Chen

ADD golang_firewall /var/golang_firewall/golang_firewall

EXPOSE 3333

CMD ["./var/golang_firewall/golang_firewall"]
