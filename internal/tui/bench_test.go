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

// BenchmarkFilter measures one keystroke in the filter bar: the query is
// parsed and every device and process in view is matched against it.
func BenchmarkFilter(b *testing.B) {
	e := demoEngine(&testing.T{})
	m := New(e, Options{Theme: NewTheme("default", nil)})
	m.width, m.height = 200, 60
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.setSearch("util>50 !vendor:amd python")
		_ = m.devices()
		_ = m.processes()
	}
}
