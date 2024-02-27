// Copyright (c) 2024 The illium developers
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
	parser.AddCommand("getnetworkkey", "Returns node's network private key", "Returns node's network private key", &GetNetworkKey{opts: &opts})
	parser.AddCommand("getpeers", "Returns a list of peers that this node is connected to", "Returns a list of peers that this node is connected to", &GetPeers{opts: &opts})
	parser.AddCommand("getpeerinfo", "Returns info about the peer", "Returns info about the peer if it is connected", &GetPeerInfo{opts: &opts})
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
	parser.AddCommand("updatetreasurywhitelist", "Adds or removes a transaction from the treasury whitelist", "Adds or removes a transaction from the treasury whitelist. This change is committed to the datastore and will persist between sessions.", &UpdateTreasuryWhitelist{opts: &opts})
	parser.AddCommand("reconsiderblock", "Tries to reprocess the given block", "Tries to reprocess the given block", &ReconsiderBlock{opts: &opts})
	parser.AddCommand("recomputechainstate", "Rebuilds the entire chain state from genesis", "Deletes the accumulator, validator set, and nullifier set and rebuilds them by loading and re-processing all blocks from genesis.", &RecomputeChainState{opts: &opts})
	parser.AddCommand("signmessage", "Sign a message with the network key", "Sign a message with the nework key", &SignMessage{opts: &opts})
	parser.AddCommand("verifymessage", "Verify a signed message", "Verify a signed message", &VerifyMessage{opts: &opts})

	// Wallet service
	parser.AddCommand("getbalance", "Returns the combined balance of all addresses in the wallet", "Returns the combined balance of all addresses in the wallet", &GetBalance{opts: &opts})
	parser.AddCommand("getwalletseed", "Returns the mnemonic seed for the wallet", "Returns the mnemonic seed for the wallet. If the wallet seed has been deleted, an error will be returned.", &GetWalletSeed{opts: &opts})
	parser.AddCommand("getaddress", "Returns the most recent address of the wallet", "Returns the most recent address of the wallet", &GetAddress{opts: &opts})
	parser.AddCommand("gettimelockedaddress", "Returns an address which locks coins until the provided timestamp", "Returns a timelocked address based on the wallet's most recent private key. Coins sent to this address will be locked until the provided timestamp.", &GetTimelockedAddress{opts: &opts})
	parser.AddCommand("getpublicaddress", "Returns the most recent public address of the wallet", "Returns a public address built from the wallet's most recent private key.", &GetPublicAddress{opts: &opts})
	parser.AddCommand("getaddresses", "Returns all the addresses created by this wallet", "Returns all the addresses created by this wallet", &GetAddresses{opts: &opts})
	parser.AddCommand("getaddrinfo", "Returns info about the given address", "Returns info about the given address", &GetAddrInfo{opts: &opts})
	parser.AddCommand("getnewaddress", "Generates a new address and returns it", "Generates a new address and returns it. Both a new spend key and view key will be derived from the mnemonic seed.", &GetNewAddress{opts: &opts})
	parser.AddCommand("gettransactions", "Returns the list of transactions for the wallet", "Returns the list of transactions for the wallet", &GetTransactions{opts: &opts})
	parser.AddCommand("getutxos", "Returns a list of the wallet's current unspent transaction outputs (UTXOs)", "Returns a list of the wallet's current unspent transaction outputs (UTXOs)", &GetUtxos{opts: &opts})
	parser.AddCommand("getprivatekey", "Returns the serialized spend and view keys for the given address", "Returns the serialized spend and view keys for the given address", &GetPrivateKey{opts: &opts})
	parser.AddCommand("importaddress", "Imports a watch address into the wallet", "Imports a watch address into the wallet", &ImportAddress{opts: &opts})
	parser.AddCommand("createmultisigspendkeypair", "Generates a spend keypair for use in a multisig address", "Generates a spend keypair for use in a multisig address", &CreateMultisigSpendKeypair{opts: &opts})
	parser.AddCommand("createmultisigviewkeypair", "Generates a view keypair for use in a multisig address", "Generates a view keypair for use in a multisig address", &CreateMultisigViewKeypair{opts: &opts})
	parser.AddCommand("createmultisigaddress", "Generates a new multisig address using the provided public keys", "Generates a new multisig address using the provided public keys", &CreateMultisigAddress{opts: &opts})
	parser.AddCommand("createmultisignature", "Generates and returns a signature for use when proving a multisig transaction", "Generates and returns a signature for use when proving a multisig transaction", &CreateMultiSignature{opts: &opts})
	parser.AddCommand("provemultisig", "Creates a proof for a transaction with a multisig input", "Creates a proof for a transaction with a multisig input", &ProveMultisig{opts: &opts})
	parser.AddCommand("walletlock", "Encrypts the wallet's private keys", "Encrypts the wallet's private keys", &WalletLock{opts: &opts})
	parser.AddCommand("walletunlock", "Decrypts the wallet seed and holds it in memory for the specified period of time", "Decrypts the wallet seed and holds it in memory for the specified period of time", &WalletUnlock{opts: &opts})
	parser.AddCommand("setwalletpassphrase", "Encrypts the wallet for the first time", "Encrypts the wallet for the first time", &SetWalletPassphrase{opts: &opts})
	parser.AddCommand("changewalletpassphrase", "Changes the passphrase used to encrypt the wallet private keys", "Changes the passphrase used to encrypt the wallet private keys", &ChangeWalletPassphrase{opts: &opts})
	parser.AddCommand("deleteprivatekeys", "Deletes the wallet's private keys and seed from disk", "Deletes the wallet's private keys and seed from disk essentially turning the wallet into a watch-only wallet. It will still record incoming transactions but cannot spend them.", &DeletePrivateKeys{opts: &opts})
	parser.AddCommand("createrawtransaction", "Creates a new, unsigned (unproven) transaction using the given parameters", "Creates a new, unsigned (unproven) transaction using the given parameters", &CreateRawTransaction{opts: &opts})
	parser.AddCommand("createrawstaketransaction", "Creates a new, unsigned (unproven) stake transaction using the given parameters", "Creates a new, unsigned (unproven) stake transaction using the given parameters", &CreateRawStakeTransaction{opts: &opts})
	parser.AddCommand("decodetransaction", "Decode a serialized transaction", "Decodes a serialized transaction in hex format and prints out the JSON", &DecodeTransaction{opts: &opts})
	parser.AddCommand("decoderawtransaction", "Decode a raw transaction", "Decodes a raw transaction in hex format and prints out the JSON", &DecodeRawTransaction{opts: &opts})
	parser.AddCommand("proverawtransaction", "Creates the zk-proof for the transaction", "Creates the zk-proof for the transaction. Assuming there are no errors, this transaction should be ready for broadcast.", &ProveRawTransaction{opts: &opts})
	parser.AddCommand("stake", "Stakes the selected wallet UTXOs and turns the node into a validator", "Stakes the selected wallet UTXOs and turns the node into a validator", &Stake{opts: &opts})
	parser.AddCommand("setautostakerewards", "Automatically stakes validator rewards", "Automatically stakes validator rewards", &SetAutoStakeRewards{opts: &opts})
	parser.AddCommand("spend", "Sends coins from the wallet", "Sends coins from the wallet according to the provided parameters", &Spend{opts: &opts})
	parser.AddCommand("timelockcoins", "Lock coins in a timelocked address", "Send coins into a timelocked address, from which the wallet may spend from after the timelock expires. This is primarily used for adding weight to stake.", &TimelockCoins{opts: &opts})

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

	var (
		creds credentials.TransportCredentials
		err   error
	)
	if opts.RPCCert != "" {
		creds, err = credentials.NewClientTLSFromFile(certFile, "")
		if err != nil {
			return nil, err
		}
	} else {
		creds = credentials.NewClientTLSFromCert(nil, "")
	}
	ma, err := multiaddr.NewMultiaddr(opts.ServerAddr)
	if err != nil {
		return nil, err
	}

	netAddr, err := manet.ToNetAddr(ma)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(netAddr.String(), grpc.WithTransportCredentials(creds), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1000000)))
	if err != nil {
		return nil, err
	}
	return pb.NewBlockchainServiceClient(conn), nil
}

