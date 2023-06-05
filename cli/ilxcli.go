// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/rpc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	authenticationTokenKey = "AuthenticationToken"
	defaultConfigFilename  = "ilxcli.conf"
)

type options struct {
	ShowVersion bool   `short:"v" long:"version" description:"Display version information and exit"`
	ConfigFile  string `short:"C" long:"configfile" description:"Path to configuration file"`
	AuthToken   string `short:"t" long:"authtoken" description:"The ilxd node gRPC authentican token if needed"`
	ServerAddr  string `short:"a" long:"serveraddr" description:"The address of the ilxd gRPC server (in multiaddr format)" default:"/ip4/127.0.0.1/tcp/5001"`
	RPCCert     string `long:"rpccert" description:"A path to the SSL certificate to use with gRPC (this is only need if using a self-signed cert)" default:"~/.ilxd/rpc.cert"`
}

func main() {

	var configFile string
	for i, arg := range os.Args {
		if strings.HasPrefix(arg, "--configfile=") {
			configFile = strings.Split(arg, "--configfile=")[1]
		} else if arg == "-C" && len(os.Args) > i+1 {
			configFile = os.Args[i+1]
		}
	}
	if configFile == "" {
		configFile = filepath.Join(repo.DefaultHomeDir, defaultConfigFilename)
	}

	var opts options
	parser := flags.NewParser(&opts, flags.Default)
	err := flags.NewIniParser(parser).ParseFile(configFile)
	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			fmt.Fprintf(os.Stderr, "Error parsing config "+
				"file: %v\n", err)
			usageMessage := "Use ilxcli -h to show usage"
			fmt.Fprintln(os.Stderr, usageMessage)
			log.Fatal(err)
		}
	}
	if len(os.Args) == 2 && os.Args[1] == "-v" {
		fmt.Println(repo.VersionString())
		return
	}

	parser = flags.NewNamedParser("ilxcli", flags.HelpFlag)
	parser.AddGroup("Connection options", "Configuration options for connecting to ilxd", &opts)

	// Blockchain service
	parser.AddCommand("getmempoolinfo", "Returns the state of the current mempool", "Returns the state of the current mempool", &GetMempoolInfo{&opts})
	parser.AddCommand("getmempool", "Returns all the transactions in the mempool", "Returns all the transactions in the mempool", &GetMempool{&opts})
	parser.AddCommand("getblockchaininfo", "Returns data about the blockchain", "Returns data about the blockchain including the most recent block hash and height", &GetBlockchainInfo{&opts})
	parser.AddCommand("getblockinfo", "Returns a block header plus some extra metadata", "Returns a block header plus some extra metadata", &GetBlockInfo{opts: &opts})
	parser.AddCommand("getblock", "Returns the detailed data for a block", "Returns the detailed data for a block", &GetBlock{opts: &opts})
	parser.AddCommand("getcompressedblock", "Returns a block in compressed format", "Returns a block that is stripped down to just the outputs. It is the bare minimum information a client side wallet needs to compute its internal state.", &GetCompressedBlock{opts: &opts})
	parser.AddCommand("gettransaction", "Returns the transaction for the given transaction ID", "Returns the transaction for the given transaction ID. Requires TxIndex.", &GetTransaction{opts: &opts})
	parser.AddCommand("getmerkleproof", "Returns a Merkle (SPV) proof for a specific transaction in the provided block", "Returns a Merkle (SPV) proof for a specific transaction in the provided block. Requires TxIndex.", &GetMerkleProof{opts: &opts})
	parser.AddCommand("getvalidator", "Returns all the information about the given validator", "Returns all the information about the given validator including the number of staked coins.", &GetValidator{opts: &opts})
	parser.AddCommand("getvalidatorsetinfo", "Returns information about the validator set", "Returns information about the validator set.", &GetValidatorSetInfo{opts: &opts})
	parser.AddCommand("getvalidatorset", "Returns all the validators in the current validator set", "Returns all the validators in the current validator set.", &GetValidatorSet{opts: &opts})
	parser.AddCommand("getaccumulatorcheckpoint", "Returns the accumulator at the requested height", "Returns the accumulator at the requested height. If there is no checkpoint at that height, the *prior* checkpoint found in the chain will be returned. If there is no prior checkpoint (as is prior to the first), an error will be returned.", &GetAccumulatorCheckpoint{opts: &opts})
	parser.AddCommand("submittransaction", "Validates a transaction and submits it to the network", "Validates a transaction and submits it to the network. An error will be returned if it fails validation.", &SubmitTransaction{opts: &opts})

	// Node service
	parser.AddCommand("gethostinfo", "Returns info about the libp2p host", "Returns info about the libp2p host", &GetHostInfo{opts: &opts})
	parser.AddCommand("getpeers", "Returns a list of peers that this node is connected to", "Returns a list of peers that this node is connected to", &GetPeers{opts: &opts})
	parser.AddCommand("addpeer", "Attempts to connect to the provided peer", "Attempts to connect to the provided peer", &AddPeer{opts: &opts})
	parser.AddCommand("blockpeer", "Blocks the given peer for the provided time period", "Blocks the given peer for the provided time period", &BlockPeer{opts: &opts})
	parser.AddCommand("unblockpeer", "Removes a peer from the block list", "Removes a peer from the block list", &UnblockPeer{opts: &opts})
	parser.AddCommand("setloglevel", "Changes the logging level of the node", "Changes the logging level of the node", &SetLogLevel{opts: &opts})
	parser.AddCommand("getminfeeperkilobyte", "Returns the node's current minimum transaction fee", "Returns the node's current minimum transaction fee needed to relay transactions and admit them into the mempool. Validators will also set their initial preference for blocks containing transactions with fees below this threshold to not-preferred.", &GetMinFeePerKilobyte{opts: &opts})
	parser.AddCommand("setminfeeperkilobyte", "Sets the node's fee policy", "Sets the node's fee policy", &SetMinFeePerKilobyte{opts: &opts})
	parser.AddCommand("getminstake", "Returns the node's current minimum stake policy", "Returns the node's current minimum stake policy. Stake transactions staking less than this amount will not be admitted into the mempool and will not be relayed. Validators will also set their initial preference for blocks containing stake transactions below this threshold to not-preferred.", &GetMinStake{opts: &opts})
	parser.AddCommand("setminstake", "Sets the node's minimum stake policy", "Sets the node's minimum stake policy", &SetMinStake{opts: &opts})
	parser.AddCommand("getblocksizesoftlimit", "Returns the node's current blocksize soft limit", "Returns the node's current blocksize soft limit. Validators will also set their initial preference for blocks over this size to not-preferred.", &GetBlockSizeSoftLimit{opts: &opts})
	parser.AddCommand("setblocksizesoftlimit", "Sets the node's blocksize soft limit policy", "Sets the node's blocksize soft limit policy.", &SetBlockSizeSoftLimit{opts: &opts})
	parser.AddCommand("gettreasurywhitelist", "Returns the current treasury whitelist for the node", "Returns the current treasury whitelist for the node. Blocks containing TreasuryTransactions not found in this list will have their initial preference set to not-preferred.", &GetTreasuryWhitelist{opts: &opts})
	parser.AddCommand("updatetreasurywhitelist", "Adds or removes a transaction from the treasury whitelist", "Adds or removes a transaction from the treasury whitelist", &UpdateTreasuryWhitelist{opts: &opts})
	parser.AddCommand("reconsiderblock", "Tries to reprocess the given block", "Tries to reprocess the given block", &ReconsiderBlock{opts: &opts})
	parser.AddCommand("recomputechainstate", "Rebuilds the entire chain state from genesis", "Deletes the accumulator, validator set, and nullifier set and rebuilds them by loading and re-processing all blocks from genesis.", &RecomputeChainState{opts: &opts})

	if _, err := parser.Parse(); err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			fmt.Println(err)
			os.Exit(0)
		}
		log.Fatal(err)
	}

}

