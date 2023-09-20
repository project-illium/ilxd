// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package consensus

type ValidatorSetConnection interface {
	ConnectedStakePercentage() float64
}
