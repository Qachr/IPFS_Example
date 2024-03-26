package IpfsLink

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"

	//formatIPFS "github.com/ipfs/go-ipld-format"
	icore "github.com/ipfs/interface-go-ipfs-core"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/bootstrap"
	"github.com/ipfs/kubo/core/coreapi"
	libp2pIFPS "github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// DiscoveryInterval is how often we re-publish our mDNS records.
const DiscoveryInterval = time.Hour

// DiscoveryServiceTag is used in our mDNS advertisements to discover other chat peers.
const DiscoveryServiceTag = "pubsub-chat-example"

// printErr is like fmt.Printf, but writes to stderr.
func printErr(m string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, m, args...)
}

// defaultNick generates a nickname based on the $USER environment variable and
// the last 8 chars of a peer ID.
func defaultNick(p peer.ID) string {
	return fmt.Sprintf("%s-%s", os.Getenv("USER"), shortID(p))
}

// shortID returns the last 8 chars of a base58-encoded peer id.
func shortID(p peer.ID) string {
	pretty := p.Pretty()
	return pretty[len(pretty)-8:]
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Printf("discovered new peer %s\n", pi.Addrs[0])
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", pi.ID.Pretty(), err)
	}
}

// setupDiscovery creates an mDNS discovery service and attaches it to the libp2p Host.
// This lets us automatically discover peers on the same LAN and connect to them.
func setupDiscovery(h host.Host) error {
	// setup mDNS discovery to find local peers
	s := mdns.NewMdnsService(h, DiscoveryServiceTag, &discoveryNotifee{h: h})
	return s.Start()
}

type IpfsLink struct {
	Ctx      context.Context
	IpfsCore icore.CoreAPI
	IpfsNode *core.IpfsNode
}

func InitNode(ipfsBootstrap []byte, swarmKey bool) (*IpfsLink, error) {
	ct, _ := context.WithCancel(context.Background())

	// Spawn a local peer using a temporary path, for testing purposes
	var idBootstrap peer.AddrInfo
	var ipfsA icore.CoreAPI
	var nodeA *core.IpfsNode
	var err error

	// if we got a bootstrap done, or not
	if len(ipfsBootstrap) > 0 {

		e := idBootstrap.UnmarshalJSON(ipfsBootstrap)
		if e != nil {
			panic(fmt.Errorf("couldn't Unmarshal bootstrap peer addr info, error : %s", e))
		}
		ipfsA, nodeA, err = spawnEphemeral(ct, &idBootstrap, swarmKey)
	} else {
		// if we got no bootstrap, we are the bootstrap
		ipfsA, nodeA, err = spawnEphemeral(ct, nil, swarmKey)
	}

	if err != nil {
		panic(fmt.Errorf("failed to spawn peer node: %s", err))
	}

	ipfs := IpfsLink{
		Ctx:      ct,
		IpfsCore: ipfsA,
		IpfsNode: nodeA,
	}

	//fmt.Println(ipfs.IpfsNode.Peerstore.PeerInfo(ipfs.IpfsNode.PeerHost.ID()))
	return &ipfs, err
}

var loadPluginsOnce sync.Once

var flagExp = flag.Bool("experimental", false, "enable experimental features")

func createTempRepo(btstrap []peer.AddrInfo) (string, error) {
	repoPath, err := os.MkdirTemp("", "ipfs-shell")
	if err != nil {
		return "", fmt.Errorf("failed to get temp dir: %s", err)
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(io.Discard, 2048)
	if err != nil {
		return "", err
	}

	cfg.SetBootstrapPeers(btstrap)

	// Create the repo with the config
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to init ephemeral node: %s", err)
	}

	return repoPath, nil
}

/// ------ Spawning the node

// Creates an IPFS node and returns its coreAPI
func createNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2pIFPS.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, err
	}
	return node, nil

}

func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

