package mcp

import (
	"context"
	"slices"
	"testing"

	"github.com/clubpay/ronykit/ronyup/internal"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestFilterPrefix(t *testing.T) {
	items := []string{"packages", "architecture", "characteristics"}

	tests := []struct {
		name   string
		prefix string
		want   []string
	}{
		{name: "empty prefix returns all", prefix: "", want: items},
		{name: "match single", prefix: "pack", want: []string{"packages"}},
		{name: "match multiple", prefix: "c", want: []string{"characteristics"}},
		{name: "case insensitive", prefix: "ARCH", want: []string{"architecture"}},
		{name: "no match", prefix: "zzz", want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := filterPrefix(items, tc.prefix)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("filterPrefix(%v, %q) = %v, want %v", items, tc.prefix, got, tc.want)
			}
		})
	}
}

func TestNamesForCategory(t *testing.T) {
	kb := mustLoadKB(t)

	t.Run("packages returns package short names", func(t *testing.T) {
		names := namesForCategory(kb, "packages")
		if len(names) == 0 {
			t.Fatal("expected at least one package name")
		}

		if !slices.Contains(names, "di") {
			t.Fatalf("expected 'di' in packages, got %v", names)
		}
	})

	t.Run("architecture returns slugs", func(t *testing.T) {
		names := namesForCategory(kb, "architecture")
		if len(names) == 0 {
			t.Fatal("expected at least one architecture slug")
		}

		if !slices.Contains(names, "thin-handlers") {
			t.Fatalf("expected 'thin-handlers' in architecture, got %v", names)
		}
	})

	t.Run("characteristics returns keyword names", func(t *testing.T) {
		names := namesForCategory(kb, "characteristics")
		if len(names) == 0 {
			t.Fatal("expected at least one characteristic name")
		}
	})

	t.Run("unknown category returns all names", func(t *testing.T) {
		all := namesForCategory(kb, "")
		pkgs := namesForCategory(kb, "packages")
		arch := namesForCategory(kb, "architecture")
		chars := namesForCategory(kb, "characteristics")

		wantLen := len(pkgs) + len(arch) + len(chars)
		if len(all) != wantLen {
			t.Fatalf("empty category returned %d names, want %d", len(all), wantLen)
		}
	})
}

