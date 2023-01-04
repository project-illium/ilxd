// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circuits/stake"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"github.com/project-illium/ilxd/zk/scripts/transfer"
	"time"
)

func (h *TestHarness) generateBlocks(nBlocks int) ([]*blocks.Block, map[types.Nullifier]*SpendableNote, error) {
	newBlocks := make([]*blocks.Block, 0, nBlocks)
	acc := h.acc.Clone()
	fee := uint64(1)
	nCommitments := acc.NumElements()
	bestID, bestHeight := h.chain.BestBlock()

	remainingNotes := make(map[types.Nullifier]*SpendableNote)
	for k, v := range h.spendableNotes {
		remainingNotes[k] = v
	}

	for n := 0; n < nBlocks; n++ {
		outputsPerTx := h.txsPerBlock
		numTxs := h.txsPerBlock
		if len(remainingNotes) < h.txsPerBlock {
			outputsPerTx = h.txsPerBlock / len(remainingNotes)
			numTxs = len(remainingNotes)
		}

		notes := make([]*SpendableNote, 0, len(remainingNotes))
		for _, note := range remainingNotes {
			notes = append(notes, note)
		}

		toDelete := make([]types.Nullifier, 0, len(remainingNotes))
		txs := make([]*transactions.Transaction, 0, len(remainingNotes))
		for i := 0; i < numTxs; i++ {
			sn := notes[i]

			commitment, err := sn.Note.Commitment()
			if err != nil {
				return nil, nil, err
			}
			inclusionProof, err := acc.GetProof(commitment)
			if err != nil {
				return nil, nil, err
			}

			nullifier, err := types.CalculateNullifier(inclusionProof.Index, sn.Note.Salt, sn.Note.UnlockingScript.SnarkVerificationKey, sn.Note.UnlockingScript.PublicParams...)
			if err != nil {
				return nil, nil, err
			}
			toDelete = append(toDelete, nullifier)

			var (
				outputs     = make([]*transactions.Output, 0, outputsPerTx)
				outputNotes = make([]*SpendableNote, 0, outputsPerTx)
			)

			for x := 0; x < outputsPerTx; x++ {
				nCommitments++
				privKey, pubKey, err := crypto.GenerateEd25519Key(rand.Reader)
				if err != nil {
					return nil, nil, err
				}
				pubKeyBytes, err := pubKey.Raw()
				if err != nil {
					return nil, nil, err
				}
				_, verificationKey, err := crypto.GenerateEd25519Key(rand.Reader)
				if err != nil {
					return nil, nil, err
				}
				verificationKeyBytes, err := verificationKey.Raw()
				if err != nil {
					return nil, nil, err
				}

				var salt [32]byte
				rand.Read(salt[:])

				outputNote := &types.SpendNote{
					UnlockingScript: types.UnlockingScript{
						SnarkVerificationKey: verificationKeyBytes,
						PublicParams:         [][]byte{pubKeyBytes},
					},
					Amount:  (sn.Note.Amount / uint64(outputsPerTx)) - fee,
					AssetID: types.IlliumCoinID,
					Salt:    salt,
					State:   [32]byte{},
				}
				outputNotes = append(outputNotes, &SpendableNote{
					Note:       outputNote,
					PrivateKey: privKey,
				})

				outputCommitment, err := outputNote.Commitment()
				if err != nil {
					return nil, nil, err
				}

				outNullifier, err := types.CalculateNullifier(nCommitments-1, outputNote.Salt, outputNote.UnlockingScript.SnarkVerificationKey, outputNote.UnlockingScript.PublicParams...)
				if err != nil {
					return nil, nil, err
				}

				remainingNotes[outNullifier] = &SpendableNote{
					Note:       outputNote,
					PrivateKey: privKey,
				}

				outputs = append(outputs, &transactions.Output{
					Commitment:      outputCommitment,
					EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
					Ciphertext:      make([]byte, blockchain.CipherTextLen),
				})
			}
			standardTx := &transactions.StandardTransaction{
				Outputs:    outputs,
				Fee:        1,
				Nullifiers: [][]byte{nullifier.Bytes()},
				TxoRoot:    acc.Root().Bytes(),
				Proof:      nil,
			}

			sigHash, err := standardTx.SigHash()
			if err != nil {
				return nil, nil, err
			}

			mockUnlockingProof := make([]byte, 3500)
			rand.Read(mockUnlockingProof)

			privateParams := &standard.PrivateParams{
				Inputs: []standard.PrivateInput{
					{
						Amount:          sn.Note.Amount,
						Salt:            sn.Note.Salt,
						AssetID:         sn.Note.AssetID,
						State:           [32]byte{},
						CommitmentIndex: inclusionProof.Index,
						InclusionProof: standard.InclusionProof{
							Hashes:      inclusionProof.Hashes,
							Flags:       inclusionProof.Flags,
							Accumulator: inclusionProof.Accumulator,
						},
						SnarkVerificationKey: sn.Note.UnlockingScript.SnarkVerificationKey,
						UserParams:           sn.Note.UnlockingScript.PublicParams,
						SnarkProof:           mockUnlockingProof,
					},
				},
			}
			for _, outNote := range outputNotes {
				scriptHash := outNote.Note.UnlockingScript.Hash()
				privateParams.Outputs = append(privateParams.Outputs, standard.PrivateOutput{
					State:      [32]byte{},
					Amount:     outNote.Note.Amount,
					Salt:       outNote.Note.Salt,
					AssetID:    outNote.Note.AssetID,
					ScriptHash: scriptHash[:],
				})
			}

			publicOutputs := make([]standard.PublicOutput, len(outputNotes))
			for i, output := range outputs {
				publicOutputs[i] = standard.PublicOutput{
					Commitment: output.Commitment,
					EncKey:     output.EphemeralPubkey,
					CipherText: output.Ciphertext,
				}
			}

			publicPrams := &standard.PublicParams{
				TXORoot:    acc.Root().Bytes(),
				SigHash:    sigHash,
				Outputs:    publicOutputs,
				Nullifiers: [][]byte{nullifier.Bytes()},
				Fee:        fee,
				Coinbase:   0,
				MintID:     nil,
				MintAmount: 0,
				Locktime:   time.Time{},
			}

			proof, err := zk.CreateSnark(standard.StandardCircuit, privateParams, publicPrams)
			if err != nil {
				return nil, nil, err
			}
			standardTx.Proof = proof
			txs = append(txs, transactions.WrapTransaction(standardTx))
		}

		txids := make([][]byte, 0, len(txs))
		for _, tx := range txs {
			txids = append(txids, tx.ID().Bytes())
		}
		merkles := blockchain.BuildMerkleTreeStore(txids)

		h.timeSource++

		var (
			networkKey crypto.PrivKey
			validator  peer.ID
		)
		for k, v := range h.validators {
			networkKey = v.networkKey
			validator = k
		}
		valBytes, err := validator.Marshal()
		if err != nil {
			return nil, nil, err
		}

		blk := &blocks.Block{
			Header: &blocks.BlockHeader{
				Version:     1,
				Height:      bestHeight + 1,
				Parent:      bestID.Bytes(),
				Timestamp:   h.timeSource,
				TxRoot:      merkles[len(merkles)-1],
				Producer_ID: valBytes,
				Signature:   nil,
			},
			Transactions: txs,
		}

		sigHash, err := blk.Header.SigHash()
		if err != nil {
			return nil, nil, err
		}
		sig, err := networkKey.Sign(sigHash)
		if err != nil {
			return nil, nil, err
		}
		blk.Header.Signature = sig

		newBlocks = append(newBlocks, blk)
		bestHeight++
		bestID = blk.ID()

		for _, out := range blk.Outputs() {
			acc.Insert(out.Commitment, true)
		}

		for _, del := range toDelete {
			delete(remainingNotes, del)
		}
	}
	return newBlocks, remainingNotes, nil
}

