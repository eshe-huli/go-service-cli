package domain

import "testing"

func TestGreeting(t *testing.T) {
	for _, tc := range []struct{ name, want string }{{"Ben", "Hello, Ben!"}, {"  Valérie  ", "Hello, Valérie!"}} {
		if got := Greeting(tc.name); got != tc.want {
			t.Fatalf("got %q want %q", got, tc.want)
		}
	}
}
