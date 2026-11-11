package printer

import "testing"

func TestBambuDisplayName_UsesModelTextNotSerial(t *testing.T) {
	cases := []struct {
		usn  string
		want string
	}{
		{"uuid:3DPrinter-X1-Carbon", "Bambu X1 @ 192.0.2.10"},
		{"P1P-abc", "Bambu P1P @ 192.0.2.10"},
		{"P1S", "Bambu P1S @ 192.0.2.10"},
		{"P2S-lan", "Bambu P2S @ 192.0.2.10"},
		{"H2D", "Bambu H2D @ 192.0.2.10"},
		{"H2S", "Bambu H2S @ 192.0.2.10"},
		{"A1 mini", "Bambu A1 mini @ 192.0.2.10"},
		{"A1", "Bambu A1 @ 192.0.2.10"},
		{"A2", "Bambu A2 @ 192.0.2.10"},
		{"H3", "Bambu H3 @ 192.0.2.10"},
		{"ABCDEF0123456789", "Bambu Printer @ 192.0.2.10"},
		{"", "Bambu Printer @ 192.0.2.10"},
	}
	for _, tc := range cases {
		got := bambuDisplayName(tc.usn, "192.0.2.10")
		if got != tc.want {
			t.Errorf("usn %q: got %q, want %q", tc.usn, got, tc.want)
		}
	}
}
