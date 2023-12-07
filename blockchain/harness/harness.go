// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"bytes"
	"embed"
	"encoding/binary"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"google.golang.org/protobuf/proto"
)

type SpendableNote struct {
	Note             *types.SpendNote
	UnlockingScript  *types.UnlockingScript
	PrivateKey       crypto.PrivKey
	cachedScriptHash types.ID
}

type validator struct {
	networkKey crypto.PrivKey
}

type TestHarness struct {
	chain          *blockchain.Blockchain
	acc            *blockchain.Accumulator
	spendableNotes map[types.Nullifier]*SpendableNote
	validators     map[peer.ID]*validator
	txsPerBlock    int
	timeSource     int64
	cfg            *config
}

//go:embed blocks/blocks.dat
var blocksData embed.FS

//go:embed blocks/blocks2.dat
var blocks2Data embed.FS

func NewTestHarness(opts ...Option) (*TestHarness, error) {
	var cfg config
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	harness := &TestHarness{
		acc:            blockchain.NewAccumulator(),
		spendableNotes: make(map[types.Nullifier]*SpendableNote),
		validators:     make(map[peer.ID]*validator),
		txsPerBlock:    cfg.nTxsPerBlock,
		cfg:            &cfg,
	}
	validatorID, err := peer.IDFromPrivateKey(cfg.networkKey)
	if err != nil {
		return nil, err
	}
	harness.validators[validatorID] = &validator{
		networkKey: cfg.networkKey,
	}

	var genesisBlock *blocks.Block
	if cfg.pregenerate > 0 || cfg.extension {
		data, err := blocksData.ReadFile("blocks/blocks.dat")
		if err != nil {
			return nil, err
		}
		file := bytes.NewReader(data)
		harness.acc = blockchain.NewAccumulator()
		for {
			lenBytes := make([]byte, 4)
			_, err := file.Read(lenBytes)
			if err != nil {
				return nil, err
			}
			l := binary.BigEndian.Uint32(lenBytes)
			blkBytes := make([]byte, l)
			_, err = file.Read(blkBytes)
			if err != nil {
				return nil, err
			}

			var blk blocks.Block
			if err := proto.Unmarshal(blkBytes, &blk); err != nil {
				return nil, err
			}
			for _, out := range blk.Outputs() {
				harness.acc.Insert(out.Commitment, false)
			}

			if blk.Header.Height == 0 {
				cfg.params.GenesisBlock = &blk
				harness.chain, err = blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Params(cfg.params))
				if err != nil {
					return nil, err
				}
				genesisBlock = cfg.params.GenesisBlock
				harness.timeSource = genesisBlock.Header.Timestamp
			} else {
				if err := harness.chain.ConnectBlock(&blk, blockchain.BFFastAdd); err != nil {
					return nil, err
				}
				harness.timeSource = blk.Header.Timestamp + 1
			}
			if cfg.extension && blk.Header.Height == 15000 {
				data, err := blocks2Data.ReadFile("blocks/blocks2.dat")
				if err != nil {
					return nil, err
				}
				file = bytes.NewReader(data)
				cfg.pregenerate = 25000
			}

			if blk.Header.Height == uint32(cfg.pregenerate)-1 {
				idBytes, err := validatorID.Marshal()
				if err != nil {
					return nil, err
				}
				mockStandardScriptCommitment := make([]byte, 32)

				pubx, puby := cfg.spendKey.GetPublic().(*icrypto.NovaPublicKey).ToXY()

				note1UnlockingScript := &types.UnlockingScript{
					ScriptCommitment: mockStandardScriptCommitment,
					ScriptParams:     [][]byte{pubx, puby},
				}
				note1ScriptHash, err := note1UnlockingScript.Hash()
				if err != nil {
					return nil, err
				}
				salt, err := types.RandomSalt()
				if err != nil {
					return nil, err
				}
				note1 := &types.SpendNote{
					ScriptHash: note1ScriptHash[:],
					Amount:     100000000000,
					AssetID:    types.IlliumCoinID,
					State:      [types.StateLen]byte{},
					Salt:       salt,
				}

				sn := &SpendableNote{
					Note:            note1,
					UnlockingScript: note1UnlockingScript,
					PrivateKey:      cfg.spendKey,
				}
				commitment := note1.Commitment()
				val, err := harness.chain.GetValidator(validatorID)
				if err != nil {
					return nil, err
				}
				tx := &transactions.CoinbaseTransaction{
					Validator_ID: idBytes,
					NewCoins:     uint64(val.UnclaimedCoins),
					Outputs: []*transactions.Output{
						{Commitment: commitment[:]},
					},
				}
				if err := harness.GenerateBlockWithTransactions([]*transactions.Transaction{
					transactions.WrapTransaction(tx),
				}, []*SpendableNote{sn}); err != nil {
					return nil, err
				}

				break
			}
		}
	} else {
		genesis, spendableNote, err := createGenesisBlock(cfg.params, cfg.networkKey, cfg.spendKey, cfg.initialCoins, cfg.genesisOutputs)
		if err != nil {
			return nil, err
		}
		cfg.params.GenesisBlock = genesis
		genesisBlock = genesis
		for _, output := range genesis.Outputs() {
			harness.acc.Insert(output.Commitment, true)
		}

		commitment := spendableNote.Note.Commitment()
		proof, err := harness.acc.GetProof(commitment[:])
		if err != nil {
			return nil, err
		}

		nullifier, err := types.CalculateNullifier(proof.Index, spendableNote.Note.Salt, spendableNote.UnlockingScript.ScriptCommitment, spendableNote.UnlockingScript.ScriptParams...)
		if err != nil {
			return nil, err
		}
		harness.spendableNotes[nullifier] = spendableNote

		chain, err := blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Params(cfg.params))
		if err != nil {
			return nil, err
		}
		harness.chain = chain
		harness.timeSource = genesisBlock.Header.Timestamp
	}

	return harness, nil
}

