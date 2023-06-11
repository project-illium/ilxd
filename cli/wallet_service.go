// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/project-illium/ilxd/rpc/pb"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/walletlib"
)

type GetBalance struct {
	opts *options
}

func (x *GetBalance) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetBalance(makeContext(x.opts.AuthToken), &pb.GetBalanceRequest{})
	if err != nil {
		return err
	}
	fmt.Println(resp.Balance)
	return nil
}

type GetWalletSeed struct {
	opts *options
}

func (x *GetWalletSeed) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetWalletSeed(makeContext(x.opts.AuthToken), &pb.GetWalletSeedRequest{})
	if err != nil {
		return err
	}
	fmt.Println(resp.MnemonicSeed)
	return nil
}

type GetAddress struct {
	opts *options
}

func (x *GetAddress) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetAddress(makeContext(x.opts.AuthToken), &pb.GetAddressRequest{})
	if err != nil {
		return err
	}
	fmt.Println(resp.Address)
	return nil
}

type GetAddresses struct {
	opts *options
}

func (x *GetAddresses) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	resp, err := client.GetAddresses(makeContext(x.opts.AuthToken), &pb.GetAddressesRequest{})
	if err != nil {
		return err
	}
	for _, addr := range resp.Addresses {
		fmt.Println(addr)
	}
	return nil
}

type GetNewAddress struct {
	opts *options
}

func (x *GetNewAddress) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	resp, err := client.GetNewAddress(makeContext(x.opts.AuthToken), &pb.GetNewAddressRequest{})
	if err != nil {
		return err
	}
	fmt.Println(resp.Address)
	return nil
}

type GetTransactions struct {
	opts *options
}

func (x *GetTransactions) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetTransactions(makeContext(x.opts.AuthToken), &pb.GetTransactionsRequest{})
	if err != nil {
		return err
	}
	type tx struct {
		Txid     types.HexEncodable `json:"txid"`
		NetCoins int64              `json:"netCoins"`
	}
	txs := make([]tx, 0, len(resp.Txs))
	for _, rtx := range resp.Txs {
		txs = append(txs, tx{
			Txid:     rtx.Transaction_ID,
			NetCoins: rtx.NetCoins,
		})
	}
	out, err := json.MarshalIndent(txs, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

type GetUtxos struct {
	opts *options
}

func (x *GetUtxos) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	resp, err := client.GetUtxos(makeContext(x.opts.AuthToken), &pb.GetUtxosRequest{})
	if err != nil {
		return err
	}
	type utxo struct {
		Address    string             `json:"address"`
		Commitment types.HexEncodable `json:"commitment"`
		Amount     uint64             `json:"amount"`
		WatchOnly  bool               `json:"watchOnly"`
	}
	utxos := make([]utxo, 0, len(resp.Utxos))
	for _, ut := range resp.Utxos {
		utxos = append(utxos, utxo{
			Address:    ut.Address,
			Commitment: ut.Commitment,
			Amount:     ut.Amount,
			WatchOnly:  ut.WatchOnly,
		})
	}
	out, err := json.MarshalIndent(utxos, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

type GetPrivateKey struct {
	Address string `short:"a" long:"addr" description:"The address to get the private key for"`
	opts    *options
}

func (x *GetPrivateKey) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetPrivateKey(makeContext(x.opts.AuthToken), &pb.GetPrivateKeyRequest{})
	if err != nil {
		return err
	}

	key, err := crypto.UnmarshalPrivateKey(resp.SerializedKeys)
	if err != nil {
		return err
	}
	walletKey, ok := key.(*walletlib.WalletPrivateKey)
	if !ok {
		return errors.New("error decoding key")
	}

	fmt.Println(walletlib.EncodePrivateKey(walletKey))
	return nil
}

type ImportAddress struct {
	Address          string `short:"a" long:"addr" description:"The address to import"`
	UnlockingScript  string `short:"u" long:"unlockingscript" description:"The unlocking script for the address. Serialized as hex string"`
	ViewPrivateKey   string `short:"k" long:"viewkey" description:"The view private key for the address. Serialized as hex string."`
	Rescan           bool   `short:"r" long:"rescan" description:"Whether or not to rescan the blockchain to try to detect transactions for this address."`
	RescanFromHeight uint32 `short:"h" long:"rescanheight" description:"The height of the chain to rescan from. Selecting a height close to the address birthday saves resources."`
	opts             *options
}

func (x *ImportAddress) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	unlockingScriptBytes, err := hex.DecodeString(x.UnlockingScript)
	if err != nil {
		return err
	}
	privKeyBytes, err := hex.DecodeString(x.ViewPrivateKey)
	if err != nil {
		return err
	}

	_, err = client.ImportAddress(makeContext(x.opts.AuthToken), &pb.ImportAddressRequest{
		Address:          x.Address,
		UnlockingScript:  unlockingScriptBytes,
		ViewPrivateKey:   privKeyBytes,
		Rescan:           x.Rescan,
		RescanFromHeight: x.RescanFromHeight,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type CreateMultisigSpendKeypair struct {
	opts *options
}

func (x *CreateMultisigSpendKeypair) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	resp, err := client.CreateMultisigSpendKeypair(makeContext(x.opts.AuthToken), &pb.CreateMultisigSpendKeypairRequest{})
	if err != nil {
		return err
	}

	kp := struct {
		PrivateKey types.HexEncodable `json:"privateKey"`
		PublicKey  types.HexEncodable `json:"publicKey"`
	}{
		PrivateKey: resp.Privkey,
		PublicKey:  resp.Pubkey,
	}
	out, err := json.MarshalIndent(&kp, "", "    ")
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

type CreateMultisigViewKeypair struct {
	opts *options
}

func (x *CreateMultisigViewKeypair) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	resp, err := client.CreateMultisigViewKeypair(makeContext(x.opts.AuthToken), &pb.CreateMultisigViewKeypairRequest{})
	if err != nil {
		return err
	}

	kp := struct {
		PrivateKey types.HexEncodable `json:"privateKey"`
		PublicKey  types.HexEncodable `json:"publicKey"`
	}{
		PrivateKey: resp.Privkey,
		PublicKey:  resp.Pubkey,
	}
	out, err := json.MarshalIndent(&kp, "", "    ")
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

type CreateMultisigAddress struct {
	ViewPubKey string   `short:"k" long:"viewpubkey" description:"The view public key for the address. Serialized as hex string."`
	Pubkeys    []string `short:"p" long:"pubkey" description:"One or more public keys to use with the address. Serialized as a hex string. Use this option more than once for more than one key."`
	Threshold  uint32   `short:"t" long:"threshold" description:"The number of keys needing to sign to the spend from this address."`
	opts       *options
}

func (x *CreateMultisigAddress) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	pubkeys := make([][]byte, 0, len(x.Pubkeys))
	for _, p := range x.Pubkeys {
		keyBytes, err := hex.DecodeString(p)
		if err != nil {
			return err
		}
		pubkeys = append(pubkeys, keyBytes)
	}

	viewKey, err := hex.DecodeString(x.ViewPubKey)
	if err != nil {
		return err
	}

	resp, err := client.CreateMultisigAddress(makeContext(x.opts.AuthToken), &pb.CreateMultisigAddressRequest{
		Pubkeys:    pubkeys,
		Threshold:  x.Threshold,
		ViewPubkey: viewKey,
	})
	if err != nil {
		return err
	}
	fmt.Println(resp.Address)
	return nil
}

type CreateMultiSignature struct {
	opts *options
}

func (x *CreateMultiSignature) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	// Implement logic here
	return nil
}

type ProveMultisig struct {
	opts *options
}

func (x *ProveMultisig) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	// Implement logic here
	return nil
}