// Spawns a node to be used just for this run (i.e. creates a tmp repo)
func spawnEphemeral(ctx context.Context, btstrap *peer.AddrInfo, swarmKey bool) (icore.CoreAPI, *core.IpfsNode, error) {
	var onceErr error
	loadPluginsOnce.Do(func() {
		onceErr = setupPlugins("")
	})
	if onceErr != nil {
		return nil, nil, onceErr
	}

	// Create a Temporary Repo
	var m []peer.AddrInfo

	if btstrap != nil {
		m = make([]peer.AddrInfo, 1)
		m[0] = *btstrap
	} else {
		m = make([]peer.AddrInfo, 0)
	}

	repoPath, err := createTempRepo(m)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp repo: %s", err)
	}

	// Create an IPFS node
	printErr("repository : %s\n", repoPath)
	if swarmKey {
		os.WriteFile(repoPath+"/swarm.key", []byte("/key/swarm/psk/1.0.0/\n/base16/\nedd99a84bbdd5c9cfc06bcc039d219b1000885ecba26901c02e7c8792bfaaa70"), fs.FileMode(os.O_CREATE|os.O_WRONLY|os.O_APPEND))
	}

	node, err := createNode(ctx, repoPath)
	if err != nil {
		return nil, nil, err
	}
	if swarmKey {
		node.PNetFingerprint = []byte("4c7dc2a2735a84b4b11ff5b39225aa771cea1abd3acf9b98708a25f286df851c")
	}
	// Connect the node to the other private network nodes

	var bstcfg bootstrap.BootstrapConfig
	if btstrap != nil {

		bstcfg = bootstrap.BootstrapConfig{

			MinPeerThreshold:  1,
			Period:            60 * time.Second,
			ConnectionTimeout: 30 * time.Second,
			BootstrapPeers: func() []peer.AddrInfo {
				m := make([]peer.AddrInfo, 1)
				m[0] = *btstrap
				return m
			},
		}
	} else {
		bstcfg = bootstrap.BootstrapConfig{
			MinPeerThreshold:  0,
			Period:            60 * time.Second,
			ConnectionTimeout: 30 * time.Second,
			BootstrapPeers: func() []peer.AddrInfo {
				m := make([]peer.AddrInfo, 0)
				// m[0] = node.Peerstore.PeerInfo(node.Identity)
				return m
			},
		}
	}

	node.Bootstrap(bstcfg)

	api, err := coreapi.NewCoreAPI(node)

	if btstrap != nil {
		api.Swarm().Connect(ctx, *btstrap)
	}

	return api, node, err
}

func AddIPFS(ipfs *IpfsLink, message []byte) (icorepath.Resolved, error) {

	peerCidFile, err := ipfs.IpfsCore.Unixfs().Add(ipfs.Ctx,
		files.NewBytesFile(message))
	if err != nil {
		panic(fmt.Errorf("could not add File: %s", err))
	}
	go ipfs.IpfsCore.Dht().Provide(ipfs.Ctx, peerCidFile)
	// if err != nil {
	// 	panic(fmt.Errorf("Could not provide File - %s", err))
	// }
	return peerCidFile, err
}

type CID struct{ str string }

func GetIPFS(ipfs *IpfsLink, cid_cid cid.Cid) (files.Node, error) {
	// str_CID, err := ContentIdentifier.Decode(c)

	cctx, _ := context.WithDeadline(ipfs.Ctx, time.Now().Add(time.Second*30))
	f, err2 := ipfs.IpfsCore.Unixfs().Get(cctx, icorepath.IpfsPath(cid_cid))
	if err2 != nil {
		printErr("could not get file with CID - %s : %s\n", cid_cid, err2)
		printErr("what we got from IPFSNODE.dag  error :  %s\n  data : %s\n====================================\n", err2, f)
	}
	// Writes data downloadedto a file
	files.WriteTo(f, cid_cid.String())
	fmt.Printf("Wrote file to %s\n", cid_cid.String())

	return f, err2
}
