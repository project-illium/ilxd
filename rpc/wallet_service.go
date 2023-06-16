// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package rpc

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"github.com/project-illium/walletlib"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
			Commitment: note.Commitment,
			Amount:     note.Amount,
			Address:    note.Address,
			WatchOnly:  note.WatchOnly,
			Staked:     note.Staked,
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
	if len(req.UnlockingScript) < hash.HashSize {
		return nil, status.Error(codes.InvalidArgument, "invalid unlocking script")
	}
	unlockingScript := types.UnlockingScript{
		ScriptCommitment: req.UnlockingScript[:hash.HashSize],
		ScriptParams:     [][]byte{req.UnlockingScript[hash.HashSize:]},
	}
	privKey, err := crypto.UnmarshalPrivateKey(req.ViewPrivateKey)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	err = s.wallet.ImportAddress(addr, unlockingScript, privKey, req.Rescan, req.RescanFromHeight)
	return &pb.ImportAddressResponse{}, err
}

// CreateMultisigSpendKeypair generates a spend keypair for use in a multisig address
func (s *GrpcServer) CreateMultisigSpendKeypair(ctx context.Context, req *pb.CreateMultisigSpendKeypairRequest) (*pb.CreateMultisigSpendKeypairResponse, error) {
	priv, pub, err := crypto.GenerateEd25519Key(rand.Reader)
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
	mockMultisigUnlockScriptCommitment := bytes.Repeat([]byte{0xee}, 32)
	threshold := make([]byte, 4)
	binary.BigEndian.PutUint32(threshold, req.Threshold)

	unlockingScript := types.UnlockingScript{
		ScriptCommitment: mockMultisigUnlockScriptCommitment,
		ScriptParams:     [][]byte{threshold},
	}
	unlockingScript.ScriptParams = append(unlockingScript.ScriptParams, req.Pubkeys...)

	viewKey, err := crypto.UnmarshalPublicKey(req.ViewPubkey)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	addr, err := walletlib.NewBasicAddress(unlockingScript, viewKey, s.chainParams)
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
	if req.Tx == nil {
		return nil, status.Error(codes.InvalidArgument, "raw tx is nil")
	}
	if req.Tx.Tx == nil {
		return nil, status.Error(codes.InvalidArgument, "tx is nil")
	}

	standardTx := req.Tx.Tx.GetStandardTransaction()
	if standardTx == nil {
		return nil, status.Error(codes.InvalidArgument, "standard tx is nil")
	}

	// Create the transaction zk proof
	privateParams := &standard.PrivateParams{
		Inputs:  []standard.PrivateInput{},
		Outputs: []standard.PrivateOutput{},
	}

	nullifiers := make([][]byte, 0, len(req.Tx.Inputs))
	for _, in := range req.Tx.Inputs {
		privIn := standard.PrivateInput{
			Amount:          in.Amount,
			CommitmentIndex: in.TxoProof.Index,
			InclusionProof: standard.InclusionProof{
				Hashes:      in.TxoProof.Hashes,
				Flags:       in.TxoProof.Flags,
				Accumulator: in.TxoProof.Accumulator,
			},
			ScriptCommitment: in.ScriptCommitment,
			ScriptParams:     in.ScriptParams,
			UnlockingParams:  req.Sigs,
		}
		copy(privIn.Salt[:], in.Salt)
		copy(privIn.AssetID[:], in.Asset_ID)
		copy(privIn.State[:], in.State)

		privateParams.Inputs = append(privateParams.Inputs, privIn)

		nullifier, err := types.CalculateNullifier(in.TxoProof.Index, privIn.Salt, privIn.ScriptCommitment, privIn.ScriptParams...)
		if err != nil {
			return nil, err
		}
		nullifiers = append(nullifiers, nullifier.Bytes())
	}
	for _, out := range req.Tx.Outputs {
		privOut := standard.PrivateOutput{
			ScriptHash: out.ScriptHash,
			Amount:     out.Amount,
		}
		copy(privOut.Salt[:], out.Salt)
		copy(privOut.AssetID[:], out.Asset_ID)
		copy(privOut.State[:], out.State)

		privateParams.Outputs = append(privateParams.Outputs, privOut)
	}

	sighash, err := standardTx.SigHash()
	if err != nil {
		return nil, err
	}
	publicParams := &standard.PublicParams{
		TXORoot:    standardTx.TxoRoot,
		SigHash:    sighash,
		Nullifiers: nullifiers,
		Fee:        standardTx.Fee,
	}

	for _, out := range standardTx.Outputs {
		publicParams.Outputs = append(publicParams.Outputs, standard.PublicOutput{
			Commitment: out.Commitment,
			CipherText: out.Ciphertext,
		})
	}

	proof, err := zk.CreateSnark(standard.StandardCircuit, privateParams, publicParams)
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
			rawInput.PrivateInput = &standard.PrivateInput{
				Amount:           in.GetInput().Amount,
				ScriptCommitment: in.GetInput().ScriptCommitment,
				ScriptParams:     in.GetInput().ScriptParams,
				UnlockingParams:  nil,
			}
			copy(rawInput.PrivateInput.Salt[:], in.GetInput().Salt)
			copy(rawInput.PrivateInput.AssetID[:], in.GetInput().Asset_ID)
			copy(rawInput.PrivateInput.State[:], in.GetInput().State)
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
		outputs = append(outputs, &walletlib.RawOutput{
			Addr:   addr,
			Amount: types.Amount(out.Amount),
		})
	}
	rawTx, err := s.wallet.CreateRawTransaction(inputs, outputs, req.AppendChangeOutput, types.Amount(req.FeePerKilobyte))
	if err != nil {
		return nil, err
	}
	resp := &pb.CreateRawTransactionResponse{
		Tx: &pb.RawTransaction{
			Tx:      transactions.WrapTransaction(rawTx.Tx),
			Inputs:  make([]*pb.PrivateInput, 0, len(rawTx.PrivateInputs)),
			Outputs: make([]*pb.PrivateOutput, 0, len(rawTx.PrivateOutputs)),
		},
	}
	for _, in := range rawTx.PrivateInputs {
		unlockingScript := types.UnlockingScript{
			ScriptCommitment: in.ScriptCommitment,
			ScriptParams:     in.ScriptParams,
		}
		scriptHash := unlockingScript.Hash()
		note := types.SpendNote{
			ScriptHash: scriptHash[:],
			Amount:     types.Amount(in.Amount),
			AssetID:    in.AssetID,
			State:      in.State,
			Salt:       in.Salt,
		}
		commitment, err := note.Commitment()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		resp.Tx.Inputs = append(resp.Tx.Inputs, &pb.PrivateInput{
			Amount:           in.Amount,
			Salt:             in.Salt[:],
			Asset_ID:         in.AssetID[:],
			State:            in.State[:],
			ScriptCommitment: in.ScriptCommitment[:],
			ScriptParams:     in.ScriptParams[:],
			TxoProof: &pb.TxoProof{
				Commitment:  commitment[:],
				Accumulator: in.InclusionProof.Accumulator,
				Hashes:      in.InclusionProof.Hashes,
				Flags:       in.InclusionProof.Flags,
				Index:       in.CommitmentIndex,
			},
		})
	}
	for _, out := range rawTx.PrivateOutputs {
		resp.Tx.Outputs = append(resp.Tx.Outputs, &pb.PrivateOutput{
			Amount:     out.Amount,
			Salt:       out.Salt[:],
			Asset_ID:   out.AssetID[:],
			State:      out.State[:],
			ScriptHash: out.ScriptHash,
		})
	}

	return resp, nil
}

