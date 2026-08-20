package credentials

import "testing"

func TestServiceDefaultsToTheToolName(t *testing.T) {
	t.Setenv(ServiceEnvVar, "")

	if got := Service(); got != "kibana-tui" {
		t.Fatalf("Service() = %q, want %q", got, "kibana-tui")
	}
}

func TestServiceHonoursTheEnvironmentOverride(t *testing.T) {
	t.Setenv(ServiceEnvVar, "some-other-tool")

	if got := Service(); got != "some-other-tool" {
		t.Fatalf("Service() = %q, want %q", got, "some-other-tool")
	}
}

func TestCompleteNeedsBothHalves(t *testing.T) {
	cases := []struct {
		name  string
		creds Credentials
		want  bool
	}{
		{"empty", Credentials{}, false},
		{"username only", Credentials{Username: "u"}, false},
		{"password only", Credentials{Password: "p"}, false},
		{"both", Credentials{Username: "u", Password: "p"}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.creds.Complete(); got != c.want {
				t.Fatalf("Complete() = %v, want %v", got, c.want)
			}
		})
	}
}
