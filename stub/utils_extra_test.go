package stub

import "testing"

func TestConvertLegacyPathFormat(t *testing.T) {
	out := convertLegacyPathFormat("/items/:id/extra")
	if out != "/items/{id}/extra" {
		t.Fatalf("unexpected legacy format: %s", out)
	}
}
