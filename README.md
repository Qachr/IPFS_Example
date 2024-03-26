# IPFS_Example

## Goal of this repository
The goal of this repository is to show how to create IPFS nodes and connect them so you can share files among a private cluster of peers.
Note : This oesn't use IPFS Cluster, this is just file sharing with no management of replication.

## What's inside

- main.go

    Main program to run inside the machines that will host the IPFS Nodes. This uses IPFSlink/Ipfslink/go
    Host must have go 1.20 installed

- scipts 

    example of what should look like the cmds to be run in the hosts, either bootstrap or not.
    Notice that two files have to be shared from the bootstrap to the other peers : "CID.txt" and "IDBootstrapIPFS"