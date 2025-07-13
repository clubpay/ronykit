package desc

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
