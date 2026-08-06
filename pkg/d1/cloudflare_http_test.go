//go:build !(js && wasm)

package d1

import (
	"database/sql/driver"
	"encoding/json"
	"io"
	"testing"
)

// TestKeyOrderFollowsTheQuery is the test that matters most here. D1 returns
// rows as JSON objects, and decoding one into a map loses the order the
// columns were selected in. If the order came from map iteration, `SELECT id,
// name` would scan a name into an id on some runs and not others — which
// presents as data corruption rather than as a bug in a driver.
func TestKeyOrderFollowsTheQuery(t *testing.T) {
	cases := []struct {
		name string
		row  string
		want []string
	}{
		{"simple", `{"id":1,"name":"ada","email":"a@b.c"}`, []string{"id", "name", "email"}},
		{"reversed", `{"email":"a@b.c","name":"ada","id":1}`, []string{"email", "name", "id"}},
		{"nested values are skipped whole", `{"id":1,"meta":{"a":{"b":2}},"tags":[1,2,3],"z":null}`,
			[]string{"id", "meta", "tags", "z"}},
		{"a column named like a delimiter", `{"{":1,"}":2}`, []string{"{", "}"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := keyOrder(json.RawMessage(tc.row))
			if err != nil {
				t.Fatalf("keyOrder: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("column %d: got %q, want %q (full: %v)", i, got[i], tc.want[i], got)
				}
			}
		})
	}
}

// TestIntegersDoNotArriveAsFloats guards the other silent one: every JSON
// number decodes to float64, so an INTEGER column would fail to scan into an
// int without this, reporting a type error far from the cause.
func TestIntegersDoNotArriveAsFloats(t *testing.T) {
	cases := []struct {
		in   any
		want driver.Value
	}{
		{float64(42), int64(42)},
		{float64(0), int64(0)},
		{float64(-7), int64(-7)},
		{1.5, 1.5},
		{"text", "text"},
		{true, true},
		{nil, nil},
	}
	for _, tc := range cases {
		if got := convert(tc.in); got != tc.want {
			t.Errorf("convert(%v): got %#v (%T), want %#v (%T)", tc.in, got, got, tc.want, tc.want)
		}
	}
}

// TestRowsScanInSelectedOrder walks a result set the way database/sql does.
func TestRowsScanInSelectedOrder(t *testing.T) {
	res := &queryResult{Results: []json.RawMessage{
		json.RawMessage(`{"id":1,"name":"ada"}`),
		json.RawMessage(`{"id":2,"name":"grace"}`),
	}}
	r, err := newRows(res)
	if err != nil {
		t.Fatal(err)
	}
	if cols := r.Columns(); len(cols) != 2 || cols[0] != "id" || cols[1] != "name" {
		t.Fatalf("columns: %v", cols)
	}

	dest := make([]driver.Value, 2)
	if err := r.Next(dest); err != nil {
		t.Fatal(err)
	}
	if dest[0] != int64(1) || dest[1] != "ada" {
		t.Fatalf("row 1: %#v", dest)
	}
	if err := r.Next(dest); err != nil {
		t.Fatal(err)
	}
	if dest[0] != int64(2) || dest[1] != "grace" {
		t.Fatalf("row 2: %#v", dest)
	}
	if err := r.Next(dest); err != io.EOF {
		t.Fatalf("after the last row: got %v, want io.EOF", err)
	}
}

// TestEmptyResultHasNoColumns: a query that matches nothing must not error.
func TestEmptyResultHasNoColumns(t *testing.T) {
	r, err := newRows(&queryResult{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Columns()) != 0 {
		t.Fatalf("columns: %v", r.Columns())
	}
	if err := r.Next(make([]driver.Value, 0)); err != io.EOF {
		t.Fatalf("got %v, want io.EOF", err)
	}
}

// TestConfigIsReportedBeforeTheRequest — a missing token should be named, not
// discovered as a 401 later.
func TestConfigIsReportedBeforeTheRequest(t *testing.T) {
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	t.Setenv("D1_DATABASE_ID", "")

	_, err := Open("DB")
	if err == nil {
		t.Fatal("expected an error when nothing is configured")
	}
	if got := err.Error(); !contains(got, "CLOUDFLARE_ACCOUNT_ID") {
		t.Errorf("the error should name the variable to set, got: %s", got)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
