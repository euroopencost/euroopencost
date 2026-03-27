package pricing

import "testing"

func TestParsePrice(t *testing.T) {
	cases := []struct {
		input   string
		want    float64
		wantErr bool
	}{
		{"0.052900 EUR", 0.0529, false},
		{"1.200000 EUR", 1.2, false},
		{"0.000000 EUR", 0.0, false},
		{"0.052900", 0.0529, false}, // no unit — still parseable
		{"", 0, true},
		{"not-a-number EUR", 0, true},
	}
	for _, tc := range cases {
		got, err := ParsePrice(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("input=%q: expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("input=%q: unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("input=%q: got %v, want %v", tc.input, got, tc.want)
		}
	}
}
