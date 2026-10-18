package fixmod

import (
	"slices"
	"strconv"
	"unsafe"
)

type item struct {
	Name string
	Qty  int
}

type opts struct {
	Host string
	Port int
}

func withDefaults(o opts) opts {
	if o.Host == "" {
		o.Host = "localhost"
	}

	if o.Port == 0 {
		o.Port = 80
	}
	if o.Port == 0 {
		o.Port = 8080
	}

	return o
}

func label(primary, secondary string) string {
	if primary != "" {
		return primary
	}

	if secondary != "" {
		return secondary
	}

	return "none"
}

func names(items []item) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Name)
	}

	return out
}

func inStock(items []item) []item {
	out := []item{}
	for _, it := range items {
		if it.Qty > 0 {
			out = append(out, it)
		}
	}

	return out
}

func counts(items []item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = strconv.Itoa(it.Qty)
	}

	return out
}

func has(items []string, name string) bool {
	return slices.Contains(items, name)
}

func asString(v any) string {
	s, _ := v.(string)

	return s
}

func view(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