func TestCompletionHandler_CategoryArg(t *testing.T) {
	kb := mustLoadKB(t)
	handler := completionHandler(kb)
	ctx := context.Background()

	tests := []struct {
		name      string
		value     string
		wantAll   bool
		wantItems []string
	}{
		{
			name:    "empty value returns all categories",
			value:   "",
			wantAll: true,
		},
		{
			name:      "prefix 'p' matches packages",
			value:     "p",
			wantItems: []string{"packages"},
		},
		{
			name:      "prefix 'a' matches architecture",
			value:     "a",
			wantItems: []string{"architecture"},
		},
		{
			name:      "prefix 'ch' matches characteristics",
			value:     "ch",
			wantItems: []string{"characteristics"},
		},
		{
			name:      "no match returns empty",
			value:     "zzz",
			wantItems: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := handler(ctx, &mcpsdk.CompleteRequest{
				Params: &mcpsdk.CompleteParams{
					Ref: &mcpsdk.CompleteReference{
						Type: "ref/resource",
						URI:  resourceTemplateURI,
					},
					Argument: mcpsdk.CompleteParamsArgument{
						Name:  "category",
						Value: tc.value,
					},
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Completion.Values == nil {
				t.Fatal("Values must not be nil (should be empty slice)")
			}

			if tc.wantAll {
				if len(result.Completion.Values) != 3 {
					t.Fatalf("expected 3 categories, got %v", result.Completion.Values)
				}

				return
			}

			if !slices.Equal(result.Completion.Values, tc.wantItems) {
				t.Fatalf("got %v, want %v", result.Completion.Values, tc.wantItems)
			}
		})
	}
}

func TestCompletionHandler_NameArgWithContext(t *testing.T) {
	kb := mustLoadKB(t)
	handler := completionHandler(kb)
	ctx := context.Background()

	t.Run("name completions scoped to packages category", func(t *testing.T) {
		result, err := handler(ctx, &mcpsdk.CompleteRequest{
			Params: &mcpsdk.CompleteParams{
				Ref: &mcpsdk.CompleteReference{
					Type: "ref/resource",
					URI:  resourceTemplateURI,
				},
				Argument: mcpsdk.CompleteParamsArgument{
					Name:  "name",
					Value: "di",
				},
				Context: &mcpsdk.CompleteContext{
					Arguments: map[string]string{
						"category": "packages",
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !slices.Contains(result.Completion.Values, "di") {
			t.Fatalf("expected 'di' in completions, got %v", result.Completion.Values)
		}

		for _, v := range result.Completion.Values {
			found := false

			for _, pkg := range kb.Packages {
				if pkg.ShortName == v {
					found = true

					break
				}
			}

			if !found {
				t.Fatalf("completion %q is not a valid package name", v)
			}
		}
	})

	t.Run("name completions scoped to architecture", func(t *testing.T) {
		result, err := handler(ctx, &mcpsdk.CompleteRequest{
			Params: &mcpsdk.CompleteParams{
				Ref: &mcpsdk.CompleteReference{
					Type: "ref/resource",
					URI:  resourceTemplateURI,
				},
				Argument: mcpsdk.CompleteParamsArgument{
					Name:  "name",
					Value: "thin",
				},
				Context: &mcpsdk.CompleteContext{
					Arguments: map[string]string{
						"category": "architecture",
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !slices.Contains(result.Completion.Values, "thin-handlers") {
			t.Fatalf("expected 'thin-handlers' in completions, got %v", result.Completion.Values)
		}
	})

	t.Run("name without context returns all names", func(t *testing.T) {
		result, err := handler(ctx, &mcpsdk.CompleteRequest{
			Params: &mcpsdk.CompleteParams{
				Ref: &mcpsdk.CompleteReference{
					Type: "ref/resource",
					URI:  resourceTemplateURI,
				},
				Argument: mcpsdk.CompleteParamsArgument{
					Name:  "name",
					Value: "",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		allNames := namesForCategory(kb, "")
		if len(result.Completion.Values) != len(allNames) {
			t.Fatalf("expected %d names, got %d", len(allNames), len(result.Completion.Values))
		}
	})
}

func TestCompletionHandler_NonMatchingRef(t *testing.T) {
	kb := mustLoadKB(t)
	handler := completionHandler(kb)
	ctx := context.Background()

	t.Run("nil request returns empty", func(t *testing.T) {
		result, err := handler(ctx, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Completion.Values) != 0 {
			t.Fatalf("expected empty values, got %v", result.Completion.Values)
		}
	})

	t.Run("nil params returns empty", func(t *testing.T) {
		result, err := handler(ctx, &mcpsdk.CompleteRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Completion.Values) != 0 {
			t.Fatalf("expected empty values, got %v", result.Completion.Values)
		}
	})

	t.Run("nil ref returns empty", func(t *testing.T) {
		result, err := handler(ctx, &mcpsdk.CompleteRequest{
			Params: &mcpsdk.CompleteParams{
				Argument: mcpsdk.CompleteParamsArgument{
					Name:  "category",
					Value: "",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Completion.Values) != 0 {
			t.Fatalf("expected empty values, got %v", result.Completion.Values)
		}
	})

	t.Run("wrong URI template returns empty", func(t *testing.T) {
		result, err := handler(ctx, &mcpsdk.CompleteRequest{
			Params: &mcpsdk.CompleteParams{
				Ref: &mcpsdk.CompleteReference{
					Type: "ref/resource",
					URI:  "file:///some/other/{path}",
				},
				Argument: mcpsdk.CompleteParamsArgument{
					Name:  "category",
					Value: "",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Completion.Values) != 0 {
			t.Fatalf("expected empty values, got %v", result.Completion.Values)
		}
	})
}

func TestCompletionHandler_NameArgCaseInsensitiveCategoryContext(t *testing.T) {
	kb := mustLoadKB(t)
	handler := completionHandler(kb)
	ctx := context.Background()

	result, err := handler(ctx, &mcpsdk.CompleteRequest{
		Params: &mcpsdk.CompleteParams{
			Ref: &mcpsdk.CompleteReference{
				Type: "REF/RESOURCE",
				URI:  " knowledge://ronyup/{category}/{name} ",
			},
			Argument: mcpsdk.CompleteParamsArgument{
				Name:  " NAME ",
				Value: "di",
			},
			Context: &mcpsdk.CompleteContext{
				Arguments: map[string]string{
					"category": " Packages ",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !slices.Contains(result.Completion.Values, "di") {
		t.Fatalf("expected 'di' in completions, got %v", result.Completion.Values)
	}

	for _, v := range result.Completion.Values {
		found := false

		for _, pkg := range kb.Packages {
			if pkg.ShortName == v {
				found = true

				break
			}
		}

		if !found {
			t.Fatalf("completion %q not in packages", v)
		}
	}
}

func TestCompletionHandler_ValuesNeverNil(t *testing.T) {
	kb := mustLoadKB(t)
	handler := completionHandler(kb)
	ctx := context.Background()

	result, err := handler(ctx, &mcpsdk.CompleteRequest{
		Params: &mcpsdk.CompleteParams{
			Ref: &mcpsdk.CompleteReference{
				Type: "ref/resource",
				URI:  resourceTemplateURI,
			},
			Argument: mcpsdk.CompleteParamsArgument{
				Name:  "category",
				Value: "zzz_no_match",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Completion.Values == nil {
		t.Fatal("Values must never be nil, should be empty slice")
	}
}

func TestCompletionE2E(t *testing.T) {
	kb := mustLoadKB(t)
	srv := newServer(serverConfig{
		name:         "ronyup-test",
		version:      "v0.0.0-test",
		instructions: "test",
		skeletonFS:   internal.Skeleton,
		cmdRunner:    defaultRunner{},
		kb:           kb,
	})

	ctx := context.Background()
	ct, st := mcpsdk.NewInMemoryTransports()

	ss, err := srv.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}

	t.Cleanup(func() { _ = ss.Close() })

	client := mcpsdk.NewClient(&mcpsdk.Implementation{
		Name:    "test-client",
		Version: "v0.0.0",
	}, nil)

	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}

	t.Cleanup(func() { _ = cs.Close() })

	t.Run("complete category", func(t *testing.T) {
		result, err := cs.Complete(ctx, &mcpsdk.CompleteParams{
			Ref: &mcpsdk.CompleteReference{
				Type: "ref/resource",
				URI:  resourceTemplateURI,
			},
			Argument: mcpsdk.CompleteParamsArgument{
				Name:  "category",
				Value: "",
			},
		})
		if err != nil {
			t.Fatalf("Complete() error: %v", err)
		}

		if len(result.Completion.Values) != 3 {
			t.Fatalf("expected 3 categories, got %v", result.Completion.Values)
		}

		want := []string{"packages", "architecture", "characteristics"}
		if !slices.Equal(result.Completion.Values, want) {
			t.Fatalf("got %v, want %v", result.Completion.Values, want)
		}
	})

	t.Run("complete category with prefix", func(t *testing.T) {
		result, err := cs.Complete(ctx, &mcpsdk.CompleteParams{
			Ref: &mcpsdk.CompleteReference{
				Type: "ref/resource",
				URI:  resourceTemplateURI,
			},
			Argument: mcpsdk.CompleteParamsArgument{
				Name:  "category",
				Value: "pack",
			},
		})
		if err != nil {
			t.Fatalf("Complete() error: %v", err)
		}

		if !slices.Equal(result.Completion.Values, []string{"packages"}) {
			t.Fatalf("expected [packages], got %v", result.Completion.Values)
		}
	})

	t.Run("complete name with category context", func(t *testing.T) {
		result, err := cs.Complete(ctx, &mcpsdk.CompleteParams{
			Ref: &mcpsdk.CompleteReference{
				Type: "ref/resource",
				URI:  resourceTemplateURI,
			},
			Argument: mcpsdk.CompleteParamsArgument{
				Name:  "name",
				Value: "di",
			},
			Context: &mcpsdk.CompleteContext{
				Arguments: map[string]string{
					"category": "packages",
				},
			},
		})
		if err != nil {
			t.Fatalf("Complete() error: %v", err)
		}

		if !slices.Contains(result.Completion.Values, "di") {
			t.Fatalf("expected 'di' in completions, got %v", result.Completion.Values)
		}

		for _, v := range result.Completion.Values {
			found := false

			for _, pkg := range kb.Packages {
				if pkg.ShortName == v {
					found = true

					break
				}
			}

			if !found {
				t.Fatalf("completion %q not in packages", v)
			}
		}
	})

	t.Run("complete name for architecture", func(t *testing.T) {
		result, err := cs.Complete(ctx, &mcpsdk.CompleteParams{
			Ref: &mcpsdk.CompleteReference{
				Type: "ref/resource",
				URI:  resourceTemplateURI,
			},
			Argument: mcpsdk.CompleteParamsArgument{
				Name:  "name",
				Value: "thin",
			},
			Context: &mcpsdk.CompleteContext{
				Arguments: map[string]string{
					"category": "architecture",
				},
			},
		})
		if err != nil {
			t.Fatalf("Complete() error: %v", err)
		}

		if !slices.Contains(result.Completion.Values, "thin-handlers") {
			t.Fatalf("expected 'thin-handlers', got %v", result.Completion.Values)
		}
	})
}
