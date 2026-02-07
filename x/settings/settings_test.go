package settings

import "testing"

type nestedCfg struct {
	Host string `settings:"host" default:"localhost"`
	Port int    `settings:"port" default:"5432"`
}

type cfg struct {
	DB      nestedCfg `settings:"db"`
	Enabled bool      `settings:"enabled" default:"true"`
	Skip    string
}

func TestUnmarshalDefaultsAndOverrides(t *testing.T) {
	s := New()
	s.Set("db.host", "db.local")

	var out cfg
	if err := s.Unmarshal(&out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if out.DB.Host != "db.local" {
		t.Fatalf("Host = %q, want %q", out.DB.Host, "db.local")
	}
	if out.DB.Port != 5432 {
		t.Fatalf("Port = %d, want %d", out.DB.Port, 5432)
	}
	if out.Enabled != true {
		t.Fatalf("Enabled = %v, want true", out.Enabled)
	}
	if out.Skip != "" {
		t.Fatalf("Skip = %q, want empty", out.Skip)
	}
}