type WalletLock struct {
	opts *options
}

func (x *WalletLock) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	_, err = client.WalletLock(makeContext(x.opts.AuthToken), &pb.WalletLockRequest{})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type WalletUnlock struct {
	Passphrase string `short:"p" long:"passphrase" description:"The wallet passphrase"`
	Duration   uint32 `short:"d" long:"duration" description:"The number of seconds to unlock the wallet for"`
	opts       *options
}

func (x *WalletUnlock) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	_, err = client.WalletUnlock(makeContext(x.opts.AuthToken), &pb.WalletUnlockRequest{
		Passphrase: x.Passphrase,
		Duration:   x.Duration,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type SetWalletPassphrase struct {
	Passphrase string `short:"p" long:"passphrase" description:"The passphrase to set"`
	opts       *options
}

func (x *SetWalletPassphrase) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	_, err = client.SetWalletPassphrase(makeContext(x.opts.AuthToken), &pb.SetWalletPassphraseRequest{
		Passphrase: x.Passphrase,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type ChangeWalletPassphrase struct {
	Passphrase    string `short:"p" long:"passphrase" description:"The wallet's current passphrase"`
	NewPassphrase string `short:"n" long:"newpassphrase" description:"The passphrase to change it to"`
	opts          *options
}

func (x *ChangeWalletPassphrase) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	_, err = client.ChangeWalletPassphrase(makeContext(x.opts.AuthToken), &pb.ChangeWalletPassphraseRequest{
		CurrentPassphrase: x.Passphrase,
		NewPassphrase:     x.NewPassphrase,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type DeletePrivateKeys struct {
	opts *options
}

func (x *DeletePrivateKeys) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	_, err = client.DeletePrivateKeys(makeContext(x.opts.AuthToken), &pb.DeletePrivateKeysRequest{})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type CreateRawTransaction struct {
	opts *options
}

func (x *CreateRawTransaction) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	// Implement logic here
	return nil
}

type ProveRawTransaction struct {
	opts *options
}

func (x *ProveRawTransaction) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	// Implement logic here
	return nil
}

type Stake struct {
	Commitment []string `short:"c" long:"commitment" description:"A utxo commitment to stake. Encoded as a hex string. You can stake more than one. To do so just use this option more than once."`
	opts       *options
}

func (x *Stake) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}
	
	commitments := make([][]byte, 0, len(x.Commitment))
	for _, c := range commitments {
		cBytes, err := hex.DecodeString(c)
		if err != nil {
			return err
		}
		commitments = append(commitments, cBytes)
	}

	_, err = client.Stake(makeContext(x.opts.AuthToken), &pb.StakeRequest{
		Commitments: commitments,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type SetAutoStakeRewards struct {
	Autostake bool `short:"a" long:"autostake" description:"Whether to turn on or off autostaking of rewards"`
	opts      *options
}

func (x *SetAutoStakeRewards) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	_, err = client.SetAutoStakeRewards(makeContext(x.opts.AuthToken), &pb.SetAutoStakeRewardsRequest{
		Autostake: x.Autostake,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type Spend struct {
	opts *options
}

func (x *Spend) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	// Implement logic here
	return nil
}
