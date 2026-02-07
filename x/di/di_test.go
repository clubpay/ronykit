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
