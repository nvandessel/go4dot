package dashboard

import (
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/doctor"
)

func newTestHealthPanel(checks []doctor.Check) *HealthPanel {
	p := NewHealthPanel(nil, "")
	p.loading = false
	p.result = &doctor.CheckResult{Checks: checks}
	return p
}

func TestHealthPanel_GetListVisibleHeight(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		height   int
		expected int
	}{
		{"normal panel size", 40, 20, 16},
		{"small panel", 10, 5, 1},
		{"very small panel clamps to 1", 10, 3, 1},
		{"minimum height", 10, 4, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewHealthPanel(nil, "")
			p.SetSize(tt.width, tt.height)
			got := p.getListVisibleHeight()
			if got != tt.expected {
				t.Errorf("getListVisibleHeight() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestHealthPanel_RenderSummary(t *testing.T) {
	tests := []struct {
		name           string
		checks         []doctor.Check
		expectContains []string
		expectAbsent   []string
	}{
		{
			name: "errors warnings and ok",
			checks: []doctor.Check{
				{Name: "a", Status: doctor.StatusError},
				{Name: "b", Status: doctor.StatusWarning},
				{Name: "c", Status: doctor.StatusOK},
			},
			expectContains: []string{"1 err", "1 warn", "1 ok"},
		},
		{
			name: "only ok",
			checks: []doctor.Check{
				{Name: "a", Status: doctor.StatusOK},
				{Name: "b", Status: doctor.StatusOK},
			},
			expectContains: []string{"2 ok"},
			expectAbsent:   []string{"err", "warn"},
		},
		{
			name: "only errors",
			checks: []doctor.Check{
				{Name: "a", Status: doctor.StatusError},
			},
			expectContains: []string{"1 err"},
			expectAbsent:   []string{"ok", "warn"},
		},
		{
			name:           "no checks renders fallback",
			checks:         []doctor.Check{},
			expectContains: []string{"No checks"},
		},
		{
			name: "double-space separator",
			checks: []doctor.Check{
				{Name: "a", Status: doctor.StatusError},
				{Name: "b", Status: doctor.StatusOK},
			},
			expectContains: []string{"  "},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestHealthPanel(tt.checks)
			p.SetSize(80, 20)
			summary := p.renderSummary()
			for _, s := range tt.expectContains {
				if !strings.Contains(summary, s) {
					t.Errorf("renderSummary() should contain %q, got %q", s, summary)
				}
			}
			for _, s := range tt.expectAbsent {
				if strings.Contains(summary, s) {
					t.Errorf("renderSummary() should not contain %q, got %q", s, summary)
				}
			}
		})
	}
}

func TestHealthPanel_RenderCheckItems_ASCIIIcons(t *testing.T) {
	checks := []doctor.Check{
		{Name: "Platform", Status: doctor.StatusOK},
		{Name: "Stow", Status: doctor.StatusWarning},
		{Name: "Git", Status: doctor.StatusError},
		{Name: "Deps", Status: doctor.StatusSkipped},
	}
	p := newTestHealthPanel(checks)
	p.SetSize(60, 20)
	items := p.renderCheckItems()
	joined := strings.Join(items, "\n")
	for _, icon := range []string{iconOK, iconWarning, iconError, iconSkipped} {
		if !strings.Contains(joined, icon) {
			t.Errorf("expected ASCII icon %q in output", icon)
		}
	}
}

func TestHealthPanel_RenderCheckItems_NoScrollIndicators(t *testing.T) {
	checks := []doctor.Check{
		{Name: "Platform", Status: doctor.StatusOK},
		{Name: "Git", Status: doctor.StatusOK},
	}
	p := newTestHealthPanel(checks)
	p.SetSize(60, 20)
	items := p.renderCheckItems()
	joined := strings.Join(items, "\n")
	if strings.Contains(joined, "^^") {
		t.Error("unexpected scroll-up indicator")
	}
	if strings.Contains(joined, "vv") {
		t.Error("unexpected scroll-down indicator")
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestHealthPanel_RenderCheckItems_ScrollDownIndicator(t *testing.T) {
	var checks []doctor.Check
	for i := 0; i < 10; i++ {
		checks = append(checks, doctor.Check{Name: "Check" + string(rune('A'+i)), Status: doctor.StatusOK})
	}
	p := newTestHealthPanel(checks)
	p.SetSize(60, 6)
	p.listOffset = 0
	items := p.renderCheckItems()
	joined := strings.Join(items, "\n")
	if strings.Contains(joined, "^^") {
		t.Error("unexpected scroll-up indicator at top")
	}
	if !strings.Contains(joined, "vv") {
		t.Error("expected scroll-down indicator")
	}
}

func TestHealthPanel_RenderCheckItems_ScrollUpIndicator(t *testing.T) {
	var checks []doctor.Check
	for i := 0; i < 10; i++ {
		checks = append(checks, doctor.Check{Name: "Check" + string(rune('A'+i)), Status: doctor.StatusOK})
	}
	p := newTestHealthPanel(checks)
	p.SetSize(60, 6)
	p.selectedIdx = 9
	p.listOffset = 8
	items := p.renderCheckItems()
	joined := strings.Join(items, "\n")
	if !strings.Contains(joined, "^^") {
		t.Error("expected scroll-up indicator")
	}
	if !strings.Contains(joined, "8 more") {
		t.Errorf("expected '8 more', got %q", joined)
	}
}

func TestHealthPanel_RenderCheckItems_BothScrollIndicators(t *testing.T) {
	var checks []doctor.Check
	for i := 0; i < 20; i++ {
		checks = append(checks, doctor.Check{Name: "Check" + string(rune('A'+i)), Status: doctor.StatusOK})
	}
	p := newTestHealthPanel(checks)
	p.SetSize(60, 10)
	p.listOffset = 5
	p.selectedIdx = 5
	items := p.renderCheckItems()
	joined := strings.Join(items, "\n")
	if !strings.Contains(joined, "^^") {
		t.Error("expected scroll-up indicator")
	}
	if !strings.Contains(joined, "vv") {
		t.Error("expected scroll-down indicator")
	}
}

func TestHealthPanel_View_LoadingState(t *testing.T) {
	p := NewHealthPanel(nil, "")
	p.SetSize(60, 20)
	view := p.View()
	if !strings.Contains(view, "Checking...") {
		t.Errorf("expected loading message, got %q", view)
	}
}

func TestHealthPanel_View_ErrorState(t *testing.T) {
	p := NewHealthPanel(nil, "")
	p.loading = false
	p.lastError = &testError{msg: "something broke"}
	p.SetSize(60, 20)
	view := p.View()
	if !strings.Contains(view, "something broke") {
		t.Errorf("expected error message, got %q", view)
	}
}

func TestHealthPanel_View_NilResult(t *testing.T) {
	p := NewHealthPanel(nil, "")
	p.loading = false
	p.SetSize(60, 20)
	view := p.View()
	if !strings.Contains(view, "No results") {
		t.Errorf("expected 'No results', got %q", view)
	}
}

func TestHealthPanel_View_SmallDimensions(t *testing.T) {
	p := NewHealthPanel(nil, "")
	p.SetSize(3, 2)
	if p.View() != "" {
		t.Error("expected empty view for small dimensions")
	}
}

func TestHealthPanel_View_ContainsSummaryAndItems(t *testing.T) {
	checks := []doctor.Check{
		{Name: "Platform", Status: doctor.StatusOK},
		{Name: "Git", Status: doctor.StatusError},
	}
	p := newTestHealthPanel(checks)
	p.SetSize(60, 20)
	view := p.View()
	for _, s := range []string{"1 ok", "1 err", "Platform", "Git"} {
		if !strings.Contains(view, s) {
			t.Errorf("view should contain %q", s)
		}
	}
}

func TestHealthPanel_MoveDown(t *testing.T) {
	checks := []doctor.Check{
		{Name: "A", Status: doctor.StatusOK},
		{Name: "B", Status: doctor.StatusOK},
		{Name: "C", Status: doctor.StatusOK},
	}
	p := newTestHealthPanel(checks)
	p.SetSize(60, 20)
	p.moveDown()
	if p.selectedIdx != 1 {
		t.Errorf("expected 1, got %d", p.selectedIdx)
	}
	p.moveDown()
	if p.selectedIdx != 2 {
		t.Errorf("expected 2, got %d", p.selectedIdx)
	}
	p.moveDown()
	if p.selectedIdx != 2 {
		t.Errorf("expected 2 (clamped), got %d", p.selectedIdx)
	}
}

func TestHealthPanel_MoveUp(t *testing.T) {
	checks := []doctor.Check{
		{Name: "A", Status: doctor.StatusOK},
		{Name: "B", Status: doctor.StatusOK},
	}
	p := newTestHealthPanel(checks)
	p.SetSize(60, 20)
	p.selectedIdx = 1
	p.moveUp()
	if p.selectedIdx != 0 {
		t.Errorf("expected 0, got %d", p.selectedIdx)
	}
	p.moveUp()
	if p.selectedIdx != 0 {
		t.Errorf("expected 0 (clamped), got %d", p.selectedIdx)
	}
}

func TestHealthPanel_MoveDown_NilResult(t *testing.T) {
	p := NewHealthPanel(nil, "")
	p.loading = false
	p.SetSize(60, 20)
	p.moveDown()
	if p.selectedIdx != 0 {
		t.Errorf("expected 0, got %d", p.selectedIdx)
	}
}

func TestHealthPanel_EnsureVisible_ScrollsDown(t *testing.T) {
	var checks []doctor.Check
	for i := 0; i < 10; i++ {
		checks = append(checks, doctor.Check{Name: "Check" + string(rune('A'+i)), Status: doctor.StatusOK})
	}
	p := newTestHealthPanel(checks)
	p.SetSize(60, 6)
	p.listOffset = 0
	p.selectedIdx = 5
	p.ensureVisible()
	if p.listOffset == 0 {
		t.Error("expected listOffset to scroll down")
	}
}

func TestHealthPanel_EnsureVisible_ScrollsUp(t *testing.T) {
	var checks []doctor.Check
	for i := 0; i < 10; i++ {
		checks = append(checks, doctor.Check{Name: "Check" + string(rune('A'+i)), Status: doctor.StatusOK})
	}
	p := newTestHealthPanel(checks)
	p.SetSize(60, 6)
	p.listOffset = 5
	p.selectedIdx = 2
	p.ensureVisible()
	if p.listOffset > 2 {
		t.Errorf("expected listOffset <= 2, got %d", p.listOffset)
	}
}

func TestHealthPanel_EnsureVisible_AccountsForScrollDownIndicator(t *testing.T) {
	// With 10 checks and visibleHeight=2, selecting item 1 from offset 0
	// means renderCheckItems will show a scroll-down indicator (eating 1 slot).
	// Only 1 item slot remains, so the offset must advance to keep item 1 visible.
	var checks []doctor.Check
	for i := 0; i < 10; i++ {
		checks = append(checks, doctor.Check{Name: "Check" + string(rune('A'+i)), Status: doctor.StatusOK})
	}
	p := newTestHealthPanel(checks)
	p.SetSize(60, 6) // ContentHeight=4, getListVisibleHeight=2
	p.listOffset = 0
	p.selectedIdx = 1
	p.ensureVisible()
	// Without the fix, listOffset stays 0 and the scroll-down indicator
	// hides item 1. With the fix, offset adjusts so item 1 is rendered.
	if p.listOffset < 1 {
		t.Errorf("expected listOffset >= 1, got %d (scroll-down indicator not accounted for)", p.listOffset)
	}
}

func TestHealthPanel_EnsureVisible_AccountsForBothScrollIndicators(t *testing.T) {
	// With 10 checks, visibleHeight=2, offset=3, selecting item 4:
	// Both scroll-up and scroll-down indicators are shown, leaving only 1 item slot.
	// The offset must advance past 3 to keep item 4 visible.
	var checks []doctor.Check
	for i := 0; i < 10; i++ {
		checks = append(checks, doctor.Check{Name: "Check" + string(rune('A'+i)), Status: doctor.StatusOK})
	}
	p := newTestHealthPanel(checks)
	p.SetSize(60, 6) // ContentHeight=4, getListVisibleHeight=2
	p.listOffset = 3
	p.selectedIdx = 4
	p.ensureVisible()
	// Both indicators eat a line each. Only 1 item slot remains from visibleHeight=2.
	// Item 4 must be in the rendered range [listOffset, listOffset+1).
	if p.selectedIdx < p.listOffset || p.selectedIdx >= p.listOffset+1 {
		t.Errorf("selected item %d not in visible range [%d, %d)",
			p.selectedIdx, p.listOffset, p.listOffset+1)
	}
}

func TestHealthPanel_EnsureVisible_NoIndicatorsNoAdjustment(t *testing.T) {
	// When all items fit without scrolling, no indicators are shown,
	// and ensureVisible should not change the offset.
	checks := []doctor.Check{
		{Name: "A", Status: doctor.StatusOK},
		{Name: "B", Status: doctor.StatusOK},
	}
	p := newTestHealthPanel(checks)
	p.SetSize(60, 20) // Plenty of space
	p.listOffset = 0
	p.selectedIdx = 1
	p.ensureVisible()
	if p.listOffset != 0 {
		t.Errorf("expected listOffset 0, got %d", p.listOffset)
	}
}

func TestHealthPanel_EnsureVisible_ScrollUpAccountsForIndicator(t *testing.T) {
	// When scrolling up, the scroll-up indicator at the new offset should be
	// accounted for. With offset=1 and selectedIdx=1, the scroll-up indicator
	// is shown (offset>0), eating 1 item slot from visibleHeight=2. Only 1 slot
	// remains, but item 1 is at offset 1, so it's still the first rendered item.
	var checks []doctor.Check
	for i := 0; i < 10; i++ {
		checks = append(checks, doctor.Check{Name: "Check" + string(rune('A'+i)), Status: doctor.StatusOK})
	}
	p := newTestHealthPanel(checks)
	p.SetSize(60, 6) // ContentHeight=4, getListVisibleHeight=2
	p.listOffset = 5
	p.selectedIdx = 1
	p.ensureVisible()
	// After scrolling up, selectedIdx=1 must be within the rendered range.
	// The rendered range starts at listOffset.
	if p.selectedIdx < p.listOffset {
		t.Errorf("selected item %d is above listOffset %d", p.selectedIdx, p.listOffset)
	}
}

func TestHealthPanel_GetSelectedItem(t *testing.T) {
	checks := []doctor.Check{
		{Name: "Platform", Status: doctor.StatusOK},
		{Name: "Git", Status: doctor.StatusOK},
	}
	p := newTestHealthPanel(checks)
	p.selectedIdx = 1
	item := p.GetSelectedItem()
	if item == nil {
		t.Fatal("expected non-nil SelectedItem")
	}
	if item.Name != "Git" {
		t.Errorf("expected 'Git', got %q", item.Name)
	}
}

func TestHealthPanel_GetSelectedItem_NilResult(t *testing.T) {
	p := NewHealthPanel(nil, "")
	p.loading = false
	if p.GetSelectedItem() != nil {
		t.Error("expected nil SelectedItem")
	}
}

func TestHealthPanel_GetSelectedCheck(t *testing.T) {
	checks := []doctor.Check{
		{Name: "Platform", Status: doctor.StatusOK, Message: "linux"},
	}
	p := newTestHealthPanel(checks)
	check := p.GetSelectedCheck()
	if check == nil || check.Name != "Platform" {
		t.Error("expected Platform check")
	}
}

func TestHealthPanel_IsLoading(t *testing.T) {
	p := NewHealthPanel(nil, "")
	if !p.IsLoading() {
		t.Error("expected true")
	}
	p.loading = false
	if p.IsLoading() {
		t.Error("expected false")
	}
}

func TestHealthPanel_GetResult(t *testing.T) {
	result := &doctor.CheckResult{Checks: []doctor.Check{{Name: "test", Status: doctor.StatusOK}}}
	p := NewHealthPanel(nil, "")
	p.loading = false
	p.result = result
	if p.GetResult() != result {
		t.Error("expected stored result")
	}
}

func TestHealthPanel_NameTruncation(t *testing.T) {
	checks := []doctor.Check{{Name: strings.Repeat("A", 100), Status: doctor.StatusOK}}
	p := newTestHealthPanel(checks)
	p.SetSize(30, 20)
	items := p.renderCheckItems()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if !strings.Contains(items[0], "...") {
		t.Error("expected truncation with '...'")
	}
}

func TestHealthPanel_HealthSummaryLinesConstant(t *testing.T) {
	if healthSummaryLines != 2 {
		t.Errorf("expected 2, got %d", healthSummaryLines)
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
