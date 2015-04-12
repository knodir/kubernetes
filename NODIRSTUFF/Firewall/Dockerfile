FROM quantumlight/ubuntu-pcap-netfilter-ssh-iptable:latest 

MAINTAINER Peter Chen

ADD golang_firewall_consistent /var/golang_firewall/golang_firewall_consistent

EXPOSE 3333

CMD ["./var/golang_firewall/golang_firewall_consistent"]
