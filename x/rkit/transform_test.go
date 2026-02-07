package rkit_test

import (
	"testing"

	"github.com/clubpay/ronykit/x/rkit"
)

func TestToCamel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"trim_and_separators", "  hello_world ", "HelloWorld"},
		{"kebab", "user-id", "UserId"},
		{"dot", "user.id", "UserId"},
		{"acronym", "ID", "Id"},
		{"already_camel", "alreadyCamel", "AlreadyCamel"},
		{"slash", "user/id", "UserId"},
		{"backslash", "user\\id", "UserId"},
		{"pipe", "user|id", "UserId"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := rkit.ToCamel(tt.in); got != tt.want {
				t.Fatalf("ToCamel(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestToLowerCamel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"trim_and_separators", "  hello_world ", "helloWorld"},
		{"kebab", "user-id", "userId"},
		{"dot", "user.id", "userId"},
		{"acronym", "ID", "id"},
		{"already_camel", "alreadyCamel", "alreadyCamel"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := rkit.ToLowerCamel(tt.in); got != tt.want {
				t.Fatalf("ToLowerCamel(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestToSnake(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"camel", "HelloWorld", "hello_world"},
		{"lower_camel", "helloWorld", "hello_world"},
		{"acronym", "JSONData", "json_data"},
		{"with_spaces", "hello world", "hello_world"},
		{"kebab", "hello-world", "hello_world"},
		{"already_snake", "hello_world", "hello_world"},
		{"numbers", "version2Value", "version_2_value"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := rkit.ToSnake(tt.in); got != tt.want {
				t.Fatalf("ToSnake(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestToSnakeWithIgnore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		in     string
		ignore uint8
		want   string
	}{
		{"ignore_hyphen", "Foo-BarBaz", '-', "foo-bar_baz"},
		{"ignore_space", "Foo BarBaz", ' ', "foo bar_baz"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := rkit.ToSnakeWithIgnore(tt.in, tt.ignore); got != tt.want {
				t.Fatalf("ToSnakeWithIgnore(%q, %q) = %q, want %q", tt.in, tt.ignore, got, tt.want)
			}
		})
	}
}

func TestDelimitedVariants(t *testing.T) {
	t.Parallel()

	t.Run("screaming_snake", func(t *testing.T) {
		t.Parallel()

		if got := rkit.ToScreamingSnake("HelloWorld"); got != "HELLO_WORLD" {
			t.Fatalf("ToScreamingSnake = %q, want %q", got, "HELLO_WORLD")
		}
	})

	t.Run("kebab", func(t *testing.T) {
		t.Parallel()

		if got := rkit.ToKebab("HelloWorld"); got != "hello-world" {
			t.Fatalf("ToKebab = %q, want %q", got, "hello-world")
		}
	})

	t.Run("screaming_kebab", func(t *testing.T) {
		t.Parallel()

		if got := rkit.ToScreamingKebab("HelloWorld"); got != "HELLO-WORLD" {
			t.Fatalf("ToScreamingKebab = %q, want %q", got, "HELLO-WORLD")
		}
	})

	t.Run("delimited_dot", func(t *testing.T) {
		t.Parallel()

		if got := rkit.ToDelimited("HelloWorld", '.'); got != "hello.world" {
			t.Fatalf("ToDelimited = %q, want %q", got, "hello.world")
		}
	})

	t.Run("screaming_delimited_dot", func(t *testing.T) {
		t.Parallel()

		if got := rkit.ToScreamingDelimited("HelloWorld", '.', 0, true); got != "HELLO.WORLD" {
			t.Fatalf("ToScreamingDelimited = %q, want %q", got, "HELLO.WORLD")
		}
	})
}
