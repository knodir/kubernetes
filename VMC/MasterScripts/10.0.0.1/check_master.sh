#!/bin/bash

# Check etcd status
sudo systemctl status etcd_custom.service

# Check flannel status
sudo systemctl status flannel.service

# Check docker status
sudo systemctl status docker_custom.service

# Check apiserver status
sudo systemctl status apiserver.service

# Check controller manager status
sudo systemctl status controller-manager.service

# Check scheduler status
sudo systemctl status scheduler.service

# Check kube-proxy status
sudo systemctl status proxy.service

# Check kubelet status
sudo systemctl status kubelet.service 
