// Copyright (c) 2022 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package wallet

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/bech32"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/params/hash"
	"time"
)

const ScriptHashLength = hash.HashSize

type Address interface {
	EncodeAddress() string
	String() string
}

type TimedPubkey struct {
	Pubkey   crypto.PubKey
	Timelock time.Time
}

func (t *TimedPubkey) Serialize() ([]byte, error) {
	keyBytes, err := t.Pubkey.Raw()
	if err != nil {
		return nil, err
	}
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(t.Timelock.Unix()))
	ret := make([]byte, 0, len(keyBytes)+8)
	ret = append(ret, keyBytes...)
	ret = append(ret, ts...)
	return ret, nil
}

type SpendScript struct {
	Threshold uint8
	Pubkeys   []TimedPubkey
}

func (s *SpendScript) Serialize() ([]byte, error) {
	ser := []byte{s.Threshold}
	for _, key := range s.Pubkeys {
		b, err := key.Serialize()
		if err != nil {
			return nil, err
		}
		ser = append(ser, b...)
	}
	return ser, nil
}

func (s *SpendScript) Hash() ([]byte, error) {
	ser, err := s.Serialize()
	if err != nil {
		return nil, err
	}
	return hash.HashFunc(ser), nil
}

type BasicAddress struct {
	params  *params.NetworkParams
	version byte
	hash    [32]byte
	viewKey crypto.PubKey
}

func NewBasicAddress(script SpendScript, viewKey crypto.PubKey, params *params.NetworkParams) (*BasicAddress, error) {
	_, ok := viewKey.(*Curve25519PublicKey)
	if !ok {
		return nil, errors.New("viewKey must be of type Curve25519PublicKey")
	}

	ser, err := script.Serialize()
	if err != nil {
		return nil, err
	}
	h := hash.HashFunc(ser)
	var h2 [32]byte
	copy(h2[:], h)

	return &BasicAddress{
		hash:    h2,
		viewKey: viewKey,
		version: 1,
		params:  params,
	}, nil
}

func (a *BasicAddress) ScriptHash() [32]byte {
	return a.hash
}

func (a *BasicAddress) ViewKey() crypto.PubKey {
	return a.viewKey
}

func (a *BasicAddress) EncodeAddress() string {
	keyBytes, err := crypto.MarshalPublicKey(a.viewKey)
	if err != nil {
		return ""
	}
	converted, err := bech32.ConvertBits(append(a.hash[:], keyBytes...), 8, 5, true)
	if err != nil {
		return ""
	}
	combined := make([]byte, len(converted)+1)
	combined[0] = a.version
	copy(combined[1:], converted)
	ret, err := bech32.EncodeM(a.params.AddressPrefix, combined)
	if err != nil {
		return ""
	}
	return ret
}

func (a *BasicAddress) String() string {
	return a.EncodeAddress()
}

func DecodeAddress(addr string, params *params.NetworkParams) (Address, error) {
	// Decode the bech32 encoded address.
	_, data, err := bech32.DecodeNoLimit(addr)
	if err != nil {
		return nil, err
	}

	// The first byte of the decoded address is the version, it must exist.
	if len(data) < 1 {
		return nil, fmt.Errorf("no version")
	}

	// The remaining characters of the address returned are grouped into
	// words of 5 bits. In order to restore the original address bytes,
	// we'll need to regroup into 8 bit words.
	regrouped, err := bech32.ConvertBits(data[1:], 5, 8, false)
	if err != nil {
		return nil, err
	}

	var h2 [32]byte
	copy(h2[:], regrouped[:ScriptHashLength])

	pub, err := crypto.UnmarshalPublicKey(regrouped[ScriptHashLength:])
	if err != nil {
		return nil, err
	}

	ba := BasicAddress{
		params:  params,
		version: data[0],
		hash:    h2,
		viewKey: pub,
	}

	return &ba, nil
}
