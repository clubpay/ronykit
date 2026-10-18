package ee_test

import (
	"reflect"
	"testing"

	"github.com/clubpay/ronykit/ee"
	"github.com/clubpay/ronykit/ee/mem"
)

// swapCodec renames every queryable field, to prove ee translates query
// conditions through Codec.FieldName rather than assuming JSON names.
type swapCodec struct {
	codec ee.Codec
}

func newSwapCodec() swapCodec { return swapCodec{codec: ee.JSONCodec} }

func (c swapCodec) Marshal(v any) ([]byte, error) { return c.codec.Marshal(v) }

func (c swapCodec) Unmarshal(data []byte, v any) error { return c.codec.Unmarshal(data, v) }

func (c swapCodec) FieldName(f reflect.StructField) (string, bool) {
	name, ok := c.codec.FieldName(f)
	if !ok {
		return "", false
	}

	return "q_" + name, true
}

// swapDecoder mirrors swapCodec, producing the encoded field names the store
// side must see.
func swapDecoder(data []byte) (map[string]any, error) {
	doc, err := mem.JSONDecoder(data)
	if err != nil {
		return nil, err
	}

	out := make(map[string]any, len(doc))
	for k, v := range doc {
		out["q_"+k] = v
	}

	return out, nil
}

func TestCodecFieldNameContract(t *testing.T) {
	ctx := t.Context()

	// The store decodes documents through the same renaming as the codec.
	st := mem.NewStore(mem.WithDecoder(swapDecoder))
	env := ee.MustNew(ee.WithStore(st), ee.WithCodec(newSwapCodec()))
	orders := must(ee.NewEntityController[Order](env))
	t.Cleanup(func() { _ = env.Close() })

	_ = orders.Create(ctx, Order{ID: "o1", Merchant: "m1", Amount: 5})
	_ = orders.Create(ctx, Order{ID: "o2", Merchant: "m2", Amount: 1})

	got := []string{}
	for o, err := range orders.Find(ctx, ee.Where("Merchant", "m1")) {
		if err != nil {
			t.Fatal(err)
		}

		got = append(got, o.ID)
	}
	if len(got) != 1 || got[0] != "o1" {
		t.Errorf("rename filter: %v", got)
	}

	got = nil
	for o, err := range orders.Find(ctx, ee.Where("Merchant", "m2").And("Amount", ee.Gt, 0)) {
		if err != nil {
			t.Fatal(err)
		}

		got = append(got, o.ID)
	}
	if len(got) != 1 || got[0] != "o2" {
		t.Errorf("renamed and/or filter: %v", got)
	}
}
