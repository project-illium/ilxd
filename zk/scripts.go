// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package zk

import (
	"embed"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/zk/lurk/macros"
)

//go:embed lurk/basic_transfer.lurk
var basicTransferScriptLurk embed.FS
var basicTransferScriptData string
var basicTransferScriptCommitment []byte

//go:embed lurk/password_script.lurk
var passwordScriptLurk embed.FS
var passwordScriptData string

//go:embed lurk/multisig_script.lurk
var multisigScriptLurk embed.FS
var multisigScriptData string
var multisigCommitment []byte

//go:embed lurk/timelocked_multisig.lurk
var timelockedMultisigScriptLurk embed.FS
var timelockedMultisigScriptData string
var timeLockedMultisigCommitment []byte

//go:embed lurk/public_address_script.lurk
var publicAddressScriptLurk embed.FS
var publicAddressScriptData string
var publicAddressScriptCommitment []byte

//go:embed lurk/standard_validation.lurk
var standardValidationScriptLurk embed.FS
var standardValidationScriptData string

//go:embed lurk/mint_validation.lurk
var mintValidationScriptLurk embed.FS
var mintValidationScriptData string

//go:embed lurk/coinbase_validation.lurk
var coinbaseValidationScriptLurk embed.FS
var coinbaseValidationScriptData string

//go:embed lurk/stake_validation.lurk
var stakeValidationScriptLurk embed.FS
var stakeValidationScriptData string

func init() {
	mp, err := macros.NewMacroPreprocessor(macros.WithStandardLib(), macros.RemoveComments())
	if err != nil {
		panic(err)
	}
	data, err := basicTransferScriptLurk.ReadFile("lurk/basic_transfer.lurk")
	if err != nil {
		panic(err)
	}
	basicTransferScriptData, err = mp.Preprocess(string(data))
	if err != nil {
		panic(err)
	}
	basicTransferScriptCommitment, err = LurkCommit(basicTransferScriptData)
	if err != nil {
		panic(err)
	}

	data, err = passwordScriptLurk.ReadFile("lurk/password_script.lurk")
	if err != nil {
		panic(err)
	}
	passwordScriptData, err = mp.Preprocess(string(data))
	if err != nil {
		panic(err)
	}

	data, err = multisigScriptLurk.ReadFile("lurk/multisig_script.lurk")
	if err != nil {
		panic(err)
	}
	multisigScriptData, err = mp.Preprocess(string(data))
	if err != nil {
		panic(err)
	}
	multisigCommitment, err = LurkCommit(multisigScriptData)
	if err != nil {
		panic(err)
	}

	data, err = timelockedMultisigScriptLurk.ReadFile("lurk/timelocked_multisig.lurk")
	if err != nil {
		panic(err)
	}
	timelockedMultisigScriptData, err = mp.Preprocess(string(data))
	if err != nil {
		panic(err)
	}
	timeLockedMultisigCommitment, err = LurkCommit(timelockedMultisigScriptData)
	if err != nil {
		panic(err)
	}

	data, err = publicAddressScriptLurk.ReadFile("lurk/public_address_script.lurk")
	if err != nil {
		panic(err)
	}
	publicAddressScriptData, err = mp.Preprocess(string(data))
	if err != nil {
		panic(err)
	}
	publicAddressScriptCommitment, err = LurkCommit(publicAddressScriptData)
	if err != nil {
		panic(err)
	}

	data, err = standardValidationScriptLurk.ReadFile("lurk/standard_validation.lurk")
	if err != nil {
		panic(err)
	}

	standardValidationScriptData = string(data)
	data, err = mintValidationScriptLurk.ReadFile("lurk/mint_validation.lurk")
	if err != nil {
		panic(err)
	}

	mintValidationScriptData = string(data)

	data, err = coinbaseValidationScriptLurk.ReadFile("lurk/coinbase_validation.lurk")
	if err != nil {
		panic(err)
	}

	coinbaseValidationScriptData = string(data)

	data, err = stakeValidationScriptLurk.ReadFile("lurk/stake_validation.lurk")
	if err != nil {
		panic(err)
	}

	stakeValidationScriptData = string(data)
}

// BasicTransferScript returns the basic transfer lurk script
func BasicTransferScript() string {
	return basicTransferScriptData
}

