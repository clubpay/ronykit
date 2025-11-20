package logkit

import (
	"encoding/json"
	"io"
	"reflect"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var _ zapcore.Encoder = (*Sensitive)(nil)

type sensitiveConfig struct {
	zapcore.EncoderConfig
}

type Sensitive struct {
	zapcore.Encoder
}

func newSensitive(cfg sensitiveConfig) *Sensitive {
	cfg.NewReflectedEncoder = newJSONEncoder

	return &Sensitive{
		Encoder: zapcore.NewJSONEncoder(cfg.EncoderConfig),
	}
}

func (s Sensitive) Clone() zapcore.Encoder {
	return Sensitive{Encoder: s.Encoder.Clone()}
}

func (s Sensitive) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	return s.Encoder.EncodeEntry(entry, fields)
}

var _ zapcore.ReflectedEncoder = (*jsonEncoder)(nil)

type jsonEncoder struct {
	enc *json.Encoder
}

func newJSONEncoder(
	w io.Writer,
) zapcore.ReflectedEncoder {
	return &jsonEncoder{enc: json.NewEncoder(w)}
}

func (j jsonEncoder) Encode(v any) error {
	rv := reflect.Indirect(reflect.ValueOf(v))

	switch rv.Kind() {
	case reflect.Struct:
		copiedRV := reflect.New(rv.Type()).Elem()
		copiedRV.Set(rv)

		return j.enc.Encode(maskStruct(copiedRV))
	default:
		return j.enc.Encode(v)
	}
}

func maskStruct(rv reflect.Value) any {
	if !rv.CanSet() {
		newRV := reflect.New(rv.Type())
		newRV.Elem().Set(rv)

		return maskStruct(newRV.Elem())
	}

	rvt := rv.Type()
	for i := range rvt.NumField() {
		f := rv.Field(i)
		if !f.CanSet() {
			continue
		}

		switch rvt.Field(i).Tag.Get("sensitive") {
		default:
		case "-", "true":
			f.Set(reflect.Zero(f.Type()))
		case "phone", "email":
			switch f.Kind() {
			default:
				f.Set(reflect.Zero(f.Type()))
			case reflect.Pointer:
				if f.Elem().Kind() == reflect.String {
					newStr := reflect.New(f.Type().Elem())

					fe := f.Elem()
					if fe.Len() < 4 {
						newStr.Elem().SetString("****")
					} else if fe.Len() < 8 {
						s := fe.String()
						newStr.Elem().SetString(s[:2] + "****")
					} else {
						s := fe.String()
						newStr.Elem().SetString(s[:2] + "****" + s[len(s)-2:])
					}

					f.Set(newStr)
				} else {
					f.Set(reflect.Zero(f.Type()))
				}
			case reflect.String:
				if f.Len() < 4 {
					f.SetString("****")
				} else if f.Len() < 8 {
					s := f.String()
					f.SetString(s[:2] + "****")
				} else {
					s := f.String()
					f.SetString(s[:2] + "****" + s[len(s)-2:])
				}
			}
		case "":
		}
	}

	return rv.Interface()
}
