package di

import "testing"

func TestConfigSearchPathUsesConfigRoot(t *testing.T) {
	t.Cleanup(func() {
		SetConfigRoot("./config")
	})

	SetConfigRoot("/etc/myapp/config")
	if got := ConfigSearchPath("service"); got != "/etc/myapp/config/service" {
		t.Fatalf("ConfigSearchPath(service) = %q, want %q", got, "/etc/myapp/config/service")
	}
	if got := ConfigRoot(); got != "/etc/myapp/config" {
		t.Fatalf("ConfigRoot() = %q, want %q", got, "/etc/myapp/config")
	}
}

func TestSetConfigRootEmptyResetsDefault(t *testing.T) {
	t.Cleanup(func() {
		SetConfigRoot("./config")
	})

	SetConfigRoot("")
	if got := ConfigRoot(); got != "./config" {
		t.Fatalf("ConfigRoot() = %q, want %q", got, "./config")
	}
	if got := ConfigSearchPath("gateway"); got != "config/gateway" {
		t.Fatalf("ConfigSearchPath(gateway) = %q, want %q", got, "config/gateway")
	}
}

func TestConfigFilenameUsesLastPathSegment(t *testing.T) {
	if got := ConfigFilename("feature/auth"); got != "auth.local" {
		t.Fatalf("ConfigFilename(feature/auth) = %q, want %q", got, "auth.local")
	}
}
