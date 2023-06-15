// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package repo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/gcash/bchutil"
	"github.com/jessevdk/go-flags"
	"github.com/multiformats/go-multiaddr"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultLogFilename    = "ilxd.log"
	defaultConfigFilename = "ilxd.conf"
	defaultGrpcPort       = 5001

	DefaultFeePerKilobyte = 10000
	DefaultMinimumStake   = 1000000
	DefaultMaxMessageSize = 1 << 23 // 8 MiB
	DefaultSoftLimit      = 1 << 20 // 1 MiB

	DefaultMaxBanscore = 100
	DefaultBanDuration = time.Hour * 24
)

var (
	DefaultHomeDir    = AppDataDir("ilxd", false)
	defaultConfigFile = filepath.Join(DefaultHomeDir, defaultConfigFilename)
)

// Config defines the configuration options for the node.
//
// See LoadConfig for details on the configuration load process.
type Config struct {
	ShowVersion        bool          `short:"v" long:"version" description:"Display version information and exit"`
	ConfigFile         string        `short:"C" long:"configfile" description:"Path to configuration file"`
	DataDir            string        `short:"d" long:"datadir" description:"Directory to store data"`
	LogDir             string        `long:"logdir" description:"Directory to log output"`
	WalletDir          string        `long:"walletdir" description:"Directory to store wallet data"`
	LogLevel           string        `short:"l" long:"loglevel" description:"Set the logging level [debug, info, notice, error, alert, critical, emergency]." default:"info"`
	EnableDebugLogging bool          `long:"debug" description:"Enable libp2p debug logging to the terminal"`
	SeedAddrs          []string      `long:"seedaddr" description:"Override the default seed addresses with the provided values"`
	ListenAddrs        []string      `long:"listenaddr" description:"Override the default listen addresses with the provided values"`
	Testnet            bool          `short:"t" long:"testnet" description:"Use the test network"`
	Regtest            bool          `short:"r" long:"regtest" description:"Use regression testing mode"`
	RegtestVal         bool          `long:"regtestval" description:"Set self as the regtest genesis validator. This can only be done on first startup."`
	DisableNATPortMap  bool          `long:"noupnp" description:"Disable use of upnp"`
	UserAgent          string        `long:"useragent" description:"A custom user agent to advertise to the network"`
	NoTxIndex          bool          `long:"notxindex" description:"Disable the transaction index"`
	DropTxIndex        bool          `long:"droptxindex" description:"Delete the tx index from the database"`
	MaxBanscore        uint32        `long:"maxbanscore" description:"The maximum ban score a peer is allowed to have before getting banned" default:"100"`
	BanDuration        time.Duration `long:"banduration" description:"The duration for which banned peers are banned for" default:"24h"`
	WalletSeed         string        `long:"walletseed" description:"A mnemonic seed to initialize the node with. This can only be used on first startup."`
	CoinbaseAddress    string        `long:"coinbaseaddr" description:"An optional address to send all coinbase rewards to. If this option is not used the wallet will automatically select an internal address."`

	Policy  Policy     `group:"Policy"`
	RPCOpts RPCOptions `group:"RPC Options"`
}

type Policy struct {
	MinFeePerKilobyte  uint64   `long:"minfeeperkilobyte" description:"The minimum fee per kilobyte that the node will accept in the mempool and generated blocks"`
	MinStake           uint64   `long:"minstake" description:"The minimum stake required to accept a stake tx into the mempool or a generated block"`
	TreasuryWhitelist  []string `long:"treasurywhitelist" description:"Allow these treasury txids into the mempool and generated blocks"`
	BlocksizeSoftLimit uint32   `long:"blocksizesoftlimit" description:"The maximum size block this node will generate"`
	MaxMessageSize     int      `long:"maxmessagesize" description:"The maximum size of a network message. This is a hard limit. Setting this value different than all other nodes could fork you off the network."`
}

