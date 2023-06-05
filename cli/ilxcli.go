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
)

const (
	authenticationTokenKey = "AuthenticationToken"
	defaultConfigFilename  = "ilxcli.conf"
)

type cfgfile struct {
	ConfigFile string `short:"C" long:"configfile" description:"Path to configuration file"`
}

type options struct {
	cfgfile
	ShowVersion bool   `short:"v" long:"version" description:"Display version information and exit"`
	AuthToken   string `short:"t" long:"authtoken" description:"The ilxd node gRPC authentican token if needed"`
	ServerAddr  string `short:"a" long:"serveraddr" description:"The address of the ilxd gRPC server (in multiaddr format)" default:"/ip4/127.0.0.1/tcp/5001"`
	RPCCert     string `long:"rpccert" description:"A path to the SSL certificate to use with gRPC (this is only need if using a self-signed cert)" default:"~/.ilxd/rpc.cert"`
}

func main() {
	var preCfg cfgfile
	parser := flags.NewParser(&preCfg, flags.Default)

	if preCfg.ConfigFile == "" {
		preCfg.ConfigFile = filepath.Join(repo.DefaultHomeDir, defaultConfigFilename)
	}

	var opts options
	parser = flags.NewParser(&opts, flags.Default)
	err := flags.NewIniParser(parser).ParseFile(preCfg.ConfigFile)
	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			fmt.Fprintf(os.Stderr, "Error parsing config "+
				"file: %v\n", err)
			usageMessage := "Use ilxcli -h to show usage"
			fmt.Fprintln(os.Stderr, usageMessage)
			log.Fatal(err)
		}
	}

	parser = flags.NewNamedParser("ilxcli", flags.HelpFlag)
	parser.AddGroup("Connection options", "Configuration options for connecting to ilxd", &opts)

	parser.AddCommand("getmempoolinfo", "Returns the state of the current mempool", "Returns the state of the current mempool", &GetMempoolInfo{&opts})
	parser.AddCommand("getmempool", "Returns all the transactions in the mempool", "Returns all the transactions in the mempool", &GetMempool{&opts})
	parser.AddCommand("getblockchaininfo", "Returns data about the blockchain", "Returns data about the blockchain including the most recent block hash and height", &GetBlockchainInfo{&opts})
	parser.AddCommand("getblockinfo", "Returns a block header plus some extra metadata", "Returns a block header plus some extra metadata", &GetBlockInfo{opts: &opts})
	if _, err := parser.Parse(); err != nil {
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
