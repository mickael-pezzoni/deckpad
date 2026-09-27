package stats

import "testing"

func TestLuidOf(t *testing.T) {
	cases := map[string]string{
		"pid_42_luid_0x00000000_0x0000D1B5_phys_0_eng_0_engtype_3D": "0x00000000_0x0000D1B5",
		"luid_0x00000000_0x0000D1B5_phys_0":                         "0x00000000_0x0000D1B5",
		"_Total":                                                    "",
	}
	for in, want := range cases {
		if got := luidOf(in); got != want {
			t.Errorf("luidOf(%q) = %q, attendu %q", in, got, want)
		}
	}
}