type RPCOptions struct {
	RPCCert              string   `long:"rpccert" description:"A path to the SSL certificate to use with gRPC"`
	RPCKey               string   `long:"rpckey" description:"A path to the SSL key to use with gRPC"`
	ExternalIPs          []string `long:"externalips" description:"This option should be used to specify the external IP address if using the auto-generated SSL certificate."`
	GrpcListener         string   `long:"grpclisten" description:"Add an interface/port to listen for experimental gRPC connections in multiaddr format (default:/ip4/127.0.0.1/tcp/5001)"`
	GrpcAuthToken        string   `long:"grpcauthtoken" description:"Set a token here if you want to enable client authentication with gRPC."`
	DisableNodeService   bool     `long:"disablenodeservice" description:"Disable the node RPC service."`
	DisableWalletService bool     `long:"disablewalletservice" description:"Disable the wallet RPC service."`
}

// LoadConfig initializes and parses the config using a config file and command
// line options.
//
// The configuration proceeds as follows:
//  1. Start with a default config with sane settings
//  2. Pre-parse the command line to check for an alternative config file
//  3. Load configuration file overwriting defaults with any specified options
//  4. Parse CLI options and overwrite/add any specified options
//
// The above results in proper functionality without any config settings
// while still allowing the user to override settings with config files and
// command line options.  Command line options always take precedence.
func LoadConfig() (*Config, error) {
	// Default config.
	cfg := Config{
		DataDir:    DefaultHomeDir,
		ConfigFile: defaultConfigFile,
	}

	// Pre-parse the command line options to see if an alternative config
	// file or the version flag was specified.  Any errors aside from the
	// help message error can be ignored here since they will be caught by
	// the final parse below.
	preCfg := cfg
	preParser := flags.NewParser(&cfg, flags.HelpFlag)
	_, err := preParser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			return nil, err
		}
	}
	if cfg.DataDir != "" {
		preCfg.ConfigFile = filepath.Join(cfg.DataDir, defaultConfigFilename)
	}

	// Load additional config from file.
	var configFileError error
	parser := flags.NewParser(&cfg, flags.Default)

	// Show the version and exit if the version flag was specified.
	appName := filepath.Base(os.Args[0])
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	usageMessage := fmt.Sprintf("Use %s -h to show usage", appName)
	if cfg.ShowVersion {
		fmt.Println(appName, "version", VersionString())
		os.Exit(0)
	}

	if _, err := os.Stat(preCfg.ConfigFile); os.IsNotExist(err) {
		err := createDefaultConfigFile(preCfg.ConfigFile, cfg.Testnet)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating a "+
				"default config file: %v\n", err)
		}
	}

	err = flags.NewIniParser(parser).ParseFile(preCfg.ConfigFile)
	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			fmt.Fprintf(os.Stderr, "Error parsing config "+
				"file: %v\n", err)
			fmt.Fprintln(os.Stderr, usageMessage)
			return nil, err
		}
		configFileError = err
	}

	if cfg.Testnet && cfg.Regtest {
		return nil, errors.New("invalid combination of testnet and regtest")
	}

	netStr := "mainnet"
	if cfg.Testnet {
		netStr = "testnet"
	} else if cfg.Regtest {
		netStr = "regtest"
	}

	if cfg.LogDir == "" {
		cfg.LogDir = CleanAndExpandPath(path.Join(cfg.DataDir, "logs", netStr))
	}
	if cfg.WalletDir == "" {
		cfg.WalletDir = CleanAndExpandPath(path.Join(cfg.DataDir, "wallet", netStr))
		if _, err := os.Stat(cfg.WalletDir); os.IsNotExist(err) {
			err := os.MkdirAll(filepath.Dir(cfg.WalletDir), 0700)
			if err != nil {
				return nil, err
			}
		}
	}

	// Warn about missing config file only after all other configuration is
	// done.  This prevents the warning on help messages and invalid
	// options.  Note this should go directly before the return.
	if configFileError != nil {
		log.Errorf("%v", configFileError)
	}

	// Default RPC to listen on localhost only.
	if cfg.RPCOpts.GrpcListener == "" {
		addrs, err := net.LookupHost("localhost")
		if err != nil || len(addrs) == 0 {
			return nil, errors.New("error determining local host for grpc server")
		}
		ma, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", addrs[0], defaultGrpcPort))
		if err != nil {
			return nil, err
		}
		cfg.RPCOpts.GrpcListener = ma.String()
	}

	if cfg.RPCOpts.RPCCert == "" && cfg.RPCOpts.RPCKey == "" {
		cfg.RPCOpts.RPCCert = path.Join(cfg.DataDir, "rpc.cert")
		cfg.RPCOpts.RPCKey = path.Join(cfg.DataDir, "rpc.key")
	}

	cfg.DataDir = CleanAndExpandPath(path.Join(cfg.DataDir, netStr))
	if !fileExists(cfg.RPCOpts.RPCKey) && !fileExists(cfg.RPCOpts.RPCCert) {
		err := genCertPair(cfg.RPCOpts.RPCCert, cfg.RPCOpts.RPCKey, cfg.RPCOpts.ExternalIPs)
		if err != nil {
			return nil, err
		}
	}

	cfg.UserAgent = "/ilxd/" + VersionString() + "/" + cfg.UserAgent
	if cfg.Policy.MinFeePerKilobyte == 0 {
		cfg.Policy.MinFeePerKilobyte = DefaultFeePerKilobyte
	}
	if cfg.Policy.MinStake == 0 {
		cfg.Policy.MinStake = DefaultMinimumStake
	}
	if cfg.Policy.BlocksizeSoftLimit == 0 {
		cfg.Policy.BlocksizeSoftLimit = DefaultSoftLimit
	}
	if cfg.Policy.MaxMessageSize == 0 {
		cfg.Policy.MaxMessageSize = DefaultMaxMessageSize
	}

	return &cfg, nil
}