func makeNodeClient(opts *options) (pb.NodeServiceClient, error) {
	certFile := repo.CleanAndExpandPath(opts.RPCCert)

	var (
		creds credentials.TransportCredentials
		err   error
	)
	if opts.RPCCert != "" {
		creds, err = credentials.NewClientTLSFromFile(certFile, "")
		if err != nil {
			return nil, err
		}
	} else {
		creds = credentials.NewClientTLSFromCert(nil, "")
	}
	ma, err := multiaddr.NewMultiaddr(opts.ServerAddr)
	if err != nil {
		return nil, err
	}

	netAddr, err := manet.ToNetAddr(ma)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(netAddr.String(), grpc.WithTransportCredentials(creds), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1000000)))
	if err != nil {
		return nil, err
	}
	return pb.NewNodeServiceClient(conn), nil
}

func makeWalletClient(opts *options) (pb.WalletServiceClient, error) {
	certFile := repo.CleanAndExpandPath(opts.RPCCert)

	var (
		creds credentials.TransportCredentials
		err   error
	)
	if opts.RPCCert != "" {
		creds, err = credentials.NewClientTLSFromFile(certFile, "")
		if err != nil {
			return nil, err
		}
	} else {
		creds = credentials.NewClientTLSFromCert(nil, "")
	}
	ma, err := multiaddr.NewMultiaddr(opts.ServerAddr)
	if err != nil {
		return nil, err
	}

	netAddr, err := manet.ToNetAddr(ma)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(netAddr.String(), grpc.WithTransportCredentials(creds), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1000000)))
	if err != nil {
		return nil, err
	}
	return pb.NewWalletServiceClient(conn), nil
}
