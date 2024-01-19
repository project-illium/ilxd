// Copyright (c) 2024 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blocks

import (
	"encoding/json"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var _ types.Serializable = (*BlockHeader)(nil)
var _ types.Serializable = (*Block)(nil)

type headerJSON struct {
	Version     uint32             `json:"version"`
	Height      uint32             `json:"height"`
	Parent      types.HexEncodable `json:"parent"`
	Timestamp   int64              `json:"timestamp"`
	TxRoot      types.HexEncodable `json:"tx_root"`
	Producer_ID types.HexEncodable `json:"producer_ID"`
	Signature   types.HexEncodable `json:"signature"`
}

type blockJSON struct {
	Header       headerJSON                  `json:"header"`
	Transactions []*transactions.Transaction `json:"transactions"`
}

func (h *BlockHeader) ID() types.ID {
	ser, _ := h.Serialize()
	return types.NewIDFromData(ser)
}

func (h *BlockHeader) Serialize() ([]byte, error) {
	return proto.Marshal(h)
}

func (h *BlockHeader) SerializedSize() (int, error) {
	ser, err := proto.Marshal(h)
	if err != nil {
		return 0, err
	}
	return len(ser), nil
}

func (h *BlockHeader) Deserialize(data []byte) error {
	newHeader := BlockHeader{}
	if err := proto.Unmarshal(data, &newHeader); err != nil {
		return err
	}
	h.Timestamp = newHeader.Timestamp
	h.Height = newHeader.Height
	h.Signature = newHeader.Signature
	h.Producer_ID = newHeader.Producer_ID
	h.TxRoot = newHeader.TxRoot
	h.Parent = newHeader.Parent
	h.Version = newHeader.Version
	return nil
}

func (h *BlockHeader) SigHash() ([]byte, error) {
	cpy := proto.Clone(h)
	cpy.(*BlockHeader).Signature = nil

	b, err := proto.Marshal(cpy)
	if err != nil {
		return nil, err
	}

	return hash.HashFunc(b), nil
}

func (h *BlockHeader) MarshalJSON() ([]byte, error) {
	header := &headerJSON{
		Version:     h.Version,
		Height:      h.Height,
		Parent:      h.Parent,
		Timestamp:   h.Timestamp,
		TxRoot:      h.TxRoot,
		Producer_ID: h.Producer_ID,
		Signature:   h.Signature,
	}

	return json.Marshal(header)
}

func (h *BlockHeader) UnmarshalJSON(data []byte) error {
	newHeader := &headerJSON{}
	if err := json.Unmarshal(data, newHeader); err != nil {
		return err
	}
	*h = BlockHeader{
		Version:     newHeader.Version,
		Height:      newHeader.Height,
		Parent:      newHeader.Parent,
		Timestamp:   newHeader.Timestamp,
		TxRoot:      newHeader.TxRoot,
		Producer_ID: newHeader.Producer_ID,
		Signature:   newHeader.Signature,
	}
	return nil
}

func (b *Block) ID() types.ID {
	return b.Header.ID()
}

func (b *Block) Nullifiers() []types.Nullifier {
	nullifiers := make([]types.Nullifier, 0, len(b.Transactions))
	for _, t := range b.Transactions {
		switch tx := t.GetTx().(type) {
		case *transactions.Transaction_StandardTransaction:
			for _, n := range tx.StandardTransaction.Nullifiers {
				nullifiers = append(nullifiers, types.NewNullifier(n))
			}
		case *transactions.Transaction_MintTransaction:
			for _, n := range tx.MintTransaction.Nullifiers {
				nullifiers = append(nullifiers, types.NewNullifier(n))
			}

		}
	}
	return nullifiers
}

func (b *Block) Outputs() []*transactions.Output {
	outputs := make([]*transactions.Output, 0, len(b.Transactions))
	for _, t := range b.Transactions {
		outputs = append(outputs, t.Outputs()...)
	}
	return outputs
}

func (b *Block) Txids() []types.ID {
	txids := make([]types.ID, 0, len(b.Transactions))
	for _, t := range b.Transactions {
		txids = append(txids, t.ID())
	}
	return txids
}

func (b *Block) Serialize() ([]byte, error) {
	return proto.Marshal(b)
}

func (b *Block) SerializedSize() (int, error) {
	ser, err := proto.Marshal(b)
	if err != nil {
		return 0, err
	}
	return len(ser), nil
}

func (b *Block) Deserialize(data []byte) error {
	newBlock := &Block{}
	if err := proto.Unmarshal(data, newBlock); err != nil {
		return err
	}
	b.Header = newBlock.Header
	b.Transactions = newBlock.Transactions
	return nil
}

func (b *Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(*b)
}

func (b *Block) UnmarshalJSON(data []byte) error {
	newBlock := blockJSON{}
	if err := json.Unmarshal(data, &newBlock); err != nil {
		return err
	}
	*b = Block{
		Header: &BlockHeader{
			Version:     newBlock.Header.Version,
			Height:      newBlock.Header.Height,
			Parent:      newBlock.Header.Parent,
			Timestamp:   newBlock.Header.Timestamp,
			TxRoot:      newBlock.Header.TxRoot,
			Producer_ID: newBlock.Header.Producer_ID,
			Signature:   newBlock.Header.Signature,
		},
		Transactions: newBlock.Transactions,
	}
	return nil
}

func (b *XThinnerBlock) ID() types.ID {
	return b.Header.ID()
}

func (b *XThinnerBlock) Serialize() ([]byte, error) {
	return proto.Marshal(b)
}

func (b *XThinnerBlock) SerializedSize() (int, error) {
	ser, err := proto.Marshal(b)
	if err != nil {
		return 0, err
	}
	return len(ser), nil
}

func (b *XThinnerBlock) Deserialize(data []byte) error {
	newBlock := &XThinnerBlock{}
	if err := proto.Unmarshal(data, newBlock); err != nil {
		return err
	}
	b.Header = newBlock.Header
	b.TxCount = newBlock.TxCount
	b.PrefilledTxs = newBlock.PrefilledTxs
	b.Pushes = newBlock.Pushes
	b.Pops = newBlock.Pops
	b.PushBytes = newBlock.PushBytes
	return nil
}

func (b *XThinnerBlock) MarshalJSON() ([]byte, error) {
	m := protojson.MarshalOptions{
		Indent: "    ",
	}
	buf, err := m.Marshal(b)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func (b *XThinnerBlock) UnmarshalJSON(data []byte) error {
	newBlock := &XThinnerBlock{}
	if err := protojson.Unmarshal(data, newBlock); err != nil {
		return err
	}
	*b = XThinnerBlock{
		Header: &BlockHeader{
			Version:     newBlock.Header.Version,
			Height:      newBlock.Header.Height,
			Parent:      newBlock.Header.Parent,
			Timestamp:   newBlock.Header.Timestamp,
			TxRoot:      newBlock.Header.TxRoot,
			Producer_ID: newBlock.Header.Producer_ID,
			Signature:   newBlock.Header.Signature,
		},
		TxCount:      newBlock.TxCount,
		Pops:         newBlock.Pops,
		Pushes:       newBlock.Pushes,
		PushBytes:    newBlock.PushBytes,
		PrefilledTxs: newBlock.PrefilledTxs,
	}
	return nil
}

type compressedTransactionJSON struct {
	Txid       types.HexEncodable     `json:"txid"`
	Nullifiers []types.HexEncodable   `json:"nullifiers"`
	Outputs    []*transactions.Output `json:"outputs"`
}

type compressedBlockJSON struct {
	Height       uint32 `json:"height"`
	Transactions []compressedTransactionJSON
}

func (b *CompressedBlock) SerializedSize() (int, error) {
	ser, err := proto.Marshal(b)
	if err != nil {
		return 0, err
	}
	return len(ser), nil
}

func (b *CompressedBlock) Deserialize(data []byte) error {
	newBlock := &CompressedBlock{}
	if err := proto.Unmarshal(data, newBlock); err != nil {
		return err
	}
	b.Height = newBlock.Height
	b.Txs = newBlock.Txs
	return nil
}

func (b *CompressedBlock) MarshalJSON() ([]byte, error) {
	s := &compressedBlockJSON{
		Height:       b.Height,
		Transactions: make([]compressedTransactionJSON, 0, len(b.Txs)),
	}
	for _, tx := range b.Txs {
		nullifiers := make([]types.HexEncodable, 0, len(tx.Nullifiers))
		for _, n := range tx.Nullifiers {
			nullifiers = append(nullifiers, n)
		}
		s.Transactions = append(s.Transactions, compressedTransactionJSON{
			Txid:       tx.Txid,
			Nullifiers: nullifiers,
			Outputs:    tx.Outputs,
		})
	}
	return json.Marshal(s)
}

func (b *CompressedBlock) UnmarshalJSON(data []byte) error {
	newBlock := &compressedBlockJSON{}
	if err := json.Unmarshal(data, newBlock); err != nil {
		return err
	}

	*b = CompressedBlock{
		Height: newBlock.Height,
		Txs:    make([]*CompressedBlock_CompressedTx, 0, len(newBlock.Transactions)),
	}
	for _, tx := range newBlock.Transactions {
		nullifiers := make([][]byte, 0, len(tx.Nullifiers))
		for _, n := range tx.Nullifiers {
			nullifiers = append(nullifiers, n)
		}
		b.Txs = append(b.Txs, &CompressedBlock_CompressedTx{
			Txid:       tx.Txid,
			Nullifiers: nullifiers,
			Outputs:    tx.Outputs,
		})
	}
	return nil
}
