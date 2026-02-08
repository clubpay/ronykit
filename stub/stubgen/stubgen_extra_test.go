package stubgen

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/clubpay/ronykit/kit/desc"
)

type localType struct {
	Value string
}

func TestInputTagsAndOptions(t *testing.T) {
	in := NewInput("svc")
	in.AddTags("json", "xml")
	if len(in.Tags()) != 2 || in.Tags()[0] != "json" || in.Tags()[1] != "xml" {
		t.Fatalf("unexpected tags: %#v", in.Tags())
	}

	in.AddExtraOptions(map[string]string{"feature": "on"})
	if got := in.GetOption("feature"); got != "on" {
		t.Fatalf("unexpected option value: %s", got)
	}

	in.AddExtraOptions(nil)
	if got := in.GetOption("feature"); got != "on" {
		t.Fatalf("option unexpectedly changed: %s", got)
	}
}

func TestInputBuiltinPkgPaths(t *testing.T) {
	in := NewInput("svc")
	in.AddMessage(desc.ParsedMessage{
		Name: "Test",
		Fields: []desc.ParsedField{
			{
				Element: &desc.ParsedElement{
					Kind:  desc.Object,
					RType: reflect.TypeOf(time.Time{}),
				},
			},
			{
				Element: &desc.ParsedElement{
					Kind:  desc.Object,
					RType: reflect.TypeOf(localType{}),
				},
			},
		},
	})

	paths := in.GetBuiltinPkgPaths()
	found := map[string]bool{}
	for _, p := range paths {
		found[p] = true
	}
	if !found["time"] {
		t.Fatalf("expected builtin package path 'time', got: %#v", paths)
	}
	if found[reflect.TypeOf(localType{}).PkgPath()] {
		t.Fatalf("unexpected non-stdlib package path present: %#v", paths)
	}
}

func TestGeneratorGenerateWritesFiles(t *testing.T) {
	tmpDir := t.TempDir()
	gen := New(
		WithGenEngine(GenFunc(func(in *Input) ([]GeneratedFile, error) {
			if in.Name() != "myStub" {
				return nil, errors.New("unexpected stub name")
			}

			return []GeneratedFile{
				{
					Filename: "out.txt",
					Data:     []byte("ok"),
				},
			}, nil
		})),
		WithStubName("myStub"),
		WithFolderName("gen"),
		WithOutputDir(tmpDir),
	)

	if err := gen.Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	outPath := filepath.Join(tmpDir, "gen", "out.txt")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected generated file: %v", err)
	}
	if string(data) != "ok" {
		t.Fatalf("unexpected file content: %s", string(data))
	}
}

func TestMustGeneratePanicsOnError(t *testing.T) {
	gen := New(WithGenEngine(GenFunc(func(*Input) ([]GeneratedFile, error) {
		return nil, errors.New("boom")
	})))

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from MustGenerate")
		}
	}()

	gen.MustGenerate()
}

func TestNewGolangEngineDefaultFilename(t *testing.T) {
	ge := NewGolangEngine(GolangConfig{PkgName: "test"})
	g, ok := ge.(*golangGE)
	if !ok {
		t.Fatalf("unexpected engine type: %T", ge)
	}
	if g.cfg.Filename != "stub.go" {
		t.Fatalf("expected default filename stub.go, got: %s", g.cfg.Filename)
	}
}

func TestRESTMethodResponses(t *testing.T) {
	ok := desc.ParsedResponse{}
	errResp := desc.ParsedResponse{ErrCode: 500}
	rm := RESTMethod{Responses: []desc.ParsedResponse{errResp, ok}}
	if !rm.HasOKResponse() {
		t.Fatal("expected OK response")
	}
	if got := rm.GetOKResponse(); got.ErrCode != 0 {
		t.Fatalf("unexpected OK response: %+v", got)
	}
	if len(rm.GetErrors()) != 1 {
		t.Fatalf("expected 1 error response, got: %d", len(rm.GetErrors()))
	}
	if rm.HasDefaultErrorResponse() {
		t.Fatal("expected HasDefaultErrorResponse to be false by default")
	}

	rm.DefaultErrorResponse = &errResp
	if !rm.HasDefaultErrorResponse() {
		t.Fatal("expected default error response")
	}
	if rm.GetDefaultErrorResponse() == nil {
		t.Fatal("expected default error response object")
	}
}

func TestRPCMethodResponses(t *testing.T) {
	ok := desc.ParsedResponse{}
	errResp := desc.ParsedResponse{ErrCode: 404}
	rm := RPCMethod{Responses: []desc.ParsedResponse{ok, errResp}}
	if got := rm.GetOKResponse(); got.ErrCode != 0 {
		t.Fatalf("unexpected OK response: %+v", got)
	}
	if len(rm.GetErrors()) != 1 {
		t.Fatalf("expected 1 error response, got: %d", len(rm.GetErrors()))
	}
}
