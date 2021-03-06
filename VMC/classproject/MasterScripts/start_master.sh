#!/bin/bash

# Start etcd
echo -n "DISCOVERY_URL=" > ~/STARTUP_SCRIPTS/discovery_token.txt
echo -n $(curl https://discovery.etcd.io/new?size=3) >> ~/STARTUP_SCRIPTS/discovery_token.txt
sudo systemctl start etcd_custom.service

# Start flannel
sudo systemctl start flannel.service

# Start docker
sudo systemctl start docker_custom.service

# Start apiserver
sudo systemctl start apiserver.service

# Start controller manager
sudo systemctl start controller-manager.service

# Start scheduler 
sudo systemctl start scheduler.service

# Start kube-proxy
sudo systemctl start proxy.service

# Start kubelet
sudo systemctl start kubelet.service 