// createDefaultConfig copies the sample-obcrawler.conf content to the given destination path,
// and populates it with some randomly generated RPC username and password.
func createDefaultConfigFile(destinationPath string, testnet bool) error {
	// Create the destination directory if it does not exists
	err := os.MkdirAll(filepath.Dir(destinationPath), 0700)
	if err != nil {
		return err
	}

	sampleBytes, err := Asset("sample-ilxd.conf")
	if err != nil {
		return err
	}
	src := bytes.NewReader(sampleBytes)

	dest, err := os.OpenFile(destinationPath,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer dest.Close()

	// We copy every line from the sample config file to the destination,
	// only replacing the bootstrap addrs.
	reader := bufio.NewReader(src)
	for err != io.EOF {
		var line string
		line, err = reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}

		if _, err := dest.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

// CleanAndExpandPath expands environment variables and leading ~ in the
// passed path, cleans the result, and returns it.
func CleanAndExpandPath(path string) string {
	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		homeDir := filepath.Dir(DefaultHomeDir)
		path = strings.Replace(path, "~", homeDir, 1)
	}

	// NOTE: The os.ExpandEnv doesn't work with Windows-style %VARIABLE%,
	// but they variables can still be expanded via POSIX-style $VARIABLE.
	return filepath.Clean(os.ExpandEnv(path))
}

// genCertPair generates a key/cert pair to the paths provided.
func genCertPair(certFile, keyFile string, externalIPs []string) error {
	log.Infof("Generating TLS certificates...")

	org := "ilxd autogenerated cert"
	validUntil := time.Now().Add(10 * 365 * 24 * time.Hour)
	cert, key, err := bchutil.NewTLSCertPair(org, validUntil, externalIPs)
	if err != nil {
		return err
	}

	// Write cert and key files.
	if err = os.WriteFile(certFile, cert, 0666); err != nil {
		return err
	}
	if err = os.WriteFile(keyFile, key, 0600); err != nil {
		os.Remove(certFile)
		return err
	}

	log.Infof("Done generating TLS certificates")
	return nil
}

// filesExists reports whether the named file or directory exists.
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
