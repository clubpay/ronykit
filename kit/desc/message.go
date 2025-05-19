package desc

type MessageMeta struct {
	Enums map[string][]string `json:"enums"`
}

type MessageMetaOption func(*MessageMeta)

func WithFieldEnum(field string, enums []string) MessageMetaOption {
	return func(meta *MessageMeta) {
		if meta.Enums == nil {
			meta.Enums = make(map[string][]string)
		}

		meta.Enums[field] = enums
	}
}
