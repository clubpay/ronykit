package flow

import "go.temporal.io/sdk/temporal"

type (
	SearchAttributeUpdate = temporal.SearchAttributeUpdate
	SearchAttributes      = temporal.SearchAttributes
)

func NewSearchAttributes(attributes ...SearchAttributeUpdate) SearchAttributes {
	return temporal.NewSearchAttributes(attributes...)
}

func AttrString(name string, val string) SearchAttributeUpdate {
	return temporal.NewSearchAttributeKeyString(name).ValueSet(val)
}

func AttrBool(name string, val bool) SearchAttributeUpdate {
	return temporal.NewSearchAttributeKeyBool(name).ValueSet(val)
}

func AttrInt64(name string, val int64) SearchAttributeUpdate {
	return temporal.NewSearchAttributeKeyInt64(name).ValueSet(val)
}

func AttrKeywords(name string, val []string) SearchAttributeUpdate {
	return temporal.NewSearchAttributeKeyKeywordList(name).ValueSet(val)
}