// BasicTransferScriptCommitment returns the script commitment hash
// for the basic transfer script.
func BasicTransferScriptCommitment() []byte {
	ret := make([]byte, len(basicTransferScriptCommitment))
	copy(ret, basicTransferScriptCommitment)
	return ret
}

// PasswordScript returns the password lurk script
func PasswordScript() string {
	return passwordScriptData
}

// MultisigScript returns the multsig lurk script
func MultisigScript() string {
	return multisigScriptData
}

// MultisigScriptCommitment returns the script commitment hash
// for the multisig script.
func MultisigScriptCommitment() []byte {
	ret := make([]byte, len(multisigCommitment))
	copy(ret, multisigCommitment)
	return ret
}

// TimelockedMultisigScript returns the timelocked multisig lurk script
func TimelockedMultisigScript() string {
	return timelockedMultisigScriptData
}

// TimelockedMultisigScriptCommitment returns the script commitment hash
// for the timelocked multisig script.
func TimelockedMultisigScriptCommitment() []byte {
	ret := make([]byte, len(timeLockedMultisigCommitment))
	copy(ret, timeLockedMultisigCommitment)
	return ret
}

// PublicAddressScript returns the public address lurk script
func PublicAddressScript() string {
	return publicAddressScriptData
}

// PublicAddressScriptCommitment returns the script commitment hash
// for the public address script.
func PublicAddressScriptCommitment() []byte {
	ret := make([]byte, len(publicAddressScriptCommitment))
	copy(ret, publicAddressScriptCommitment)
	return ret
}

// StandardValidationProgram returns the standard validation lurk program script
func StandardValidationProgram() string {
	return standardValidationScriptData
}

// MintValidationProgram returns the mint validation lurk program script
func MintValidationProgram() string {
	return mintValidationScriptData
}

// CoinbaseValidationProgram returns the coinbase validation lurk program script
func CoinbaseValidationProgram() string {
	return coinbaseValidationScriptData
}

// TreasuryValidationProgram is an alias for the coinbase validation program
// as they use the same script.
func TreasuryValidationProgram() string {
	return coinbaseValidationScriptData
}

// StakeValidationProgram returns the stake validation lurk program script
func StakeValidationProgram() string {
	return stakeValidationScriptData
}

// MakeMultisigUnlockingParams takes in a list of public keys and signatures and returns
// a lurk expression for the signatures that can be used as unlocking params for the
// multisig script.
//
// Note: The public keys here must be in the same order as they appear in the locking params.
// Note2: The signatures must be in the same order as the public keys they belong to. // Fixme: this function can be refactored to remove this requirement
func MakeMultisigUnlockingParams(pubkeys []crypto.PubKey, sigs [][]byte, sigHash []byte) (string, error) {
	sigCpy := make([][]byte, len(sigs))
	for i, sig := range sigs {
		sigCpy[i] = make([]byte, len(sig))
		copy(sigCpy[i], sig)
	}

	keySelector := "(cons "
	for i, key := range pubkeys {
		if len(sigCpy) > 0 {
			valid, err := key.Verify(sigHash, sigCpy[0])
			if err != nil {
				return "", err
			}
			if valid {
				keySelector += "1 "
				sigCpy = sigCpy[1:]
			} else {
				keySelector += "0 "
			}
		} else {
			keySelector += "0 "
		}
		if i == len(pubkeys)-1 {
			keySelector += "nil"
		} else {
			keySelector += "(cons "
		}
	}
	for i := 0; i < len(pubkeys); i++ {
		keySelector += ")"
	}

	unlockignScript := "(cons " + keySelector + " "
	for _, sig := range sigs {
		unlockignScript += "(cons "
		if len(sig) != 64 {
			return "", errors.New("invalid signature len")
		}
		sigRx, sigRy, sigS := icrypto.UnmarshalSignature(sig)
		unlockignScript += fmt.Sprintf("(cons 0x%x (cons 0x%x (cons 0x%x nil))) ", sigRx, sigRy, sigS)
	}
	unlockignScript += "nil)"
	for i := 0; i < len(sigs); i++ {
		unlockignScript += ")"
	}
	return unlockignScript, nil
}
