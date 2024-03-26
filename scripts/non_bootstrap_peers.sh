#!/bin/bash

go build 

# needed to have the file IDBootstrapIPFS 

./IPFS_Test --mode=peer --IPFSBootstrap=IDBootstrapIPFS 