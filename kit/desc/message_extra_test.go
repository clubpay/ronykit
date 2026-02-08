package desc

import "testing"

func TestFieldMetaSwagTag(t *testing.T) {
	if (FieldMeta{}).SwagTag() != `swag:""` {
		t.Fatalf("unexpected empty swag tag: %s", (FieldMeta{}).SwagTag())
	}

	fm := FieldMeta{
		Optional:   true,
		Deprecated: true,
		OmitEmpty:  true,
		Enum:       []string{"a", "b"},
	}
	if fm.SwagTag() != `swag:"optional,deprecated,omitempty,enum:a,b"` {
		t.Fatalf("unexpected swag tag: %s", fm.SwagTag())
	}
}

func TestMessageMetaWithField(t *testing.T) {
	meta := MessageMeta{}
	WithField("name", FieldMeta{Optional: true})(&meta)
	if meta.Fields["name"].Optional != true {
		t.Fatalf("unexpected field meta: %+v", meta.Fields["name"])
	}
}
