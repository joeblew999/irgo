package main

import "testing"

// TestDevPortHonoursPORT — irgo reports where the dev server will be, and the
// project decides that from PORT. If the two disagree, irgo prints a URL that
// nothing is listening on.
//
// t.Setenv rather than os.Setenv: it restores the old value and fails a
// parallel test rather than leaking into one, and it behaves the same on
// Windows, where environment variables are case-insensitive.
func TestDevPortHonoursPORT(t *testing.T) {
	for _, tc := range []struct{ set, want string }{
		{"", ":8080"},      // unset: the default mobile shells are built against
		{"8081", ":8081"},  // bare number, which is what people type
		{":8081", ":8081"}, // already a listen address
	} {
		t.Setenv("PORT", tc.set)
		if got := devPort(); got != tc.want {
			t.Errorf("PORT=%q: reported %q, want %q", tc.set, got, tc.want)
		}
	}
}
