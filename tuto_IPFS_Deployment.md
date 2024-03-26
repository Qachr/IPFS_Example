I reserve multiple nodes in Grid5000.

each nodes must have go installed,
One of the node will be the bootstrap peer for both ipfs connection and pubsub connexion.
The others peers will be simple node connecting to this bootstrap node so they can be aware of other peers.


# If you want to have a Private Network

## On the bootstrap node :

- Set up the libp2p private network flag, this will enforce you to have a key for swarming peers.
bash :
    > export LIBP2P_FORCE_PNET=1 # NOT For Global IPFS Network

Then In your go deployment you have several steps to define your IPFS Node:
### define configuration folder
- First define the wanted Configuration  :
    > cfg, err := config.Init(io.Discard, 2048)

    With config from "github.com/ipfs/kubo/config"
- Set up the bootstrap peer to none, as we are the bootstrap peer.
    > btstrap := make([]peer.AddrInfo, 0) # NOT For Global IPFS Network
	> cfg.SetBootstrapPeers(btstrap) # NOT For Global IPFS Network

- Create the config defined earlier in a folder "repoPath",  chosen earlier (here its generated with a random name)
	>fsrepo.Init(repoPath, cfg)

    with fsrepo from "github.com/ipfs/kubo/repo/fsrepo"
- Add a swarm key because this wouldn't start otherwise because of LIBP2P_FORCE_PNET, here the key is hardcoded, because it is just testing
    > key = "/key/swarm/psk/1.0.0/\n/base16/\nedd99a84bbdd5c9cfc06bcc039d219b1000885ecba26901c02e7c8792bfaaa70" # NOT For Global IPFS Network

    > os.WriteFile(repoPath+"/swarm.key", []byte(key), fs.FileMode(os.O_CREATE|os.O_WRONLY|os.O_APPEND)) # NOT For Global IPFS Network

### Start IPFS node
- Create the IPFS Node, folowing the config that has been set up in the config folder
    > node, err := createNode(ctx, repoPath)
- Precise a bootstrap Config so we can run a bootstrap connexion now : # NOT For Global IPFS Network, instead to addBootStrapPeers
    >     bstcfg = bootstrap.BootstrapConfig{
	>		MinPeerThreshold:  0,
	>		Period:            60 * time.Second,
	>		ConnectionTimeout: 30 * time.Second,
	>		BootstrapPeers: func() []peer.AddrInfo {
	>			m := make([]peer.AddrInfo, 0)
	>			return m
	>		},
    >     }
    >     node.Bootstrap(bstcfg)
- Start the core API, that will manage files sharing in IPFS
	> ipfsCore, err := coreapi.NewCoreAPI(node)
if btstrap != nil {
		api.Swarm().Connect(ctx, *btstrap)
	}

### Create a file containing your node address
- The information has to be a peer.AddrInfo, the best way to get our own info was to ask our swarm peer management what is our own peer.AddrInfo
    > bootstrapPeerAddrInfo, err := node.Peerstore.PeerInfo(node.Identity)
	
- Then I marshall and write this into a file
	> WriteFile("./IDBootstrapIPFS", bootstrapPeerAddrInfo.Marshall())

- Finally send this file to every other nodes : Depending on how you manage you nodes, for me its
    > scp root@BOOTSTRAPNODE:~/CRDT_IPFS/IDBootstrapIPFS IDBootstrapIPFS
    > scp IDBootstrapIPFS root@OTHERNODE:~/CRDT_IPFS/IDBootstrapIPFS

## On the other nodes 

- Set up the libp2p private network flag, this will enforce you to have a key for swarming peers.
bash :
    > export LIBP2P_FORCE_PNET=1 # NOT For Global IPFS Network

Then In your go deployment you have several steps to define your IPFS Node:
### define configuration folder
- First define the wanted Configuration  :
    > cfg, err := config.Init(io.Discard, 2048)

    With config from "github.com/ipfs/kubo/config"
- Set up the bootstrap peer to the bootstrap PeerAddrInfo this node has received
    >       var idBootstrap peer.AddrInfo
	>       idBootstrap.UnmarshalJSON(ipfsBootstrap)
    >       btstrap := make([]peer.AddrInfo, 1)
    >       btstrap[1] = idBootstrap
	>       cfg.SetBootstrapPeers(btstrap) # NOT For Global IPFS Network, instead to addBootStrapPeers

- Create the config defined earlier in a folder "repoPath",  chosen earlier (here its generated with a random name)
	>fsrepo.Init(repoPath, cfg)

    with fsrepo from "github.com/ipfs/kubo/repo/fsrepo"
- Add a swarm key because this wouldn't start otherwise because of LIBP2P_FORCE_PNET, here the key is hardcoded, because it is just testing
    > key = "/key/swarm/psk/1.0.0/\n/base16/\nedd99a84bbdd5c9cfc06bcc039d219b1000885ecba26901c02e7c8792bfaaa70" # NOT For Global IPFS Network

    > os.WriteFile(repoPath+"/swarm.key", []byte(key), fs.FileMode(os.O_CREATE|os.O_WRONLY|os.O_APPEND)) # NOT For Global IPFS Network

### Start IPFS node
- Create the IPFS Node, folowing the config that has been set up in the config folder
    > node, err := createNode(ctx, repoPath)
- Precise a bootstrap Config so we can run a bootstrap connexion now : # NOT For Global IPFS Network, instead to addBootStrapPeers
    >     bstcfg = bootstrap.BootstrapConfig{
	>		MinPeerThreshold:  0,
	>		Period:            60 * time.Second,
	>		ConnectionTimeout: 30 * time.Second,
	>		BootstrapPeers: func() []peer.AddrInfo {
    >           btstrap := make([]peer.AddrInfo, 1)
    >           btstrap[1] = idBootstrap
	>			return btstrap
	>		},
    >     }
    >     node.Bootstrap(bstcfg)
- Start the core API, that will manage files sharing in IPFS
	> api, err := coreapi.NewCoreAPI(node)
## How to use it :
### Get a file from CID
- One function returns you directly the files.Node
    > filesDl, err = ipfsCore.Unixfs().Get(cctx, icorepath.IpfsPath(str_CID))
- then you can write this files.Node to a file 
	> filesDl.WriteTo(fil, "fileRetreivedFromIPFS.txt")
### Provide a file	
- First add the file (here message is the file content) to the file system, so we have the CID
    > peerCidFile, err := IpfsCore.Unixfs().Add(Ctx,files.NewBytesFile(message))
- Then add the CID to your DHT 
	> ipfs.IpfsCore.Dht().Provide(ipfs.Ctx, peerCidFile)

# If you want to connect to global IPFS

The Only differences when you don't want a private network but you want to be connected to the global network are :
- No need to remove all the defaults bootstrap Peers from all nodes, just add the one you are considering as a bootstrap
- DO NOT configure the LIBP2P_FORCE_PNET=1, and do not set any swarm.key on any nodes
- when making a bootstrap round, raise the number of minimum peers to 4/5



To check later :
// https://github.com/ipfs/kubo/blob/21728eb0002ae7f79b52af7a48142330b3da81a0/core/node/libp2p/pnet.go#L37  

Also : // https://github.com/ipfs/kubo/blob/master/docs/experimental-features.md#private-networks $\rightarrow$ Private network experimental feature