package mcp

import (
	"strings"
	"testing"

	"github.com/clubpay/ronykit/ronyup/cmd/mcp/knowledge"
)

func TestPackageInventoryCoversToolkit(t *testing.T) {
	t.Parallel()

	kb := mustLoadKB(t)
	got := map[string]knowledge.PackageDoc{}

	for _, pkg := range kb.Packages {
		got[pkg.ShortName] = pkg
	}

	want := []string{
		"apidoc", "batch", "cache", "datasource", "di", "errs", "flow",
		"i18n", "kit", "logkit", "meterkit", "p", "ratelimit", "rkit",
		"rony", "settings", "stub", "testkit", "tracekit",
	}

	for _, name := range want {
		pkg, ok := got[name]
		if !ok {
			t.Errorf("missing packages/%s.md (short_name %q)", name, name)

			continue
		}

		if strings.TrimSpace(pkg.ImportPath) == "" {
			t.Errorf("packages/%s.md has empty import_path", name)
		}

		if strings.TrimSpace(pkg.UsageHint) == "" {
			t.Errorf("packages/%s.md has empty Usage Hint", name)
		}
	}
}

func TestCharacteristicURIsMatchFilenames(t *testing.T) {
	t.Parallel()

	kb := mustLoadKB(t)
	wantPrimary := []string{
		"api", "cache", "database", "di", "i18n", "idempotent",
		"redis", "telemetry", "testing", "workflow",
	}

	byName := map[string]knowledge.CharacteristicDoc{}
	for _, ch := range kb.Characteristics {
		name := charResourceName(ch)
		if ch.Name != ch.Slug {
			t.Errorf("characteristic %q: Name %q != filename slug %q", ch.Slug, ch.Name, ch.Slug)
		}

		byName[name] = ch
	}

	for _, name := range wantPrimary {
		if _, ok := byName[name]; !ok {
			t.Errorf("missing characteristics/%s.md (resource name %q)", name, name)
		}
	}

	if resolveKnowledge(kb, categoryCharacteristics, "cache") == "" {
		t.Fatal("characteristics/cache should resolve")
	}

	if resolveKnowledge(kb, categoryCharacteristics, "api") == "" {
		t.Fatal("characteristics/api should resolve")
	}

	if resolveKnowledge(kb, categoryCharacteristics, "database") == "" {
		t.Fatal("characteristics/database should resolve")
	}

	if resolveKnowledge(kb, categoryCharacteristics, "testing") == "" {
		t.Fatal("characteristics/testing should resolve")
	}

	if resolveKnowledge(kb, categoryCharacteristics, "rest") == "" {
		t.Fatal("characteristics/rest alias should resolve to api")
	}

	cacheText := resolveKnowledge(kb, categoryCharacteristics, "cache")
	if strings.Contains(strings.ToLower(cacheText), "initredis") {
		t.Fatal("characteristics/cache must not teach InitRedis")
	}

	redisText := resolveKnowledge(kb, categoryCharacteristics, "redis")
	if redisText == "" {
		t.Fatal("characteristics/redis should resolve")
	}

	if !strings.Contains(redisText, "InitRedis") {
		t.Fatal("characteristics/redis should point at datasource.InitRedis")
	}

	if redisText == cacheText {
		t.Fatal("redis and cache characteristics must be distinct")
	}
}

func TestWriteServiceCodeUsesSetupOptionGroup(t *testing.T) {
	t.Parallel()

	kb := mustLoadKB(t)

	var body string

	for _, p := range kb.Prompts {
		if p.Name == "write-service-code" {
			body = p.Template

			break
		}
	}

	if body == "" {
		t.Fatal("write-service-code prompt not loaded")
	}

	if strings.Contains(body, "*desc.Service") {
		t.Error("write-service-code still documents Desc() *desc.Service")
	}

	if strings.Contains(body, "rony.Setup[") {
		t.Error("write-service-code still calls rony.Setup from Desc()")
	}

	if !strings.Contains(body, "rony.SetupOptionGroup") {
		t.Error("write-service-code must document rony.SetupOptionGroup")
	}
}

func TestKnowledgeSnippetsAvoidStaleSymbols(t *testing.T) {
	t.Parallel()

	kb := mustLoadKB(t)

	if strings.Contains(resolveKnowledge(kb, categoryArchitecture, "handler-relay"), "rony.ANY") {
		t.Error("handler-relay still uses rony.ANY")
	}

	if strings.Contains(resolveKnowledge(kb, categoryArchitecture, "gen-stub"), "ProvideXStub") {
		t.Error("gen-stub still uses di.ProvideXStub")
	}

	ratelimit := resolveKnowledge(kb, categoryPackages, "ratelimit")
	if strings.Contains(ratelimit, "errs.RateLimited") {
		t.Error("packages/ratelimit still uses errs.RateLimited")
	}

	apidoc := resolveKnowledge(kb, categoryPackages, "apidoc")
	if strings.Contains(apidoc, "Call apidoc.New()") || strings.Contains(apidoc, "apidoc.New() with") {
		t.Error("packages/apidoc still uses the zero-arg apidoc.New() constructor")
	}

	var workflow string

	for _, p := range kb.Prompts {
		if p.Name == "write-workflow" {
			workflow = p.Template

			break
		}
	}

	if strings.Contains(workflow, "workflow.DefaultVersion") {
		t.Error("write-workflow still uses workflow.DefaultVersion")
	}
}
