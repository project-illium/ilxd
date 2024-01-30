// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"embed"
	"encoding/binary"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circparams"
	"google.golang.org/protobuf/proto"
	"io"
	"os"
)

// SpendableNote holds all the information needed
// by the harness to make a spend.
type SpendableNote struct {
	Note             *types.SpendNote
	LockingScript    *types.LockingScript
	PrivateKey       crypto.PrivKey
	cachedScriptHash types.ID
}

type blockFile struct {
	f       io.ReadCloser
	nBlocks int
}

type validator struct {
	networkKey crypto.PrivKey
}

// TestHarness is a harness for the blockchain which
// creates a new instance of the blockchain and exposes
// methods for extending the chain.
type TestHarness struct {
	chain          *blockchain.Blockchain
	acc            *blockchain.Accumulator
	spendableNotes map[types.Nullifier]*SpendableNote
	validators     map[peer.ID]*validator
	txsPerBlock    int
	timeSource     int64
	prover         zk.Prover
	verifier       zk.Verifier
	cfg            *config
}

// BlocksData is a file containing 21000 blocks
//
//go:embed blocks/blocks.dat
var BlocksData embed.FS

// Blocks2Data is a file containing 6000 blocks
// extending the chain found in blocks.dat starting
// at block 15000
//
//go:embed blocks/blocks2.dat
var Blocks2Data embed.FS

// NewTestHarness returns a new TestHarness with the
// configured options.
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

	prover := &zk.MockProver{}
	prover.SetProofLen(1)
	verifier := &zk.MockVerifier{}
	verifier.SetValid(true)
	harness := &TestHarness{
		acc:            blockchain.NewAccumulator(),
		spendableNotes: make(map[types.Nullifier]*SpendableNote),
		validators:     make(map[peer.ID]*validator),
		txsPerBlock:    cfg.nTxsPerBlock,
		timeSource:     0,
		prover:         prover,
		verifier:       verifier,
		cfg:            &cfg,
	}
	validatorID, err := peer.IDFromPrivateKey(cfg.networkKey)
	if err != nil {
		return nil, err
	}
	harness.validators[validatorID] = &validator{
		networkKey: cfg.networkKey,
	}
	params := *cfg.params
	params.Name = "TestHarness"

	if len(harness.cfg.blockFiles) > 0 {
		harness.acc = blockchain.NewAccumulator()
		for _, blkFile := range harness.cfg.blockFiles {
			i := 1
			for {
				lenBytes := make([]byte, 4)
				_, err := blkFile.f.Read(lenBytes)
				if err != nil {
					return nil, err
				}
				l := binary.BigEndian.Uint32(lenBytes)
				blkBytes := make([]byte, l)
				_, err = blkFile.f.Read(blkBytes)
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
					params.GenesisBlock = &blk
					harness.chain, err = blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Params(&params), blockchain.Verifier(harness.verifier))
					if err != nil {
						return nil, err
					}
					harness.timeSource = blk.Header.Timestamp
				} else {
					if err := harness.chain.ConnectBlock(&blk, blockchain.BFFastAdd|blockchain.BFNoValidation); err != nil {
						return nil, err
					}
					harness.timeSource = blk.Header.Timestamp + 1
				}
				if i == blkFile.nBlocks {
					blkFile.f.Close()
					break
				}
				i++
			}
		}
	} else {
		genesis, spendableNote, err := createGenesisBlock(cfg.params, cfg.networkKey, cfg.spendKey, cfg.initialCoins, cfg.genesisOutputs, harness.prover)
		if err != nil {
			return nil, err
		}
		if harness.cfg.writeToFile != nil {
			if err := writeBlockToFile(harness.cfg.writeToFile, genesis); err != nil {
				return nil, err
			}
		}
		cfg.params.GenesisBlock = genesis
		for _, output := range genesis.Outputs() {
			harness.acc.Insert(output.Commitment, true)
		}

		commitment, err := spendableNote.Note.Commitment()
		if err != nil {
			return nil, err
		}
		proof, err := harness.acc.GetProof(commitment[:])
		if err != nil {
			return nil, err
		}

		nullifier, err := types.CalculateNullifier(proof.Index, spendableNote.Note.Salt, spendableNote.LockingScript.ScriptCommitment.Bytes(), spendableNote.LockingScript.LockingParams...)
		if err != nil {
			return nil, err
		}
		harness.spendableNotes[nullifier] = spendableNote
		harness.timeSource = genesis.Header.Timestamp + 300

		chain, err := blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Datastore(cfg.datastore), blockchain.Params(&params), blockchain.Verifier(harness.verifier))
		if err != nil {
			return nil, err
		}
		harness.chain = chain
	}

	return harness, nil
}

// Close closes the harness
func (h *TestHarness) Close() {
	h.Blockchain().Close()
	if h.cfg.writeToFile != nil {
		h.cfg.writeToFile.Close()
	}
}

// ValidatorKey returns the current validator private
// key for the harness.
func (h *TestHarness) ValidatorKey() crypto.PrivKey {
	return h.cfg.networkKey
}

// GenerateBlocks generates the provided number of blocks
// and connects them to the chain. This will error if there
// is an issue connecting the blocks.
func (h *TestHarness) GenerateBlocks(n int) error {
	if len(h.spendableNotes) == 0 {
		if _, err := h.GenerateNewCoinbase(); err != nil {
			return err
		}
	}
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
		if h.cfg.writeToFile != nil {
			if err := writeBlockToFile(h.cfg.writeToFile, blk); err != nil {
				return err
			}
		}
	}
	h.spendableNotes = notes
	return nil
}

