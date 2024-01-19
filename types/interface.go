// Copyright (c) 2024 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

import (
	"encoding/hex"
	"encoding/json"
	"strings"
)

type Serializable interface {
	Serialize() ([]byte, error)
	Deserialize(data []byte) error
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
}

type HexEncodable []byte

func (h HexEncodable) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(h))
}

func (h *HexEncodable) UnmarshalJSON(data []byte) error {
	if strings.HasPrefix(string(data), `"`) {
		data = data[1:]
	}
	if strings.HasSuffix(string(data), `"`) {
		data = data[:len(data)-1]
	}
	b, err := hex.DecodeString(string(data))
	if err != nil {
		return err
	}
	*h = b
	return nil
}
