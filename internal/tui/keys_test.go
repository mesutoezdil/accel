package tui

import "testing"

// A key the config asks for is taken off whatever holds it by default. This
// used to depend on map order, so binding quit to a key another action owned
// worked or did not from one start to the next.
func TestKeymapOverrideTakesAKeyFromItsDefaultOwner(t *testing.T) {
	km := NewKeymap(map[string]string{"quit": "x"})
	if km["x"] != ActQuit {
		t.Fatalf("x is bound to %q, want quit", km["x"])
	}
	for k, a := range km {
		if a == ActQuit && k != "x" {
			t.Errorf("quit is still on %q as well", k)
		}
	}
	if km["X"] != ActMarkAll {
		t.Errorf("X lost its binding: %q", km["X"])
	}
}

func TestKeymapIgnoresAnUnknownAction(t *testing.T) {
	km := NewKeymap(map[string]string{"launch_missiles": "!"})
	if _, ok := km["!"]; ok {
		t.Error("an action that does not exist took a key")
	}
}
