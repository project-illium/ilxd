// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package types

type Serializable interface {
	Serialize() ([]byte, error)
	Deserialize(data []byte) error
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
}
