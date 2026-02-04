package state

// StateManager defines the interface for managing installation state.
// This interface allows for easier testing by providing a mockable contract.
type StateManager interface {
	// Load reads the state from disk. Returns nil if no state file exists.
	Load() (*State, error)

	// Save writes the given state to disk.
	Save(s *State) error

	// Delete removes the state file.
	Delete() error

	// Exists checks if a state file exists.
	Exists() bool
}

// DefaultStateManager is the production implementation of StateManager.
type DefaultStateManager struct{}

// Load reads the state from disk.
func (m *DefaultStateManager) Load() (*State, error) {
	return Load()
}

// Save writes the given state to disk.
func (m *DefaultStateManager) Save(s *State) error {
	return s.Save()
}

// Delete removes the state file.
func (m *DefaultStateManager) Delete() error {
	return Delete()
}

// Exists checks if a state file exists.
func (m *DefaultStateManager) Exists() bool {
	return Exists()
}

// NewStateManager creates a new DefaultStateManager.
func NewStateManager() StateManager {
	return &DefaultStateManager{}
}
