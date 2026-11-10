package sshconn

import "testing"

func TestParseOSInfo(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want OSInfo
	}{
		{"ubuntu",
			"PRETTY_NAME=\"Ubuntu 24.04.3 LTS\"\nNAME=\"Ubuntu\"\nID=ubuntu\nID_LIKE=debian\nLinux\n",
			OSInfo{ID: "ubuntu", Name: "Ubuntu 24.04.3 LTS", Like: "debian"}},
		{"single quotes", "ID='alpine'\nNAME='Alpine Linux'\nLinux\n",
			OSInfo{ID: "alpine", Name: "Alpine Linux"}},
		{"macos", "Darwin\n", OSInfo{ID: "macos", Name: "macOS"}},
		{"freebsd", "FreeBSD\n", OSInfo{ID: "freebsd", Name: "FreeBSD"}},
		{"bare linux", "Linux\n", OSInfo{ID: "linux", Name: "Linux"}},
		{"unknown kernel", "SunOS\n", OSInfo{Name: "SunOS"}},
		{"empty", "", OSInfo{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := parseOSInfo(c.in); got != c.want {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
		})
	}
}