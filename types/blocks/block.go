// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blocks

import (
	"bufio"
	"bytes"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
)

var _ types.Serializable = (*BlockHeader)(nil)
var _ types.Serializable = (*Block)(nil)

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
	m := jsonpb.Marshaler{
		Indent: "    ",
	}
	var buf bytes.Buffer
	err := m.Marshal(bufio.NewWriter(&buf), h)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (h *BlockHeader) UnmarshalJSON(data []byte) error {
	newHeader := &BlockHeader{}
	if err := jsonpb.Unmarshal(bytes.NewReader(data), newHeader); err != nil {
		return err
	}
	h = newHeader
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
		switch tx := t.GetTx().(type) {
		case *transactions.Transaction_StandardTransaction:
			for _, out := range tx.StandardTransaction.Outputs {
				outputs = append(outputs, out)
			}
		case *transactions.Transaction_CoinbaseTransaction:
			for _, out := range tx.CoinbaseTransaction.Outputs {
				outputs = append(outputs, out)
			}
		case *transactions.Transaction_MintTransaction:
			for _, out := range tx.MintTransaction.Outputs {
				outputs = append(outputs, out)
			}
		case *transactions.Transaction_TreasuryTransaction:
			for _, out := range tx.TreasuryTransaction.Outputs {
				outputs = append(outputs, out)
			}
		}
	}
	return outputs
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
	m := jsonpb.Marshaler{
		Indent: "    ",
	}
	var buf bytes.Buffer
	err := m.Marshal(bufio.NewWriter(&buf), b)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (b *Block) UnmarshalJSON(data []byte) error {
	newBlock := &Block{}
	if err := jsonpb.Unmarshal(bytes.NewReader(data), newBlock); err != nil {
		return err
	}
	b = newBlock
	return nil
}
