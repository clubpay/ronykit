package di

import "testing"

type testSettings struct {
	DB struct {
		Host string
		Port int
	}
	Redis struct {
		User string
	}
}

func TestGetField(t *testing.T) {
	var s testSettings
	s.DB.Host = "db.local"
	s.DB.Port = 5432
	s.Redis.User = "redis-user"

	if got := getField[string](s, "DB", "Host"); got != "db.local" {
		t.Fatalf("Host = %q, want %q", got, "db.local")
	}
	if got := getField[int](s, "DB", "Port"); got != 5432 {
		t.Fatalf("Port = %d, want %d", got, 5432)
	}
	if got := getField[string](s, "Redis", "User"); got != "redis-user" {
		t.Fatalf("User = %q, want %q", got, "redis-user")
	}
}

func TestGetOptionalField(t *testing.T) {
	var s testSettings
	s.Redis.User = "redis-user"

	// testSettings.Redis has no TLS field: must return the zero value, not panic.
	if got := getOptionalField[bool](s, "Redis", "TLS"); got != false {
		t.Fatalf("TLS = %v, want %v", got, false)
	}

	// Missing parent field must also return the zero value, not panic.
	if got := getOptionalField[string](s, "Cache", "Host"); got != "" {
		t.Fatalf("Cache.Host = %q, want %q", got, "")
	}

	type settingsWithTLS struct {
		Redis struct {
			TLS bool
		}
	}

	var withTLS settingsWithTLS
	withTLS.Redis.TLS = true

	if got := getOptionalField[bool](withTLS, "Redis", "TLS"); got != true {
		t.Fatalf("TLS = %v, want %v", got, true)
	}
}

func TestConfigSearchPath(t *testing.T) {
	// 1. Without CONFIG_DIR env variable
	t.Setenv("CONFIG_DIR", "")
	got := ConfigSearchPath("test_kind")
	want := "./config/test_kind"
	if got != want {
		t.Errorf("ConfigSearchPath() without CONFIG_DIR = %q, want %q", got, want)
	}

	// 2. With CONFIG_DIR env variable
	t.Setenv("CONFIG_DIR", "/custom/config/dir")
	got = ConfigSearchPath("test_kind")
	want = "/custom/config/dir/test_kind"
	if got != want {
		t.Errorf("ConfigSearchPath() with CONFIG_DIR = %q, want %q", got, want)
	}
}
