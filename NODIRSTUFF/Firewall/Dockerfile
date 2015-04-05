FROM quantumlight/ubuntu-pcap-netfilter-ssh-iptable:latest 

MAINTAINER Peter Chen, Nodir Kodirov

ADD golang_firewall /var/golang_firewall/golang_firewall_v3

EXPOSE 3333

CMD ["./var/golang_firewall/golang_firewall"]
