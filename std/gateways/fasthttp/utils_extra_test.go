package fasthttp

import "testing"

func TestUtilConversions(t *testing.T) {
	u := defaultUtil()
	if got := u.b2s([]byte("abc")); got != "abc" {
		t.Fatalf("default b2s mismatch: %s", got)
	}
	if got := string(u.s2b("def")); got != "def" {
		t.Fatalf("default s2b mismatch: %s", got)
	}

	us := speedUtil()
	if got := us.b2s([]byte("xyz")); got != "xyz" {
		t.Fatalf("speed b2s mismatch: %s", got)
	}
	if got := string(us.s2b("uvw")); got != "uvw" {
		t.Fatalf("speed s2b mismatch: %s", got)
	}
}

func TestGetMultipartFormBoundary(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"multipart/form-data; boundary=abc", "abc"},
		{"multipart/form-data; boundary=\"abc\"", "abc"},
		{"multipart/form-data; charset=utf-8; boundary=abc", "abc"},
		{"multipart/form-data", ""},
		{"text/plain", ""},
		{"multipart/form-data; foo=bar", ""},
		{"multipart/form-data; boundary=", ""},
	}

	for _, tt := range tests {
		got := getMultipartFormBoundary([]byte(tt.in))
		if tt.want == "" {
			if got != nil && len(got) != 0 {
				t.Fatalf("expected empty boundary for %q, got %q", tt.in, string(got))
			}
			continue
		}
		if string(got) != tt.want {
			t.Fatalf("expected boundary %q for %q, got %q", tt.want, tt.in, string(got))
		}
	}
}