func (h *TestHarness) ValidatorKey() crypto.PrivKey {
	return h.cfg.networkKey
}

func (h *TestHarness) GenerateBlocks(n int) error {
	blks, notes, err := h.generateBlocks(n)
	if err != nil {
		return err
	}

	for _, blk := range blks {
		if err := h.chain.ConnectBlock(blk, blockchain.BFFastAdd); err != nil {
			return err
		}
		for _, out := range blk.Outputs() {
			h.acc.Insert(out.Commitment, true)
		}
	}
	h.spendableNotes = notes
	return nil
}

func (h *TestHarness) GenerateBlockWithTransactions(txs []*transactions.Transaction, createdNotes []*SpendableNote) error {
	blk, err := h.generateBlockWithTransactions(txs)
	if err != nil {
		return err
	}
	if err := h.chain.ConnectBlock(blk, blockchain.BFFastAdd); err != nil {
		return err
	}
	for _, out := range blk.Outputs() {
		h.acc.Insert(out.Commitment, true)
	}
	for _, sn := range createdNotes {
		commitment := sn.Note.Commitment()
		proof, err := h.acc.GetProof(commitment[:])
		if err != nil {
			return err
		}
		nullifier, err := types.CalculateNullifier(proof.Index, sn.Note.Salt, sn.UnlockingScript.ScriptCommitment, sn.UnlockingScript.ScriptParams...)
		if err != nil {
			return err
		}
		h.spendableNotes[nullifier] = sn
	}
	return nil
}

func (h *TestHarness) SpendableNotes() []*SpendableNote {
	notes := make([]*SpendableNote, 0, len(h.spendableNotes))
	for _, sn := range h.spendableNotes {
		notes = append(notes, sn)
	}
	return notes
}

func (h *TestHarness) Accumulator() *blockchain.Accumulator {
	return h.acc
}

func (h *TestHarness) Blockchain() *blockchain.Blockchain {
	return h.chain
}

func (h *TestHarness) Clone() (*TestHarness, error) {
	newHarness := &TestHarness{
		acc:            h.acc.Clone(),
		spendableNotes: make(map[types.Nullifier]*SpendableNote),
		validators:     make(map[peer.ID]*validator),
		txsPerBlock:    h.txsPerBlock,
		timeSource:     h.timeSource,
	}

	chain, err := blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Params(h.chain.Params()))
	if err != nil {
		return nil, err
	}
	_, bestH, _ := h.chain.BestBlock()
	for i := uint32(1); i <= bestH; i++ {
		blk, err := h.chain.GetBlockByHeight(i)
		if err != nil {
			return nil, err
		}
		err = chain.ConnectBlock(blk, blockchain.BFFastAdd)
		if err != nil {
			return nil, err
		}
	}
	newHarness.chain = chain

	for k, v := range h.spendableNotes {
		k2 := types.NewNullifier(make([]byte, len(k)))
		copy(k2[:], k[:])

		v2 := *v
		newHarness.spendableNotes[k2] = &v2
	}
	for k, v := range h.validators {
		k2 := k
		v2 := *v
		newHarness.validators[k2] = &v2
	}
	return newHarness, nil
}
