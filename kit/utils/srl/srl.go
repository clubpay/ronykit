package srl

import (
	"regexp"
	"strings"
)

const (
	pathGlue = ":"
	glue     = "@"
)

const (
	srliStorage = iota
	srliPath
	srliID

	srlLength
)

var pattern = regexp.MustCompile(`(?:([^:\s]*):)?([^@\s]*)@?(.*)`)

// SRL stands for Simple Resource Locator, A 3 part address.
type SRL [srlLength]string

// New creates a SRL of path & id on a specific storage.
func New(storage, path, id string) SRL {
	return SRL{storage, path, id}
}

// Portable creates a SRL of path & id on unspecific storage.
func Portable(path, id string) SRL {
	return New("", path, id)
}

// Storage creates a SRL without any path & id on a specific storage.
func Storage(storage string) SRL {
	return New(storage, "", "")
}

func Parse(str string) SRL {
	q := SRL{}

	mm := pattern.FindAllStringSubmatch(str, -1)
	if len(mm) == 0 {
		return q
	}

	pp := mm[0]
	for i := srlLength; i > 0; i-- {
		if len(pp) > i {
			q[i-1] = pp[i]
		}
	}

	return q
}

// Append returns a new SRL with non-empty fields of src appended to it.
// It joins paths if available on both q & src with pathGlue
func (q SRL) Append(src SRL) SRL {
	return q.mix(
		src,
		func(qi, si string, i int) string {
			if len(si) == 0 {
				return qi
			}

			if srliPath == i && len(qi) > 0 {
				return strings.Join([]string{qi, si}, pathGlue)
			}

			return si
		},
	)
}

// Merge returns a new SRL with fields replaced by non-empty src fields.
func (q SRL) Merge(src SRL) SRL {
	return q.mix(
		src,
		func(qi, si string, i int) string {
			if len(si) == 0 {
				return qi
			}

			return si
		},
	)
}

// Replace returns a new SRL with all fields replaced by src fields.
func (q SRL) Replace(src SRL) SRL {
	return q.mix(
		src,
		func(qi, si string, i int) string {
			return si
		},
	)
}

func (q SRL) mix(src SRL, fn func(qi, si string, i int) string) SRL {
	for i := range srlLength {
		q[i] = fn(q[i], src[i], i)
	}

	return q
}

func (q SRL) String() string {
	var sb strings.Builder

	for i := range srlLength {
		pglue, sglue := "", ""

		switch i {
		case srliStorage:
			sglue = pathGlue

		case srliID:
			pglue = glue
		}

		p := q[i]
		if l := len(p); l > 0 {
			sb.Grow(l + 1)
			sb.WriteString(pglue)
			sb.WriteString(p)
			sb.WriteString(sglue)
		}
	}

	return sb.String()
}

func (q SRL) Storage() string {
	return q[srliStorage]
}

func (q SRL) Path() string {
	return q[srliPath]
}

func (q SRL) ID() string {
	return q[srliID]
}
