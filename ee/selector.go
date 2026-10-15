package ee

import "slices"

// Selector decides whether an action handles an event, based on its labels.
type Selector interface {
	Match(labels L) bool
}

type selectorFunc func(L) bool

func (f selectorFunc) Match(l L) bool { return f(l) }

// Match selects events whose label key equals value.
func Match(key, value string) Selector {
	return selectorFunc(func(l L) bool {
		v, ok := l[key]

		return ok && v == value
	})
}

// MatchIn selects events whose label key is one of values.
func MatchIn(key string, values ...string) Selector {
	return selectorFunc(func(l L) bool {
		v, ok := l[key]

		return ok && slices.Contains(values, v)
	})
}

// HasLabel selects events that carry key, whatever its value.
func HasLabel(key string) Selector {
	return selectorFunc(func(l L) bool {
		_, ok := l[key]

		return ok
	})
}

func AllOf(sels ...Selector) Selector {
	return selectorFunc(func(l L) bool {
		for _, s := range sels {
			if !s.Match(l) {
				return false
			}
		}

		return true
	})
}

func AnyOf(sels ...Selector) Selector {
	return selectorFunc(func(l L) bool {
		for _, s := range sels {
			if s.Match(l) {
				return true
			}
		}

		return false
	})
}

func Not(s Selector) Selector {
	return selectorFunc(func(l L) bool { return !s.Match(l) })
}
