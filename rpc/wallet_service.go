// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package rpc

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circparams"
	"github.com/project-illium/walletlib"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"time"

	pb "github.com/project-illium/ilxd/rpc/pb"
)

// GetBalance returns the combined balance of all addresses in the wallet
func (s *GrpcServer) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	balance, err := s.wallet.Balance()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.GetBalanceResponse{
		Balance: uint64(balance),
	}, nil
}

// GetWalletSeed returns the mnemonic seed for the wallet. If the wallet
// seed has been deleted via the `DeletePrivateKeys` RPC an error will be
// returned.
//
// **Requires wallet to be unlocked**
func (s *GrpcServer) GetWalletSeed(ctx context.Context, req *pb.GetWalletSeedRequest) (*pb.GetWalletSeedResponse, error) {
	seed, err := s.wallet.MnemonicSeed()
	if errors.Is(err, walletlib.ErrEncryptedKeychain) {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	if errors.Is(err, walletlib.ErrPublicOnlyKeychain) {
		return nil, status.Error(codes.Unavailable, err.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.GetWalletSeedResponse{
		MnemonicSeed: seed,
	}, nil
}

// GetAddress returns the most recent address of the wallet.
func (s *GrpcServer) GetAddress(ctx context.Context, req *pb.GetAddressRequest) (*pb.GetAddressResponse, error) {
	addr, err := s.wallet.Address()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.GetAddressResponse{
		Address: addr.String(),
	}, nil
}

// GetTimelockedAddress returns a timelocked address that cannot be spent
// from until the given timelock has passed.
//
// The private key used for this address is the same as the wallet's most
// recent spend key used in a basic address. This implies the key can be
// derived from seed, however the wallet will not detect incoming payments
// to this address unless the timelock is included in the utxo's state field.
func (s *GrpcServer) GetTimelockedAddress(ctx context.Context, req *pb.GetTimelockedAddressRequest) (*pb.GetTimelockedAddressResponse, error) {
	addr, err := s.wallet.TimelockedAddress(time.Unix(req.LockUntil, 0))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.GetTimelockedAddressResponse{
		Address: addr.String(),
	}, nil
}

// GetPublicAddress returns a public address for the wallet. This address type
// requires that the private output data that is normally encrypted with the
// recipient's view key be put in the transaction in the clear.
//
// The private key used for this address is the same as the wallet's most
// recent spend key used in a basic address. This implies the key can be
// derived from seed.
func (s *GrpcServer) GetPublicAddress(ctx context.Context, req *pb.GetPublicAddressRequest) (*pb.GetPublicAddressResponse, error) {
	addr, err := s.wallet.PublicAddress()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.GetPublicAddressResponse{
		Address: addr.String(),
	}, nil
}

// GetAddresses returns all the addresses create by the wallet.
func (s *GrpcServer) GetAddresses(ctx context.Context, req *pb.GetAddressesRequest) (*pb.GetAddressesResponse, error) {
	addrs, err := s.wallet.Addresses()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	resp := &pb.GetAddressesResponse{
		Addresses: make([]string, 0, 1),
	}
	for _, addr := range addrs {
		resp.Addresses = append(resp.Addresses, addr.String())
	}
	return resp, nil
}

// GetAddressInfo returns additional metadata about an address.
func (s *GrpcServer) GetAddressInfo(ctx context.Context, req *pb.GetAddressInfoRequest) (*pb.GetAddressInfoResponse, error) {
	addr, err := walletlib.DecodeAddress(req.Address, s.chainParams)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	addrInfo, err := s.wallet.AddressInfo(addr)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	lockingScript := types.LockingScript{
		ScriptCommitment: types.NewID(addrInfo.LockingScript.ScriptCommitment),
		LockingParams:    addrInfo.LockingScript.LockingParams,
	}
	lockingScript.Serialize()

	resp := &pb.GetAddressInfoResponse{
		Address:        addr.String(),
		LockingScript:  lockingScript.Serialize(),
		ViewPrivateKey: addrInfo.ViewPrivKey,
		WatchOnly:      addrInfo.WatchOnly,
	}
	return resp, nil
}

// GetNewAddress generates a new address and returns it. Both a new spend key
// and view key will be derived from the mnemonic seed.
func (s *GrpcServer) GetNewAddress(ctx context.Context, req *pb.GetNewAddressRequest) (*pb.GetNewAddressResponse, error) {
	addr, err := s.wallet.NewAddress()
	if errors.Is(err, walletlib.ErrEncryptedKeychain) {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	if errors.Is(err, walletlib.ErrPublicOnlyKeychain) {
		return nil, status.Error(codes.Unavailable, err.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.GetNewAddressResponse{
		Address: addr.String(),
	}, nil
}

// GetTransactions returns the list of transactions for the wallet
func (s *GrpcServer) GetTransactions(ctx context.Context, req *pb.GetTransactionsRequest) (*pb.GetTransactionsResponse, error) {
	txs, err := s.wallet.Transactions()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	resp := &pb.GetTransactionsResponse{
		Txs: make([]*pb.WalletTransaction, 0, 1),
	}
	for _, tx := range txs {
		resp.Txs = append(resp.Txs, &pb.WalletTransaction{
			Transaction_ID: tx.Txid[:],
			NetCoins:       int64(tx.AmountIn) - int64(tx.AmountOut),
			Inputs:         ioToPBio(tx.Inputs),
			Outputs:        ioToPBio(tx.Outputs),
		})
	}
	return resp, nil
}

// GetUtxos returns a list of the wallet's current unspent transaction outputs (UTXOs)
func (s *GrpcServer) GetUtxos(ctx context.Context, req *pb.GetUtxosRequest) (*pb.GetUtxosResponse, error) {
	notes, err := s.wallet.Notes()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	resp := &pb.GetUtxosResponse{
		Utxos: make([]*pb.Utxo, 0, 1),
	}
	for _, note := range notes {
		resp.Utxos = append(resp.Utxos, &pb.Utxo{
			Commitment:   note.Commitment,
			Amount:       note.Amount,
			Address:      note.Address,
			WatchOnly:    note.WatchOnly,
			Staked:       note.Staked,
			LockedUntill: note.LockedUntil,
		})
	}
	return resp, nil
}

// GetPrivateKey returns the serialized spend and view keys for the given address
//
// **Requires wallet to be unlocked**
func (s *GrpcServer) GetPrivateKey(ctx context.Context, req *pb.GetPrivateKeyRequest) (*pb.GetPrivateKeyResponse, error) {
	keys, err := s.wallet.PrivateKeys()
	if errors.Is(err, walletlib.ErrEncryptedKeychain) {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	if errors.Is(err, walletlib.ErrPublicOnlyKeychain) {
		return nil, status.Error(codes.Unavailable, err.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	for key, addr := range keys {
		if addr.String() == req.Address {
			ser, err := crypto.MarshalPrivateKey(&key)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			return &pb.GetPrivateKeyResponse{
				SerializedKeys: ser,
			}, nil
		}
	}
	return nil, status.Error(codes.NotFound, "address not found")
}

// ImportAddress imports a watch address into the wallet.
func (s *GrpcServer) ImportAddress(ctx context.Context, req *pb.ImportAddressRequest) (*pb.ImportAddressResponse, error) {
	addr, err := walletlib.DecodeAddress(req.Address, s.chainParams)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if len(req.LockingScript) < zk.CommitmentLen {
		return nil, status.Error(codes.InvalidArgument, "invalid locking script")
	}
	lockingScript := new(types.LockingScript)
	if err := lockingScript.Deserialize(req.LockingScript); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	privKey, err := crypto.UnmarshalPrivateKey(req.ViewPrivateKey)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	err = s.wallet.ImportAddress(addr, *lockingScript, privKey, req.Rescan, req.RescanFromHeight)
	return &pb.ImportAddressResponse{}, err
}

// CreateMultisigSpendKeypair generates a spend keypair for use in a multisig address
func (s *GrpcServer) CreateMultisigSpendKeypair(ctx context.Context, req *pb.CreateMultisigSpendKeypairRequest) (*pb.CreateMultisigSpendKeypairResponse, error) {
	priv, pub, err := icrypto.GenerateNovaKey(rand.Reader)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	privBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	pubBytes, err := crypto.MarshalPublicKey(pub)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.CreateMultisigSpendKeypairResponse{
		Privkey: privBytes,
		Pubkey:  pubBytes,
	}, nil
}

// CreateMultisigViewKeypair generates a view keypair for use in a multisig address
func (s *GrpcServer) CreateMultisigViewKeypair(ctx context.Context, req *pb.CreateMultisigViewKeypairRequest) (*pb.CreateMultisigViewKeypairResponse, error) {
	priv, pub, err := icrypto.GenerateCurve25519Key(rand.Reader)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	privBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	pubBytes, err := crypto.MarshalPublicKey(pub)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.CreateMultisigViewKeypairResponse{
		Privkey: privBytes,
		Pubkey:  pubBytes,
	}, nil
}

// CreateMultisigAddress generates a new multisig address using the provided public keys
//
// Note this address is *not* imported. You will need to call `ImportAddress` if you want to watch
// it.
func (s *GrpcServer) CreateMultisigAddress(ctx context.Context, req *pb.CreateMultisigAddressRequest) (*pb.CreateMultisigAddressResponse, error) {
	threshold := make([]byte, 4)
	binary.BigEndian.PutUint32(threshold, req.Threshold)

	scriptCommitment, err := zk.LurkCommit(zk.MultisigScript())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	lockingScript := types.LockingScript{
		ScriptCommitment: types.NewID(scriptCommitment),
		LockingParams:    [][]byte{threshold},
	}
	for _, key := range req.Pubkeys {
		pubkey, err := crypto.UnmarshalPublicKey(key)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		novaKey, ok := pubkey.(*icrypto.NovaPublicKey)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "Public key is not type Nova")
		}

		x, y := novaKey.ToXY()
		lockingScript.LockingParams = append(lockingScript.LockingParams, x, y)
	}

	viewKey, err := crypto.UnmarshalPublicKey(req.ViewPubkey)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	addr, err := walletlib.NewBasicAddress(lockingScript, viewKey, s.chainParams)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.CreateMultisigAddressResponse{
		Address: addr.String(),
	}, nil
}

// CreateMultiSignature generates and returns a signature for use when proving a multisig transaction
func (s *GrpcServer) CreateMultiSignature(ctx context.Context, req *pb.CreateMultiSignatureRequest) (*pb.CreateMultiSignatureResponse, error) {
	privKey, err := crypto.UnmarshalPrivateKey(req.PrivateKey)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	var sigHash []byte
	if req.GetTx() != nil {
		tx := req.GetTx()
		standardTx := tx.GetStandardTransaction()
		if standardTx == nil {
			return nil, status.Error(codes.InvalidArgument, "standard transaction missing")
		}
		sigHash, err = standardTx.SigHash()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else if req.GetSighash() != nil {
		sigHash = req.GetSighash()
	} else {
		return nil, status.Error(codes.InvalidArgument, "transaction or sighash required")
	}
	sig, err := privKey.Sign(sigHash)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.CreateMultiSignatureResponse{
		Signature: sig,
	}, nil
}

// ProveMultisig creates a proof for a transaction with a multisig input
func (s *GrpcServer) ProveMultisig(ctx context.Context, req *pb.ProveMultisigRequest) (*pb.ProveMultisigResponse, error) {
	if req.RawTx == nil {
		return nil, status.Error(codes.InvalidArgument, "raw tx is nil")
	}
	if req.RawTx.Tx == nil {
		return nil, status.Error(codes.InvalidArgument, "tx is nil")
	}

	standardTx := req.RawTx.Tx.GetStandardTransaction()
	if standardTx == nil {
		return nil, status.Error(codes.InvalidArgument, "standard tx is nil")
	}

	sighash, err := standardTx.SigHash()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Create the transaction zk proof
	privateParams := &circparams.StandardPrivateParams{
		Inputs:  []circparams.PrivateInput{},
		Outputs: []circparams.PrivateOutput{},
	}

	for _, in := range req.RawTx.Inputs {
		var keys []crypto.PubKey
		for i := 1; i < len(in.LockingParams); i += 2 {
			pubx, puby := in.LockingParams[i], in.LockingParams[i+1]
			pub, err := icrypto.PublicKeyFromXY(pubx, puby)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			keys = append(keys, pub)
		}

		unlockingParams, err := zk.MakeMultisigUnlockingParams(keys, req.Sigs, sighash)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		privIn := circparams.PrivateInput{
			Amount:          types.Amount(in.Amount),
			AssetID:         types.NewID(in.Asset_ID),
			Salt:            types.NewID(in.Salt),
			CommitmentIndex: in.TxoProof.Index,
			InclusionProof: circparams.InclusionProof{
				Hashes: in.TxoProof.Hashes,
				Flags:  in.TxoProof.Flags,
			},
			Script:          in.Script,
			LockingParams:   in.LockingParams,
			UnlockingParams: unlockingParams,
		}

		state := new(types.State)
		if err := state.Deserialize(in.State); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		privIn.State = *state

		privateParams.Inputs = append(privateParams.Inputs, privIn)
	}
	for _, out := range req.RawTx.Outputs {
		privOut := circparams.PrivateOutput{
			ScriptHash: types.NewID(out.ScriptHash),
			Amount:     types.Amount(out.Amount),
			AssetID:    types.NewID(out.Asset_ID),
			Salt:       types.NewID(out.Salt),
		}
		state := new(types.State)
		if err := state.Deserialize(out.State); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		privOut.State = *state

		privateParams.Outputs = append(privateParams.Outputs, privOut)
	}

	publicParams, err := standardTx.ToCircuitParams()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	proof, err := s.prover.Prove(zk.StandardValidationProgram(), privateParams, publicParams)
	if err != nil {
		return nil, err
	}

	standardTx.Proof = proof

	return &pb.ProveMultisigResponse{
		ProvedTx: transactions.WrapTransaction(standardTx),
	}, nil
}

// WalletLock encrypts the wallet's private keys
func (s *GrpcServer) WalletLock(ctx context.Context, req *pb.WalletLockRequest) (*pb.WalletLockResponse, error) {
	err := s.wallet.Lock()
	return &pb.WalletLockResponse{}, err
}

// WalletUnlock decrypts the wallet seed and holds it in memory for the specified period of time
func (s *GrpcServer) WalletUnlock(ctx context.Context, req *pb.WalletUnlockRequest) (*pb.WalletUnlockResponse, error) {
	err := s.wallet.Unlock(req.Passphrase, time.Second*time.Duration(req.Duration))
	return &pb.WalletUnlockResponse{}, err
}

// SetWalletPassphrase encrypts the wallet for the first time
func (s *GrpcServer) SetWalletPassphrase(ctx context.Context, req *pb.SetWalletPassphraseRequest) (*pb.SetWalletPassphraseResponse, error) {
	err := s.wallet.SetWalletPassphrase(req.Passphrase)
	return &pb.SetWalletPassphraseResponse{}, err
}

// ChangeWalletPassphrase changes the passphrase used to encrypt the wallet private keys
func (s *GrpcServer) ChangeWalletPassphrase(ctx context.Context, req *pb.ChangeWalletPassphraseRequest) (*pb.ChangeWalletPassphraseResponse, error) {
	err := s.wallet.ChangeWalletPassphrase(req.CurrentPassphrase, req.NewPassphrase)
	return &pb.ChangeWalletPassphraseResponse{}, err
}

// DeletePrivateKeys deletes the wallet's private keys and seed from disk essentially turning the wallet
// into a watch-only wallet. It will still record incoming transactions but cannot spend them.
//
// **Requires wallet to be unlocked**
func (s *GrpcServer) DeletePrivateKeys(ctx context.Context, req *pb.DeletePrivateKeysRequest) (*pb.DeletePrivateKeysResponse, error) {
	err := s.wallet.PrunePrivateKeys()
	if errors.Is(err, walletlib.ErrEncryptedKeychain) {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	return nil, err
}

// CreateRawTransaction creates a new, unsigned (unproven) transaction using the given parameters
func (s *GrpcServer) CreateRawTransaction(ctx context.Context, req *pb.CreateRawTransactionRequest) (*pb.CreateRawTransactionResponse, error) {
	inputs := make([]*walletlib.RawInput, 0, len(req.Inputs))
	for _, in := range req.Inputs {
		rawInput := &walletlib.RawInput{}
		if in.GetCommitment() != nil {
			rawInput.Commitment = in.GetCommitment()
		} else if in.GetInput() != nil {
			rawInput.PrivateInput = &circparams.PrivateInput{
				Amount:          types.Amount(in.GetInput().Amount),
				AssetID:         types.NewID(in.GetInput().Asset_ID),
				Salt:            types.NewID(in.GetInput().Salt),
				Script:          in.GetInput().Script,
				LockingParams:   in.GetInput().LockingParams,
				UnlockingParams: "",
			}
			state := new(types.State)
			if err := state.Deserialize(in.GetInput().State); err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			rawInput.PrivateInput.State = *state
		} else {
			return nil, status.Error(codes.InvalidArgument, "input must have commitment or private input")
		}

		inputs = append(inputs, rawInput)
	}
	outputs := make([]*walletlib.RawOutput, 0, len(req.Outputs))
	for _, out := range req.Outputs {
		addr, err := walletlib.DecodeAddress(out.Address, s.chainParams)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		rawOut := &walletlib.RawOutput{
			Addr:   addr,
			Amount: types.Amount(out.Amount),
		}
		state := new(types.State)
		if err := state.Deserialize(out.State); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		rawOut.State = *state
		outputs = append(outputs, rawOut)
	}

	rawTx, err := s.wallet.CreateRawTransaction(inputs, outputs, req.AppendChangeOutput, types.Amount(req.FeePerKilobyte))
	if err != nil {
		return nil, err
	}
	resp := &pb.CreateRawTransactionResponse{
		RawTx: &pb.RawTransaction{
			Tx:      rawTx.Tx,
			Inputs:  make([]*pb.PrivateInput, 0, len(rawTx.PrivateInputs)),
			Outputs: make([]*pb.PrivateOutput, 0, len(rawTx.PrivateOutputs)),
		},
	}

	for i, in := range rawTx.PrivateInputs {
		var commitment types.ID

		if req.Inputs[i].GetCommitment() != nil {
			commitment = types.NewID(req.Inputs[i].GetCommitment())
		} else {
			scriptCommitment, err := zk.LurkCommit(in.Script)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			lockingScript := types.LockingScript{
				ScriptCommitment: types.NewID(scriptCommitment),
				LockingParams:    in.LockingParams,
			}
			scriptHash, err := lockingScript.Hash()
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			note := types.SpendNote{
				ScriptHash: scriptHash,
				Amount:     in.Amount,
				AssetID:    in.AssetID,
				State:      in.State,
				Salt:       in.Salt,
			}
			commitment, err = note.Commitment()
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}

		rawIn := &pb.PrivateInput{
			Amount:        uint64(in.Amount),
			Asset_ID:      in.AssetID.Bytes(),
			Salt:          in.Salt.Bytes(),
			Script:        in.Script,
			LockingParams: in.LockingParams,
			TxoProof: &pb.TxoProof{
				Commitment: commitment.Bytes(),
				Hashes:     in.InclusionProof.Hashes,
				Flags:      in.InclusionProof.Flags,
				Index:      in.CommitmentIndex,
			},
		}
		ser, err := in.State.Serialize(true)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		rawIn.State = ser
		resp.RawTx.Inputs = append(resp.RawTx.Inputs, rawIn)
	}
	for _, out := range rawTx.PrivateOutputs {
		po := &pb.PrivateOutput{
			Amount:     uint64(out.Amount),
			Salt:       make([]byte, len(out.Salt)),
			Asset_ID:   make([]byte, len(out.AssetID)),
			State:      make([]byte, len(out.State)),
			ScriptHash: make([]byte, len(out.ScriptHash)),
		}
		copy(po.Salt, out.Salt.Bytes())
		copy(po.ScriptHash, out.ScriptHash.Bytes())
		copy(po.Asset_ID, out.AssetID.Bytes())
		ser, err := out.State.Serialize(true)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		po.State = ser
		resp.RawTx.Outputs = append(resp.RawTx.Outputs, po)
	}

	return resp, nil
}

// CreateRawStakeTransaction creates a new, unsigned (unproven) stake transaction using the given parameters
func (s *GrpcServer) CreateRawStakeTransaction(ctx context.Context, req *pb.CreateRawStakeTransactionRequest) (*pb.CreateRawStakeTransactionResponse, error) {
	rawInput := &walletlib.RawInput{}
	if req.Input.GetCommitment() != nil {
		rawInput.Commitment = req.Input.GetCommitment()
	} else if req.Input.GetInput() != nil {
		rawInput.PrivateInput = &circparams.PrivateInput{
			Amount:          types.Amount(req.Input.GetInput().Amount),
			AssetID:         types.NewID(req.Input.GetInput().Asset_ID),
			Salt:            types.NewID(req.Input.GetInput().Salt),
			Script:          req.Input.GetInput().Script,
			LockingParams:   req.Input.GetInput().LockingParams,
			UnlockingParams: "",
		}
		state := new(types.State)
		if err := state.Deserialize(req.Input.GetInput().State); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		rawInput.PrivateInput.State = *state

	} else {
		return nil, status.Error(codes.InvalidArgument, "input must have commitment or private input")
	}

	rawTx, err := s.wallet.CreateRawStakeTransaction(rawInput)
	if err != nil {
		return nil, err
	}
	resp := &pb.CreateRawStakeTransactionResponse{
		RawTx: &pb.RawTransaction{
			Tx:     rawTx.Tx,
			Inputs: make([]*pb.PrivateInput, 0, len(rawTx.PrivateInputs)),
		},
	}

	var commitment types.ID
	if req.Input.GetCommitment() != nil {
		commitment = types.NewID(req.Input.GetCommitment())
	} else {
		scriptCommitment, err := zk.LurkCommit(rawTx.PrivateInputs[0].Script)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		lockingScript := types.LockingScript{
			ScriptCommitment: types.NewID(scriptCommitment),
			LockingParams:    rawTx.PrivateInputs[0].LockingParams,
		}
		scriptHash, err := lockingScript.Hash()
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		note := types.SpendNote{
			ScriptHash: scriptHash,
			Amount:     types.Amount(rawTx.PrivateInputs[0].Amount),
			AssetID:    rawTx.PrivateInputs[0].AssetID,
			State:      rawTx.PrivateInputs[0].State,
			Salt:       rawTx.PrivateInputs[0].Salt,
		}
		commitment, err = note.Commitment()
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	in := &pb.PrivateInput{
		Amount:        uint64(rawTx.PrivateInputs[0].Amount),
		Salt:          rawTx.PrivateInputs[0].Salt[:],
		Asset_ID:      rawTx.PrivateInputs[0].AssetID[:],
		Script:        rawTx.PrivateInputs[0].Script[:],
		LockingParams: rawTx.PrivateInputs[0].LockingParams[:],
		TxoProof: &pb.TxoProof{
			Commitment: commitment[:],
			Hashes:     rawTx.PrivateInputs[0].InclusionProof.Hashes,
			Flags:      rawTx.PrivateInputs[0].InclusionProof.Flags,
			Index:      rawTx.PrivateInputs[0].CommitmentIndex,
		},
	}
	ser, err := rawTx.PrivateInputs[0].State.Serialize(true)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	in.State = ser
	resp.RawTx.Inputs = append(resp.RawTx.Inputs, in)

	return resp, nil
}

// ProveRawTransaction creates the zk-proof for the transaction. Assuming there are no errors, this
// transaction should be ready for broadcast.
func (s *GrpcServer) ProveRawTransaction(ctx context.Context, req *pb.ProveRawTransactionRequest) (*pb.ProveRawTransactionResponse, error) {
	if req.RawTx == nil {
		return nil, status.Error(codes.InvalidArgument, "raw tx is nil")
	}
	if req.RawTx.Tx == nil {
		return nil, status.Error(codes.InvalidArgument, "tx is nil")
	}

	if req.RawTx.Tx.GetStandardTransaction() != nil {
		standardTx := req.RawTx.Tx.GetStandardTransaction()
		sigHash, err := standardTx.SigHash()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		// Create the transaction zk proof
		privateParams := &circparams.StandardPrivateParams{
			Inputs:  []circparams.PrivateInput{},
			Outputs: []circparams.PrivateOutput{},
		}

		privkeyMap, err := s.wallet.PrivateKeys()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		for i, in := range req.RawTx.Inputs {
			if in.UnlockingParams == "" {
				scriptCommitment, err := zk.LurkCommit(in.Script)
				if err != nil {
					return nil, status.Error(codes.InvalidArgument, err.Error())
				}

				lockingScript := types.LockingScript{
					ScriptCommitment: types.NewID(scriptCommitment),
					LockingParams:    in.LockingParams,
				}
				scriptHash, err := lockingScript.Hash()
				if err != nil {
					return nil, status.Error(codes.InvalidArgument, err.Error())
				}

				var privKey crypto.PrivKey
				for k, addr := range privkeyMap {
					if addr.ScriptHash() == scriptHash {
						privKey = k.SpendKey()
						break
					}
				}
				if privKey == nil {
					return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("private key for input %d not found", i))
				}

				sig, err := privKey.Sign(sigHash)
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}

				sigRx, sigRy, sigS := icrypto.UnmarshalSignature(sig)
				in.UnlockingParams = fmt.Sprintf("(cons 0x%x (cons 0x%x (cons 0x%x nil)))", sigRx, sigRy, sigS)
			}
			privIn := circparams.PrivateInput{
				Amount:          types.Amount(in.Amount),
				AssetID:         types.NewID(in.Asset_ID),
				Salt:            types.NewID(in.Salt),
				CommitmentIndex: in.TxoProof.Index,
				InclusionProof: circparams.InclusionProof{
					Hashes: in.TxoProof.Hashes,
					Flags:  in.TxoProof.Flags,
				},
				Script:          in.Script,
				LockingParams:   in.LockingParams,
				UnlockingParams: in.UnlockingParams,
			}
			state := new(types.State)
			if err := state.Deserialize(in.State); err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			privIn.State = *state

			privateParams.Inputs = append(privateParams.Inputs, privIn)
		}

		for _, out := range req.RawTx.Outputs {
			privOut := circparams.PrivateOutput{
				ScriptHash: types.NewID(out.ScriptHash),
				Amount:     types.Amount(out.Amount),
				AssetID:    types.NewID(out.Asset_ID),
				Salt:       types.NewID(out.Salt),
			}
			state := new(types.State)
			if err := state.Deserialize(out.State); err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			privOut.State = *state
			privateParams.Outputs = append(privateParams.Outputs, privOut)
		}

		publicParams, err := standardTx.ToCircuitParams()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		proof, err := s.prover.Prove(zk.StandardValidationProgram(), privateParams, publicParams)
		if err != nil {
			return nil, err
		}

		standardTx.Proof = proof

		return &pb.ProveRawTransactionResponse{
			ProvedTx: transactions.WrapTransaction(standardTx),
		}, nil
	} else if req.RawTx.Tx.GetStakeTransaction() != nil {
		stakeTx := req.RawTx.Tx.GetStakeTransaction()
		sigHash, err := stakeTx.SigHash()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		if len(req.RawTx.Inputs) == 0 {
			return nil, status.Error(codes.InvalidArgument, "no inputs")
		}

		if req.RawTx.Inputs[0].UnlockingParams == "" {
			scriptCommitment, err := zk.LurkCommit(req.RawTx.Inputs[0].Script)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}

			privkeyMap, err := s.wallet.PrivateKeys()
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			lockingScript := types.LockingScript{
				ScriptCommitment: types.NewID(scriptCommitment),
				LockingParams:    req.RawTx.Inputs[0].LockingParams,
			}
			scriptHash, err := lockingScript.Hash()
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}

			var privKey crypto.PrivKey
			for k, addr := range privkeyMap {
				if addr.ScriptHash() == scriptHash {
					privKey = k.SpendKey()
					break
				}
			}
			if privKey == nil {
				return nil, status.Error(codes.InvalidArgument, "private key for input not found")
			}

			sig, err := privKey.Sign(sigHash)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			sigRx, sigRy, sigS := icrypto.UnmarshalSignature(sig)
			req.RawTx.Inputs[0].UnlockingParams = fmt.Sprintf("(cons 0x%x (cons 0x%x (cons 0x%x nil)))", sigRx, sigRy, sigS)
		}

		// Create the transaction zk proof
		privateParams := &circparams.StakePrivateParams{
			Amount:          types.Amount(req.RawTx.Inputs[0].Amount),
			AssetID:         types.NewID(req.RawTx.Inputs[0].Asset_ID),
			Salt:            types.NewID(req.RawTx.Inputs[0].Salt),
			CommitmentIndex: req.RawTx.Inputs[0].TxoProof.Index,
			InclusionProof: circparams.InclusionProof{
				Hashes: req.RawTx.Inputs[0].TxoProof.Hashes,
				Flags:  req.RawTx.Inputs[0].TxoProof.Flags,
			},
			Script:          req.RawTx.Inputs[0].Script,
			LockingParams:   req.RawTx.Inputs[0].LockingParams,
			UnlockingParams: req.RawTx.Inputs[0].UnlockingParams,
		}
		state := new(types.State)
		if err := state.Deserialize(req.RawTx.Inputs[0].State); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		privateParams.State = *state

		publicParams, err := stakeTx.ToCircuitParams()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		proof, err := s.prover.Prove(zk.StakeValidationProgram(), privateParams, publicParams)
		if err != nil {
			return nil, err
		}

		stakeTx.Proof = proof

		return &pb.ProveRawTransactionResponse{
			ProvedTx: transactions.WrapTransaction(stakeTx),
		}, nil
	}
	return nil, status.Error(codes.InvalidArgument, "tx must be either standard or stake type")
}

// Stake stakes the selected wallet UTXOs and turns the node into a validator
//
// **Requires wallet to be unlocked**
func (s *GrpcServer) Stake(ctx context.Context, req *pb.StakeRequest) (*pb.StakeResponse, error) {
	commitments := make([]types.ID, 0, len(req.Commitments))
	for _, c := range req.Commitments {
		commitments = append(commitments, types.NewID(c))
	}
	err := s.wallet.Stake(commitments)
	return &pb.StakeResponse{}, err
}

// SetAutoStakeRewards make it such that any validator rewards that are earned are automatically staked
//
// **Requires wallet to be unlocked**
func (s *GrpcServer) SetAutoStakeRewards(ctx context.Context, req *pb.SetAutoStakeRewardsRequest) (*pb.SetAutoStakeRewardsResponse, error) {
	err := s.autoStakeFunc(req.Autostake)
	return &pb.SetAutoStakeRewardsResponse{}, err
}

// Spend sends coins from the wallet according to the provided parameters
//
// **Requires wallet to be unlocked**
func (s *GrpcServer) Spend(ctx context.Context, req *pb.SpendRequest) (*pb.SpendResponse, error) {
	commitments := make([]types.ID, 0, len(req.InputCommitments))
	for _, c := range req.InputCommitments {
		commitments = append(commitments, types.NewID(c))
	}
	addr, err := walletlib.DecodeAddress(req.ToAddress, s.chainParams)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	txid, err := s.wallet.Spend(addr, types.Amount(req.Amount), types.Amount(req.FeePerKilobyte), commitments...)
	if err != nil {
		return nil, err
	}
	return &pb.SpendResponse{Transaction_ID: txid[:]}, nil
}

// TimelockCoins moves coins into a timelocked address using the requested timelock.
// The internal wallet will be able to spend the coins after the timelock expires and
// the transaction will be recoverable if the wallet is restored from seed.
//
// This RPC primarily exists to lock coins for staking purposes.
//
// **Requires wallet to be unlocked**
func (s *GrpcServer) TimelockCoins(ctx context.Context, req *pb.TimelockCoinsRequest) (*pb.TimelockCoinsResponse, error) {
	commitments := make([]types.ID, 0, len(req.InputCommitments))
	for _, c := range req.InputCommitments {
		commitments = append(commitments, types.NewID(c))
	}

	txid, err := s.wallet.TimelockCoins(types.Amount(req.Amount), time.Unix(req.LockUntil, 0), types.Amount(req.FeePerKilobyte), commitments...)
	if err != nil {
		return nil, err
	}
	return &pb.TimelockCoinsResponse{Transaction_ID: txid[:]}, nil
}

// SweepWallet sweeps all the coins from this wallet to the provided address.
// This RPC is provided so that you don't have to try to guess the correct fee
// to take the wallet's balance down to zero. Here the fee will be subtracted
// from the total funds.
//
// **Requires wallet to be unlocked**
func (s *GrpcServer) SweepWallet(ctx context.Context, req *pb.SweepWalletRequest) (*pb.SweepWalletResponse, error) {
	addr, err := walletlib.DecodeAddress(req.ToAddress, s.chainParams)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	commitments := make([]types.ID, 0, len(req.InputCommitments))
	for _, c := range req.InputCommitments {
		commitments = append(commitments, types.NewID(c))
	}
	txid, err := s.wallet.SweepWallet(addr, types.Amount(req.FeePerKilobyte), commitments...)
	if err != nil {
		return nil, err
	}
	return &pb.SweepWalletResponse{Transaction_ID: txid[:]}, nil
}

// SubscribeWalletTransactions subscribes to a stream of WalletTransactionsNotifications that return
// whenever a transaction belonging to the wallet finalizes.
func (s *GrpcServer) SubscribeWalletTransactions(req *pb.SubscribeWalletTransactionsRequest, stream pb.WalletService_SubscribeWalletTransactionsServer) error {
	sub := s.wallet.SubscribeTransactions()
	defer sub.Close()

	for {
		select {
		case walletTx := <-sub.C:
			if walletTx != nil {
				err := stream.Send(&pb.WalletTransactionNotification{
					Transaction: &pb.WalletTransaction{
						Transaction_ID: walletTx.Txid.Bytes(),
						NetCoins:       int64(walletTx.AmountIn) - int64(walletTx.AmountOut),
						Inputs:         ioToPBio(walletTx.Inputs),
						Outputs:        ioToPBio(walletTx.Outputs),
					},
					Block_ID:    walletTx.BlockID.Bytes(),
					BlockHeight: walletTx.BlockHeight,
				})
				if err == io.EOF {
					return nil
				} else if err != nil {
					return status.Error(codes.InvalidArgument, err.Error())
				}
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}

// SubscribeWalletSyncNotifications streams notifications about the status of the wallet sync.
func (s *GrpcServer) SubscribeWalletSyncNotifications(req *pb.SubscribeWalletSyncNotificationsRequest, stream pb.WalletService_SubscribeWalletSyncNotificationsServer) error {
	sub := s.wallet.SubscribeSyncNotifications()
	defer sub.Close()

	for {
		select {
		case notif := <-sub.C:
			if notif != nil {
				err := stream.Send(&pb.WalletSyncNotification{
					CurrentHeight: notif.CurrentBlock,
					BestHeight:    notif.BestBlock,
				})
				if err == io.EOF {
					return nil
				} else if err != nil {
					return status.Error(codes.InvalidArgument, err.Error())
				}
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}

func ioToPBio(ios []walletlib.IO) []*pb.WalletTransaction_IO {
	ret := make([]*pb.WalletTransaction_IO, 0, len(ios))
	for _, io := range ios {
		switch t := io.(type) {
		case *walletlib.TxIO:
			ret = append(ret, &pb.WalletTransaction_IO{
				IoType: &pb.WalletTransaction_IO_TxIo{
					TxIo: &pb.WalletTransaction_IO_TxIO{
						Address: t.Address.String(),
						Amount:  uint64(t.Amount),
					},
				},
			})
		case *walletlib.Unknown:
			ret = append(ret, &pb.WalletTransaction_IO{
				IoType: &pb.WalletTransaction_IO_Unknown_{
					Unknown: &pb.WalletTransaction_IO_Unknown{},
				},
			})
		}
	}
	return ret
}
