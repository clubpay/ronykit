package desc

import "strings"

type MessageMeta struct {
	Fields map[string]FieldMeta
}

type MessageMetaOption func(*MessageMeta)

func WithField(field string, fieldMeta FieldMeta) MessageMetaOption {
	return func(meta *MessageMeta) {
		if meta.Fields == nil {
			meta.Fields = make(map[string]FieldMeta)
		}

		meta.Fields[field] = fieldMeta
	}
}

type FormDataValue struct {
	Name string
	Type string
}

type FieldMeta struct {
	Optional   bool
	Deprecated bool
	OmitEmpty  bool
	Enum       []string
	FormData   *FormDataValue
}

func (fm FieldMeta) SwagTag() string {
	hasItem := false
	sb := strings.Builder{}
	sb.WriteString(`swag:"`)
	if fm.Optional {
		sb.WriteString("optional")
		hasItem = true
	}
	if fm.Deprecated {
		if hasItem {
			sb.WriteRune(',')
		}
		sb.WriteString("deprecated")
		hasItem = true
	}
	if fm.OmitEmpty {
		if hasItem {
			sb.WriteRune(',')
		}
		sb.WriteString("omitempty")
		hasItem = true
	}
	if len(fm.Enum) > 0 {
		if hasItem {
			sb.WriteRune(',')
		}
		sb.WriteString("enum:")
		sb.WriteString(strings.Join(fm.Enum, ","))
		hasItem = true
	}
	sb.WriteString(`"`)

	return sb.String()
}
