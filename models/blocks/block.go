// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blocks

import (
	"bufio"
	"bytes"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/project-illium/ilxd/models"
)

var _ models.Serializable = (*BlockHeader)(nil)
var _ models.Serializable = (*Block)(nil)

func (h *BlockHeader) ID() models.ID {
	ser, _ := h.Serialize()
	return models.NewIDFromData(ser)
}

func (h *BlockHeader) Serialize() ([]byte, error) {
	return proto.Marshal(h)
}

func (h *BlockHeader) Deserialize(data []byte) error {
	newHeader := &BlockHeader{}
	if err := proto.Unmarshal(data, newHeader); err != nil {
		return err
	}
	h = newHeader
	return nil
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

func (h *Block) ID() models.ID {
	return h.Header.ID()
}

func (b *Block) Serialize() ([]byte, error) {
	return proto.Marshal(b)
}

func (b *Block) Deserialize(data []byte) error {
	newBlock := &Block{}
	if err := proto.Unmarshal(data, newBlock); err != nil {
		return err
	}
	b = newBlock
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
