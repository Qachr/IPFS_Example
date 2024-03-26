package main

import (
	"IPFS_Test/IpfsLink"
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/ipfs/go-cid"
)

func main() {
	fmt.Printf("Hello")

	mode := flag.String("mode", "", "mode of the current application")
	IPFSbootstrap := flag.String("IPFSBootstrap", "", "IPFS bootstrap peer to have a private network")
	swarmKey := flag.Bool("SwarmKey", false, "IPFS bootstrap peer to have a private network")

	flag.Parse()

	fmt.Printf("Mode : %s\nBootstrapFile: %s\nSwarmKey: %d\n", *mode, *IPFSbootstrap, *swarmKey)

	if *mode == "bootstrap" {
		bootstrap_main(*swarmKey)
	} else {
		non_bs_main(*IPFSbootstrap, *swarmKey)
	}
	fmt.Printf("Press Enter to finish demo\n")

	input := bufio.NewScanner(os.Stdin)
	input.Scan()

}

func bootstrap_main(swarmKey bool) {
	sys1, err := IpfsLink.InitNode(make([]byte, 0), swarmKey)
	if err != nil {
		panic("Could not instantiate bootstrap :")
	}

	bytesIPFS_Node, err := sys1.IpfsNode.Peerstore.PeerInfo(sys1.IpfsNode.Identity).MarshalJSON()
	if err != nil {
		fmt.Printf("Failed To Marshall IFPS Identity & LibP2P clients : %s", err)
	}
	if _, err := os.Stat("./IDBootstrapIPFS"); !errors.Is(err, os.ErrNotExist) {
		os.Remove("./IDBootstrapIPFS")
	}
	WriteFile("./IDBootstrapIPFS", bytesIPFS_Node)

	Path, _ := IpfsLink.AddIPFS(sys1, []byte("This is the Data I push into IPFS"))

	b, err := json.Marshal(Path.Cid())

	if err != nil {
		fmt.Printf("error, could not marshal cid, the error : %s", err)
	}

	WriteFile("CID.txt", b)

}

func non_bs_main(IPFSBootstrap string, swarmKey bool) {

	IPFSbootstrapBytes, err := os.ReadFile(IPFSBootstrap)
	if err != nil {
		fmt.Printf("Failed To Read IFPS bootstrap peer multiaddr : %s", err)
	}
	sys1, _ := IpfsLink.InitNode(IPFSbootstrapBytes, swarmKey)

	cidByte, _ := os.ReadFile("CID.txt")
	var cid_cid cid.Cid
	err = json.Unmarshal(cidByte, &cid_cid)
	if err != nil {
		fmt.Printf("Failed To Unmarshall IFPS CID : \nerror :  %s", err)
	}
	fmt.Printf("Asking the CID : %s \n", cidByte)

	IpfsLink.GetIPFS(sys1, cid_cid)

}

func WriteFile(fileName string, b []byte) {
	fil, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Errorf("WRITEFILE - could Not Open RootNode to update rootnodefolder\nerror: %s", err))
	}
	_, err = fil.Write(b)
	if err != nil {
		panic(fmt.Errorf("could Not write in RootNode to WRITEFILE - \nerror: %s", err))
	}
	err = fil.Close()
	if err != nil {
		panic(fmt.Errorf("could Not Close - WRITEFILE\nerror: %s", err))
	}
}
