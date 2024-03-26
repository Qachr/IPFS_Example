#!/bin/bash

sysctl -w net.ipv6.conf.all.disable_ipv6=1 && sysctl -w net.ipv6.conf.default.disable_ipv6=1
LIBP2P_FORCE_PNET=1 

./IPFS_Test --mode=peer --SwarmKey=true --IPFSBootstrap=IDBootstrapIPFS 