func makeContext(authToken string) context.Context {
	ctx := context.Background()
	if authToken != "" {
		md := metadata.Pairs(authenticationTokenKey, authToken)
		ctx = metadata.NewOutgoingContext(context.Background(), md)
	}
	return ctx
}

func makeBlockchainClient(opts *options) (pb.BlockchainServiceClient, error) {
	certFile := repo.CleanAndExpandPath(opts.RPCCert)

	creds, err := credentials.NewClientTLSFromFile(certFile, "localhost")
	if err != nil {
		return nil, err
	}
	ma, err := multiaddr.NewMultiaddr(opts.ServerAddr)
	if err != nil {
		return nil, err
	}

	netAddr, err := manet.ToNetAddr(ma)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(netAddr.String(), grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}
	return pb.NewBlockchainServiceClient(conn), nil
}

func makeNodeClient(opts *options) (pb.NodeServiceClient, error) {
	certFile := repo.CleanAndExpandPath(opts.RPCCert)

	creds, err := credentials.NewClientTLSFromFile(certFile, "localhost")
	if err != nil {
		return nil, err
	}
	ma, err := multiaddr.NewMultiaddr(opts.ServerAddr)
	if err != nil {
		return nil, err
	}

	netAddr, err := manet.ToNetAddr(ma)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(netAddr.String(), grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}
	return pb.NewNodeServiceClient(conn), nil
}
