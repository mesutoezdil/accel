package host

import "testing"

func TestField(t *testing.T) {
	for _, c := range []struct {
		in   string
		i    int
		want string
	}{
		{"  1000 kB", 0, "1000"},
		{"  1000 kB", 1, "kB"},
		{"  1000 kB", 2, ""},
		{"", 0, ""},
		{"   ", 0, ""}, // a key printed with no value after it
		{"\n", 0, ""},
		{"4096", 0, "4096"},
	} {
		if got := field(c.in, c.i); got != c.want {
			t.Errorf("field(%q, %d) = %q, want %q", c.in, c.i, got, c.want)
		}
	}
}