// GenerateBlockWithTransactions allows for creating blocks with a list
// of custom transactions. To build the inputs for the transactions the
// SpendableNotes() method will need to be called to get input notes
// that can be spent. The output notes must also be provided so the
// harness can use them to generate future blocks.
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
		commitment, err := sn.Note.Commitment()
		if err != nil {
			return err
		}
		proof, err := h.acc.GetProof(commitment[:])
		if err != nil {
			return err
		}
		nullifier, err := types.CalculateNullifier(proof.Index, sn.Note.Salt, sn.LockingScript.ScriptCommitment.Bytes(), sn.LockingScript.LockingParams...)
		if err != nil {
			return err
		}
		h.spendableNotes[nullifier] = sn
	}
	if h.cfg.writeToFile != nil {
		if err := writeBlockToFile(h.cfg.writeToFile, blk); err != nil {
			return err
		}
	}
	return nil
}

// GenerateNewCoinbase will create a new SpendableNote from a
// coinbase transaction. The harness validator must have
// unclaimed coins in the chain for this to work.
//
// The primary use for this method is for creating new spendable
// note after loading blocks from disk as the harness does not
// have the information about previous notes in this case.
func (h *TestHarness) GenerateNewCoinbase() (*SpendableNote, error) {
	valID, err := peer.IDFromPrivateKey(h.cfg.networkKey)
	if err != nil {
		return nil, err
	}
	valBytes, err := valID.Marshal()
	if err != nil {
		return nil, err
	}
	val, err := h.chain.GetValidator(valID)
	if err != nil {
		return nil, err
	}

	pkx, pky := h.cfg.spendKey.GetPublic().(*icrypto.NovaPublicKey).ToXY()

	lockingScript := &types.LockingScript{
		ScriptCommitment: types.NewID(zk.BasicTransferScriptCommitment()),
		LockingParams:    [][]byte{pkx, pky},
	}

	scriptHash, err := lockingScript.Hash()
	if err != nil {
		return nil, err
	}

	salt, err := types.RandomSalt()
	if err != nil {
		return nil, err
	}

	note := &types.SpendNote{
		ScriptHash: scriptHash,
		Amount:     val.UnclaimedCoins,
		AssetID:    types.IlliumCoinID,
		Salt:       salt,
		State:      nil,
	}

	commitment, err := note.Commitment()
	if err != nil {
		return nil, err
	}

	tx := &transactions.CoinbaseTransaction{
		Validator_ID: valBytes,
		NewCoins:     uint64(val.UnclaimedCoins),
		Outputs: []*transactions.Output{
			{
				Commitment: commitment.Bytes(),
				Ciphertext: make([]byte, blockchain.CiphertextLen),
			},
		},
		Signature: nil,
		Proof:     nil,
	}

	sigHash, err := tx.SigHash()
	if err != nil {
		return nil, err
	}
	sig, err := h.cfg.networkKey.Sign(sigHash)
	if err != nil {
		return nil, err
	}
	tx.Signature = sig

	publicParams, err := tx.ToCircuitParams()
	if err != nil {
		return nil, err
	}

	privateParams := &circparams.CoinbasePrivateParams{
		{
			ScriptHash: note.ScriptHash,
			Amount:     note.Amount,
			AssetID:    note.AssetID,
			Salt:       note.Salt,
			State:      note.State,
		},
	}
	proof, err := h.prover.Prove(zk.CoinbaseValidationProgram(), privateParams, publicParams)
	if err != nil {
		return nil, err
	}
	tx.Proof = proof

	sn := &SpendableNote{
		Note:             note,
		LockingScript:    lockingScript,
		PrivateKey:       h.cfg.spendKey,
		cachedScriptHash: scriptHash,
	}
	if err := h.GenerateBlockWithTransactions([]*transactions.Transaction{transactions.WrapTransaction(tx)}, []*SpendableNote{sn}); err != nil {
		return nil, err
	}
	return sn, nil
}

// SpendableNotes returns the current list of SpendableNotes
// for the harness.
func (h *TestHarness) SpendableNotes() []*SpendableNote {
	notes := make([]*SpendableNote, 0, len(h.spendableNotes))
	for _, sn := range h.spendableNotes {
		notes = append(notes, sn)
	}
	return notes
}

// Accumulator returns the harness' accumulator
func (h *TestHarness) Accumulator() *blockchain.Accumulator {
	return h.acc
}

// Blockchain returns the harness' instance of the blockchain
func (h *TestHarness) Blockchain() *blockchain.Blockchain {
	return h.chain
}

// Clone returns a copy of the TestHarness
func (h *TestHarness) Clone() (*TestHarness, error) {
	newHarness := &TestHarness{
		acc:            h.acc.Clone(),
		spendableNotes: make(map[types.Nullifier]*SpendableNote),
		validators:     make(map[peer.ID]*validator),
		txsPerBlock:    h.txsPerBlock,
		timeSource:     h.timeSource,
		verifier:       h.verifier,
		prover:         h.prover,
		cfg: &config{
			networkKey: h.cfg.networkKey,
			spendKey:   h.cfg.spendKey,
		},
	}

	chain, err := blockchain.NewBlockchain(blockchain.DefaultOptions(), blockchain.Params(h.chain.Params()), blockchain.Verifier(h.verifier))
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

func writeBlockToFile(f *os.File, blk *blocks.Block) error {
	ser, err := proto.Marshal(blk)
	if err != nil {
		return err
	}
	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(len(ser)))
	if _, err := f.Write(lenBytes); err != nil {
		return err
	}
	if _, err := f.Write(ser); err != nil {
		return err
	}
	return nil
}