// ProveRawTransaction creates the zk-proof for the transaction. Assuming there are no errors, this
// transaction should be ready for broadcast.
func (s *GrpcServer) ProveRawTransaction(ctx context.Context, req *pb.ProveRawTransactionRequest) (*pb.ProveRawTransactionResponse, error) {
	if req.Tx == nil {
		return nil, status.Error(codes.InvalidArgument, "raw tx is nil")
	}
	if req.Tx.Tx == nil {
		return nil, status.Error(codes.InvalidArgument, "tx is nil")
	}

	standardTx := req.Tx.Tx.GetStandardTransaction()
	if standardTx == nil {
		return nil, status.Error(codes.InvalidArgument, "standard tx is nil")
	}

	sigHash, err := standardTx.SigHash()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(req.PrivateKeys) != len(standardTx.Nullifiers) {
		return nil, status.Error(codes.InvalidArgument, "not enough private keys")
	}

	// Create the transaction zk proof
	privateParams := &standard.PrivateParams{
		Inputs:  []standard.PrivateInput{},
		Outputs: []standard.PrivateOutput{},
	}

	for i, in := range req.Tx.Inputs {
		privKey, err := crypto.UnmarshalPrivateKey(req.PrivateKeys[i])
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		switch k := privKey.(type) {
		case *crypto.Ed25519PrivateKey:
		case *walletlib.WalletPrivateKey:
			privKey = k.SpendKey()
		default:
			return nil, status.Error(codes.InvalidArgument, "unknown private key type")
		}
		sig, err := privKey.Sign(sigHash)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		privIn := standard.PrivateInput{
			Amount:          in.Amount,
			CommitmentIndex: in.TxoProof.Index,
			InclusionProof: standard.InclusionProof{
				Hashes:      in.TxoProof.Hashes,
				Flags:       in.TxoProof.Flags,
				Accumulator: in.TxoProof.Accumulator,
			},
			ScriptCommitment: in.ScriptCommitment,
			ScriptParams:     in.ScriptParams,
			UnlockingParams:  [][]byte{sig},
		}
		copy(privIn.Salt[:], in.Salt)
		copy(privIn.AssetID[:], in.Asset_ID)
		copy(privIn.State[:], in.State)

		privateParams.Inputs = append(privateParams.Inputs, privIn)
	}

	for _, out := range req.Tx.Outputs {
		privOut := standard.PrivateOutput{
			ScriptHash: out.ScriptHash,
			Amount:     out.Amount,
		}
		copy(privOut.Salt[:], out.Salt)
		copy(privOut.AssetID[:], out.Asset_ID)
		copy(privOut.State[:], out.State)
		privateParams.Outputs = append(privateParams.Outputs, privOut)
	}

	publicParams := &standard.PublicParams{
		TXORoot:    standardTx.TxoRoot,
		SigHash:    sigHash,
		Nullifiers: standardTx.Nullifiers,
		Fee:        standardTx.Fee,
	}

	for _, out := range standardTx.Outputs {
		publicParams.Outputs = append(publicParams.Outputs, standard.PublicOutput{
			Commitment: out.Commitment,
			CipherText: out.Ciphertext,
		})
	}

	proof, err := zk.CreateSnark(standard.StandardCircuit, privateParams, publicParams)
	if err != nil {
		return nil, err
	}

	standardTx.Proof = proof

	return &pb.ProveRawTransactionResponse{
		ProvedTx: transactions.WrapTransaction(standardTx),
	}, nil
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
