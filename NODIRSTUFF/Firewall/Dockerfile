FROM quantumlight/ubuntu-pcap-netfilter-ssh-iptable:latest 

MAINTAINER Peter Chen

ADD golang_firewall_poll1000 /var/golang_firewall/golang_firewall_poll1000

EXPOSE 3333

CMD ["./var/golang_firewall/golang_firewall_poll1000"]
