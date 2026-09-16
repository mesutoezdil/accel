package tui

import "testing"

// BenchmarkRender measures the overview at a large terminal.
func BenchmarkRender(b *testing.B) {
	e := demoEngine(&testing.T{})
	m := New(e, Options{Theme: NewTheme("default", nil)})
	m.width, m.height = 200, 60
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkRenderDashboard(b *testing.B) {
	e := demoEngine(&testing.T{})
	m := New(e, Options{Theme: NewTheme("default", nil)})
	m.width, m.height = 200, 60
	m.setTab(tabDashboard)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}
