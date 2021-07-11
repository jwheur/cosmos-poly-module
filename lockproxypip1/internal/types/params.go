/*
 * Copyright (C) 2020 The poly network Authors
 * This file is part of The poly network library.
 *
 * The  poly network  is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The  poly network  is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 * You should have received a copy of the GNU Lesser General Public License
 * along with The poly network .  If not, see <http://www.gnu.org/licenses/>.
 */

package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/x/params"
)

// Default parameter namespace
const (
	DefaultParamspace = ModuleName
)

// Parameter store keys
var (
	Version = []byte("Version")
)

// ParamKeyTable for the lockproxypip1 module
func ParamKeyTable() params.KeyTable {
	return params.NewKeyTable().RegisterParamSet(&Params{})
}

// Params - used for initializing default parameter for lockproxy1 at genesis
type Params struct {
	Version uint64 `json:"version" yaml:"version"` // current lock proxy version
}

// NewParams creates a new Params object
func NewParams() Params {
	return Params{
		Version: 1,
	}
}

// DefaultParams defines the parameters for this module
func DefaultParams() Params {
	return NewParams()
}

// Validate validates the params
func (p Params) Validate() error {
	if err := validateVersion(p.Version); err != nil {
		return err
	}
	return nil
}

func validateVersion(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("Invalid version parameter type: %T", i)
	}
	if v < 0 {
		return fmt.Errorf("Invalid negative version, got: %d", v)
	}
	return nil
}

func (p Params) String() string {
	return fmt.Sprintf(`LockProxyPip1 Params:
  Version:             %d`, p.Version,
	)
}

// ParamSetPairs - Implements params.ParamSet
func (p *Params) ParamSetPairs() params.ParamSetPairs {
	return params.ParamSetPairs{
		params.NewParamSetPair(Version, &p.Version, validateVersion),
	}
}
