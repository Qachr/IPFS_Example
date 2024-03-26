#!/bin/bash

go build

./IPFS_Test --mode=bootstrap

# Then Retrieve IDBootstrapIPFS and CID.txt, to send it to others peers