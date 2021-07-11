package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GenesisState - ccm state
type GenesisState struct {
	Params     Params             `json:"params" yaml:"params"`         // For storing module version.
	Nonce      sdk.Int            `json:"nonce" yaml:"nonce"`           // An auto-incrementing nonce for withdrawals.
	ChainIDs   map[string][]byte  `json:"chain_ids" yaml:"chain_ids"`   // Records chainIDs (value is always []byte("1") if exists)
	Registries map[string][]byte  `json:"registries" yaml:"registries"` // Records registries (value is always []byte("1") if exists)
	Operators  map[string][]byte  `json:"operators" yaml:"operators"`   // Records operators (value is operator address as bytes)
	Balances   map[string]sdk.Int `json:"balances" yaml:"balances"`     // Records balances (deprecated)
}

// NewGenesisState creates a new GenesisState object
func NewGenesisState(params Params, nonce sdk.Int, chainIDs, registries, operators map[string][]byte, balances map[string]sdk.Int) GenesisState {
	return GenesisState{
		Params:     params,
		Nonce:      nonce,
		ChainIDs:   chainIDs,
		Registries: registries,
		Operators:  operators,
		Balances:   balances,
	}
}

// DefaultGenesisState creates a default GenesisState object
func DefaultGenesisState() GenesisState {
	return GenesisState{
		Params:     DefaultParams(),
		Nonce:      sdk.ZeroInt(),
		ChainIDs:   make(map[string][]byte),
		Registries: make(map[string][]byte),
		Operators:  make(map[string][]byte),
		Balances:   make(map[string]sdk.Int),
	}
}

// ValidateGenesis validates the provided genesis state to ensure the
// expected invariants holds.
func ValidateGenesis(data GenesisState) error {
	if err := data.Params.Validate(); err != nil {
		return err
	}

	return nil
}
