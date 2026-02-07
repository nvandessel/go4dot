package status

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/state"
	"github.com/nvandessel/go4dot/internal/stow"
)

// SyncStatus represents the sync state of an individual config.
type SyncStatus string

const (
	SyncStatusSynced       SyncStatus = "synced"
	SyncStatusDrifted      SyncStatus = "drifted"
	SyncStatusNotInstalled SyncStatus = "not_installed"
)

// ConfigStatus holds status details for a single config.
type ConfigStatus struct {
	Name         string     `json:"name"`
	IsCore       bool       `json:"is_core"`
	Status       SyncStatus `json:"status"`
	NewFiles     int        `json:"new_files,omitempty"`
	MissingFiles int        `json:"missing_files,omitempty"`
	Conflicts    int        `json:"conflicts,omitempty"`
}

// DependencyStatus holds a summary of dependency checking.
type DependencyStatus struct {
	Installed      int `json:"installed"`
	Missing        int `json:"missing"`
	VersionMissing int `json:"version_mismatch"`
	Total          int `json:"total"`
}

// PlatformInfo holds platform details for display.
type PlatformInfo struct {
	OS             string `json:"os"`
	Distro         string `json:"distro,omitempty"`
	DistroVersion  string `json:"distro_version,omitempty"`
	PackageManager string `json:"package_manager"`
	Architecture   string `json:"architecture"`
	IsWSL          bool   `json:"is_wsl,omitempty"`
}

// Overview is the full status report.
type Overview struct {
	Platform     PlatformInfo       `json:"platform"`
	DotfilesPath string             `json:"dotfiles_path"`
	ConfigCount  int                `json:"config_count"`
	Configs      []ConfigStatus     `json:"configs"`
	Dependencies DependencyStatus   `json:"dependencies"`
	LastSync     *time.Time         `json:"last_sync,omitempty"`
	Initialized  bool               `json:"initialized"`
}

// GatherOptions configures what data is collected during gathering.
type GatherOptions struct {
	// SkipDrift disables drift checking (faster but less info).
	SkipDrift bool
	// SkipDeps disables dependency checking (faster but less info).
	SkipDeps bool
}

// Gatherer collects status data from the system. It is designed for
// dependency injection so that each subsystem can be replaced during testing.
type Gatherer struct {
	PlatformDetector func() (*platform.Platform, error)
	ConfigLoader     func() (*config.Config, string, error)
	StateLoader      func() (*state.State, error)
	DriftChecker     func(cfg *config.Config, dotfilesPath string) (*stow.DriftSummary, error)
	DepsChecker      func(cfg *config.Config, p *platform.Platform) (*deps.CheckResult, error)
}

// NewGatherer creates a Gatherer with production implementations.
func NewGatherer() *Gatherer {
	return &Gatherer{
		PlatformDetector: platform.Detect,
		ConfigLoader:     config.LoadFromDiscovery,
		StateLoader:      state.Load,
		DriftChecker:     stow.FullDriftCheck,
		DepsChecker:      deps.Check,
	}
}

// Gather collects all status information into an Overview.
func (g *Gatherer) Gather(opts GatherOptions) (*Overview, error) {
	// Detect platform
	p, err := g.PlatformDetector()
	if err != nil {
		return nil, fmt.Errorf("detecting platform: %w", err)
	}

	overview := &Overview{
		Platform: PlatformInfo{
			OS:             p.OS,
			Distro:         p.Distro,
			DistroVersion:  p.DistroVersion,
			PackageManager: p.PackageManager,
			Architecture:   p.Architecture,
			IsWSL:          p.IsWSL,
		},
	}

	// Load config (non-fatal if missing)
	cfg, configPath, err := g.ConfigLoader()
	if err != nil {
		// No config found - return minimal overview
		return overview, nil
	}

	dotfilesPath := filepath.Dir(configPath)
	overview.DotfilesPath = dotfilesPath
	overview.Initialized = true

	allConfigs := cfg.GetAllConfigs()
	overview.ConfigCount = len(allConfigs)

	// Load state (non-fatal)
	st, _ := g.StateLoader()
	if st != nil && !st.LastUpdate.IsZero() {
		t := st.LastUpdate
		overview.LastSync = &t
	}

	// Build installed config lookup from state
	installedSet := make(map[string]bool)
	if st != nil {
		installedSet = st.GetInstalledConfigNames()
	}

	// Build core config lookup
	coreSet := make(map[string]bool)
	for _, c := range cfg.Configs.Core {
		coreSet[c.Name] = true
	}

	// Drift check
	var driftMap map[string]*stow.DriftResult
	if !opts.SkipDrift {
		driftSummary, driftErr := g.DriftChecker(cfg, dotfilesPath)
		if driftErr == nil && driftSummary != nil {
			driftMap = driftSummary.ResultsMap()
		}
	}

	// Build per-config status
	for _, c := range allConfigs {
		cs := ConfigStatus{
			Name:   c.Name,
			IsCore: coreSet[c.Name],
		}

		if !installedSet[c.Name] {
			cs.Status = SyncStatusNotInstalled
		} else if dr, ok := driftMap[c.Name]; ok && dr.HasDrift {
			cs.Status = SyncStatusDrifted
			cs.NewFiles = len(dr.NewFiles)
			cs.MissingFiles = len(dr.MissingFiles)
			cs.Conflicts = len(dr.ConflictFiles)
		} else {
			cs.Status = SyncStatusSynced
		}

		overview.Configs = append(overview.Configs, cs)
	}

	// Dependency check
	if !opts.SkipDeps {
		depResult, depErr := g.DepsChecker(cfg, p)
		if depErr == nil && depResult != nil {
			overview.Dependencies = summarizeDeps(depResult)
		}
	}

	return overview, nil
}

// summarizeDeps tallies dep statuses into a summary.
func summarizeDeps(r *deps.CheckResult) DependencyStatus {
	var ds DependencyStatus
	for _, checks := range [][]deps.DependencyCheck{r.Critical, r.Core, r.Optional} {
		for _, c := range checks {
			ds.Total++
			switch c.Status {
			case deps.StatusInstalled:
				ds.Installed++
			case deps.StatusMissing:
				ds.Missing++
			case deps.StatusVersionMismatch:
				ds.VersionMissing++
			}
		}
	}
	return ds
}
