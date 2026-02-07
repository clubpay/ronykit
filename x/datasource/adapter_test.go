package datasource

import "testing"

func TestDBParamsDSN(t *testing.T) {
	params := DBParams{
		Host: "localhost",
		Port: 5432,
		User: "u",
		Pass: "p",
		DB:   "db",
	}
	got := params.DSN()
	want := "host=localhost user=u password=p database=db port=5432 sslmode=disable"
	if got != want {
		t.Fatalf("DSN = %q, want %q", got, want)
	}

	params.SSLMode = "require"
	got = params.DSN()
	want = "host=localhost user=u password=p database=db port=5432 sslmode=disable sslmode=require"
	if got != want {
		t.Fatalf("DSN with SSL = %q, want %q", got, want)
	}
}

func TestRedisParamsDSN(t *testing.T) {
	params := RedisParams{
		Host:     "localhost",
		Port:     6379,
		User:     "u",
		Pass:     "p@ss",
		DBNumber: 2,
	}
	got := params.DSN()
	want := "redis://u:p%40ss@localhost:6379/2"
	if got != want {
		t.Fatalf("DSN = %q, want %q", got, want)
	}

	params.DBNumber = 0
	got = params.DSN()
	want = "redis://u:p%40ss@localhost:6379"
	if got != want {
		t.Fatalf("DSN without DB = %q, want %q", got, want)
	}
}
