package dashboard

import (
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/doctor"
)

// makeChecks creates n doctor.Check entries for testing.
func makeChecks(n int) []doctor.Check {
	checks := make([]doctor.Check, n)
	for i := range checks {
		checks[i] = doctor.Check{
			Name:   strings.Repeat("x", 5),
			Status: doctor.StatusOK,
		}
	}
	return checks
}

// TestHealthPanel_EnsureVisible_ScrollIndicatorEdgeCase verifies that
// scrolling down past the initial visible window correctly accounts for
// the scroll-up indicator that appears once listOffset becomes non-zero.
//
// Before the fix, ensureVisible() computed the visible slot count without
// considering scroll indicators.  Moving listOffset from 0 → N introduced
// a "↑" indicator that consumed a display line, potentially leaving the
// selected item outside the visible range.
func TestHealthPanel_EnsureVisible_ScrollIndicatorEdgeCase(t *testing.T) {
	// Panel height 7 → ContentHeight = 5, baseHeight = 4 (5 − 1 for summary).
	// With 6 checks and listOffset 0 the ↓ indicator is shown, giving 3
	// visible item slots (4 − 1 for ↓).  Selecting index 3 forces a scroll
	// that introduces the ↑ indicator too.
	p := NewHealthPanel(nil, "")
	p.SetSize(40, 7)
	p.loading = false
	p.result = &doctor.CheckResult{Checks: makeChecks(6)}

	p.selectedIdx = 0
	p.listOffset = 0

	// Select item just past the initial visible window.
	p.selectedIdx = 3
	p.ensureVisible()

	// Verify selected item is within the visible range that View() will render.
	totalChecks := len(p.result.Checks)
	baseHeight := p.ContentHeight() - 1
	slots := baseHeight
	if p.listOffset > 0 {
		slots-- // ↑ indicator
	}
	endIdx := p.listOffset + slots
	if endIdx > totalChecks {
		endIdx = totalChecks
	}
	if endIdx < totalChecks {
		slots-- // ↓ indicator
		endIdx = p.listOffset + slots
		if endIdx > totalChecks {
			endIdx = totalChecks
		}
	}

	if p.selectedIdx < p.listOffset || p.selectedIdx >= endIdx {
		t.Errorf("selectedIdx %d not visible: listOffset=%d, visible range [%d, %d)",
			p.selectedIdx, p.listOffset, p.listOffset, endIdx)
	}
}

// TestHealthPanel_EnsureVisible_ScrollDownFully exercises scrolling to the
// very last item when both ↑ and ↓ indicators are needed along the way.
func TestHealthPanel_EnsureVisible_ScrollDownFully(t *testing.T) {
	p := NewHealthPanel(nil, "")
	p.SetSize(40, 7) // ContentHeight=5, baseHeight=4
	p.loading = false
	p.result = &doctor.CheckResult{Checks: makeChecks(10)}

	// Simulate scrolling one item at a time to the end.
	for i := 0; i < len(p.result.Checks); i++ {
		p.selectedIdx = i
		p.ensureVisible()

		totalChecks := len(p.result.Checks)
		baseHeight := p.ContentHeight() - 1
		slots := baseHeight
		if p.listOffset > 0 {
			slots--
		}
		endIdx := p.listOffset + slots
		if endIdx > totalChecks {
			endIdx = totalChecks
		}
		if endIdx < totalChecks {
			slots--
			endIdx = p.listOffset + slots
			if endIdx > totalChecks {
				endIdx = totalChecks
			}
		}

		if p.selectedIdx < p.listOffset || p.selectedIdx >= endIdx {
			t.Fatalf("step %d: selectedIdx %d not visible: listOffset=%d, range [%d, %d)",
				i, p.selectedIdx, p.listOffset, p.listOffset, endIdx)
		}
	}
}

// TestHealthPanel_EnsureVisible_ScrollUpRestoresIndicator ensures that
// scrolling back up after reaching the bottom keeps the selected item
// visible while correctly removing the ↑ indicator when reaching the top.
func TestHealthPanel_EnsureVisible_ScrollUpRestoresIndicator(t *testing.T) {
	p := NewHealthPanel(nil, "")
	p.SetSize(40, 7)
	p.loading = false
	p.result = &doctor.CheckResult{Checks: makeChecks(8)}

	// Scroll to bottom first.
	p.selectedIdx = len(p.result.Checks) - 1
	p.ensureVisible()

	// Then scroll back to top one by one.
	for i := p.selectedIdx; i >= 0; i-- {
		p.selectedIdx = i
		p.ensureVisible()

		if p.selectedIdx < p.listOffset {
			t.Fatalf("step up %d: selectedIdx %d < listOffset %d",
				i, p.selectedIdx, p.listOffset)
		}
	}

	// At the top, listOffset should be 0 (no ↑ indicator).
	if p.listOffset != 0 {
		t.Errorf("expected listOffset 0 at top, got %d", p.listOffset)
	}
}

// TestHealthPanel_View_ScrollIndicators verifies that the rendered view
// includes ↑ and ↓ indicators when items exist above or below the viewport.
func TestHealthPanel_View_ScrollIndicators(t *testing.T) {
	p := NewHealthPanel(nil, "")
	p.SetSize(40, 7) // ContentHeight=5, baseHeight=4
	p.loading = false
	p.focused = true
	p.result = &doctor.CheckResult{Checks: makeChecks(10)}

	// At the top: no ↑, but ↓ should show.
	p.selectedIdx = 0
	p.listOffset = 0
	view := p.View()

	if strings.Contains(view, "↑") {
		t.Error("expected no ↑ indicator at top of list")
	}
	if !strings.Contains(view, "↓") {
		t.Error("expected ↓ indicator when items exist below viewport")
	}

	// Scroll to middle: both ↑ and ↓ should show.
	p.selectedIdx = 5
	p.ensureVisible()
	view = p.View()

	if !strings.Contains(view, "↑") {
		t.Error("expected ↑ indicator in middle of list")
	}
	if !strings.Contains(view, "↓") {
		t.Error("expected ↓ indicator in middle of list")
	}

	// Scroll to bottom: ↑ but no ↓.
	p.selectedIdx = 9
	p.ensureVisible()
	view = p.View()

	if !strings.Contains(view, "↑") {
		t.Error("expected ↑ indicator at bottom of list")
	}
	if strings.Contains(view, "↓") {
		t.Error("expected no ↓ indicator at bottom of list")
	}
}

// TestHealthPanel_EnsureVisible_SmallPanel verifies the fallback behaviour
// for panels too small to display scroll indicators (baseHeight < 3).
func TestHealthPanel_EnsureVisible_SmallPanel(t *testing.T) {
	p := NewHealthPanel(nil, "")
	p.SetSize(40, 4) // ContentHeight=2, baseHeight=1
	p.loading = false
	p.result = &doctor.CheckResult{Checks: makeChecks(5)}

	for i := 0; i < 5; i++ {
		p.selectedIdx = i
		p.ensureVisible()

		if p.selectedIdx < p.listOffset {
			t.Fatalf("small panel step %d: selectedIdx %d < listOffset %d",
				i, p.selectedIdx, p.listOffset)
		}
	}
}