func createGenesisBlock(params *params.NetworkParams, networkKey, spendKey crypto.PrivKey,
	initialCoins uint64, additionalOutputs []*transactions.Output) (*blocks.Block, *types.SpendNote, error) {

	// First we'll create the spend note for the coinbase transaction.
	// The initial coins will be generated to the spendKey.
	var salt1 [32]byte
	rand.Read(salt1[:])

	_, verificationKey, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	verificationKeyBytes, err := verificationKey.Raw()
	if err != nil {
		return nil, nil, err
	}
	spendPubkeyBytes, err := crypto.MarshalPublicKey(spendKey.GetPublic())
	if err != nil {
		return nil, nil, err
	}

	note1 := &types.SpendNote{
		UnlockingScript: types.UnlockingScript{
			SnarkVerificationKey: verificationKeyBytes,
			PublicParams:         [][]byte{spendPubkeyBytes},
		},
		Amount:  initialCoins / 2,
		AssetID: types.IlliumCoinID,
		Salt:    salt1,
		State:   [32]byte{},
	}

	var salt2 [32]byte
	rand.Read(salt2[:])

	note2 := &types.SpendNote{
		UnlockingScript: types.UnlockingScript{
			SnarkVerificationKey: verificationKeyBytes,
			PublicParams:         [][]byte{spendPubkeyBytes},
		},
		Amount:  initialCoins / 2,
		AssetID: types.IlliumCoinID,
		Salt:    salt2,
		State:   [32]byte{},
	}

	// Next we're going to start building the coinbase transaction
	commitment1, err := note1.Commitment()
	if err != nil {
		return nil, nil, err
	}
	commitment2, err := note2.Commitment()
	if err != nil {
		return nil, nil, err
	}
	validatorID, err := peer.IDFromPublicKey(networkKey.GetPublic())
	if err != nil {
		return nil, nil, err
	}
	idBytes, err := validatorID.Marshal()
	if err != nil {
		return nil, nil, err
	}

	coinbaseTx := &transactions.CoinbaseTransaction{
		Validator_ID: idBytes,
		NewCoins:     initialCoins,
		Outputs: []*transactions.Output{
			{
				Commitment:      commitment1,
				EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
				Ciphertext:      make([]byte, blockchain.CipherTextLen),
			},
			{
				Commitment:      commitment2,
				EphemeralPubkey: make([]byte, blockchain.PubkeyLen),
				Ciphertext:      make([]byte, blockchain.CipherTextLen),
			},
		},
	}
	coinbaseTx.Outputs = append(coinbaseTx.Outputs, additionalOutputs...)

	// And now sign the coinbase transaction with the network key
	sigHash, err := coinbaseTx.SigHash()
	if err != nil {
		return nil, nil, err
	}

	sig, err := networkKey.Sign(sigHash)
	if err != nil {
		return nil, nil, err
	}
	coinbaseTx.Signature = sig

	// Finally we're going to create the zk-snark proof for the coinbase
	// transaction.
	spendScriptHash1 := note1.UnlockingScript.Hash()
	spendScriptHash2 := note2.UnlockingScript.Hash()

	nullifier1, err := types.CalculateNullifier(0, salt1, note1.UnlockingScript.SnarkVerificationKey, note1.UnlockingScript.PublicParams...)
	if err != nil {
		return nil, nil, err
	}
	nullifier2, err := types.CalculateNullifier(1, salt2, note2.UnlockingScript.SnarkVerificationKey, note2.UnlockingScript.PublicParams...)
	if err != nil {
		return nil, nil, err
	}

	publicParams := &standard.PublicParams{
		Outputs: []standard.PublicOutput{
			{
				Commitment: commitment1,
			},
			{
				Commitment: commitment2,
			},
		},
		Nullifiers: [][]byte{nullifier1.Bytes(), nullifier2.Bytes()},
		Fee:        0,
		Coinbase:   initialCoins,
	}
	privateParams := &standard.PrivateParams{
		Outputs: []standard.PrivateOutput{
			{
				ScriptHash: spendScriptHash1[:],
				Amount:     initialCoins / 2,
				Salt:       note1.Salt,
				AssetID:    note1.AssetID,
				State:      note1.State,
			},
			{
				ScriptHash: spendScriptHash2[:],
				Amount:     initialCoins / 2,
				Salt:       note2.Salt,
				AssetID:    note2.AssetID,
				State:      note2.State,
			},
		},
	}

	proof, err := zk.CreateSnark(standard.StandardCircuit, privateParams, publicParams)
	if err != nil {
		return nil, nil, err
	}
	coinbaseTx.Proof = proof

	// Next we have to build the transaction staking the coins generated
	// in the prior coinbase transaction. This is needed because if no
	// validators are set in the genesis block we can't move the chain
	// forward.
	//
	// Notice there is a special validation rule for the genesis block
	// that doesn't apply to any other block. Normally, transactions
	// must contain a txoRoot for a block already in the chain. However,
	// in the case of the genesis block there are no other blocks in the
	// chain yet. So the rules allow the genesis block to reference its
	// own txoRoot.
	acc := blockchain.NewAccumulator()
	for i, output := range coinbaseTx.Outputs {
		acc.Insert(output.Commitment, i == 0)
	}
	txoRoot := acc.Root()
	inclusionProof, err := acc.GetProof(commitment1)
	if err != nil {
		return nil, nil, err
	}

	stakeTx := &transactions.StakeTransaction{
		Validator_ID: idBytes,
		Amount:       initialCoins,
		Nullifier:    nullifier1.Bytes(),
		TxoRoot:      txoRoot.Bytes(), // See note above
	}

	// Sign the stake transaction
	sigHash2, err := stakeTx.SigHash()
	if err != nil {
		return nil, nil, err
	}

	sig2, err := networkKey.Sign(sigHash2)
	if err != nil {
		return nil, nil, err
	}
	stakeTx.Signature = sig2

	// And generate the zk-snark proof
	unlockingParams := &standard.UnlockingSnarkParams{PublicParams: standard.PublicParams{SigHash: sigHash2}, UserParams: [][]byte{spendPubkeyBytes}}
	sig3, err := spendKey.Sign(sigHash2)
	if err != nil {
		return nil, nil, err
	}
	unlockingPriv := &transfer.PrivateParams{Signature: sig3}
	unlockingProof, err := zk.CreateSnark(transfer.TransferScript, unlockingPriv, unlockingParams)
	if err != nil {
		return nil, nil, err
	}

	publicParams2 := &stake.PublicParams{
		TXORoot:   txoRoot.Bytes(),
		SigHash:   sigHash2,
		Amount:    initialCoins / 2,
		Nullifier: nullifier1.Bytes(),
	}
	privateParams2 := &stake.PrivateParams{
		AssetID:         types.IlliumCoinID,
		Salt:            salt1,
		State:           [32]byte{},
		CommitmentIndex: 0,
		InclusionProof: standard.InclusionProof{
			Hashes:      inclusionProof.Hashes,
			Flags:       inclusionProof.Flags,
			Accumulator: inclusionProof.Accumulator,
		},
		SnarkVerificationKey: verificationKeyBytes,
		UserParams:           [][]byte{spendPubkeyBytes},
		SnarkProof:           unlockingProof,
	}

	proof2, err := zk.CreateSnark(stake.StakeCircuit, privateParams2, publicParams2)
	if err != nil {
		return nil, nil, err
	}
	stakeTx.Proof = proof2

	// Now we add the transactions to the genesis block
	genesis := params.GenesisBlock
	genesis.Transactions = []*transactions.Transaction{
		transactions.WrapTransaction(coinbaseTx),
		transactions.WrapTransaction(stakeTx),
	}

	// And create the genesis merkle root
	txids := make([][]byte, 0, len(genesis.Transactions))
	for _, tx := range genesis.Transactions {
		txids = append(txids, tx.ID().Bytes())
	}
	merkles := blockchain.BuildMerkleTreeStore(txids)
	genesis.Header.TxRoot = merkles[len(merkles)-1]
	genesis.Header.Timestamp = time.Now().Unix()

	return &genesis, note2, nil
}
