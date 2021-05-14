package types

// GenesisState - ccm state
type GenesisState struct {
	ConsensusPeers   map[uint64]ConsensusPeers `json:"consensus_peers" yaml:"consensus_peers"`     // Peers for each PoS chain by chain ID.
	CheckpointHashes map[uint64][]byte         `json:"checkpoint_hashes" yaml:"checkpoint_hashes"` // Header hash for blocks where consensus public keys is updated for PoS chain by chain ID.
}

// NewGenesisState creates a new GenesisState object
func NewGenesisState(peers map[uint64]ConsensusPeers, hashes map[uint64][]byte) GenesisState {
	return GenesisState{
		ConsensusPeers:   peers,
		CheckpointHashes: hashes,
	}
}

// DefaultGenesisState creates a default GenesisState object
func DefaultGenesisState() GenesisState {
	return GenesisState{
		ConsensusPeers:   make(map[uint64]ConsensusPeers),
		CheckpointHashes: make(map[uint64][]byte),
	}
}

// ValidateGenesis validates the provided genesis state to ensure the
// expected invariants holds.
func ValidateGenesis(data GenesisState) error {
	return nil
}
