package types

// GenesisState - ccm state
type GenesisState struct {
	ConsensusPeers   map[string]ConsensusPeers `json:"consensus_peers" yaml:"consensus_peers"`     // Peers for each PoS chain by chain ID.
	CheckpointHashes map[string][]byte         `json:"checkpoint_hashes" yaml:"checkpoint_hashes"` // Header hash for blocks where consensus public keys is updated for PoS chain by chain ID.
}

// NewGenesisState creates a new GenesisState object
func NewGenesisState(peers map[string]ConsensusPeers, hashes map[string][]byte) GenesisState {
	return GenesisState{
		ConsensusPeers:   peers,
		CheckpointHashes: hashes,
	}
}

// DefaultGenesisState creates a default GenesisState object
func DefaultGenesisState() GenesisState {
	return GenesisState{
		ConsensusPeers:   make(map[string]ConsensusPeers),
		CheckpointHashes: make(map[string][]byte),
	}
}

// ValidateGenesis validates the provided genesis state to ensure the
// expected invariants holds.
func ValidateGenesis(data GenesisState) error {
	return nil
}
