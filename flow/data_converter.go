package flow

import (
	"github.com/clubpay/ronykit/flow/internal/scramble"

	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

func EncryptedDataConverter(key string) converter.DataConverter {
	return converter.NewCodecDataConverter(
		converter.GetDefaultDataConverter(),
		converter.NewZlibCodec(converter.ZlibCodecOptions{}),
		EncryptedPayloadCodec(key),
	)
}

func EncryptedPayloadCodec(key string) converter.PayloadCodec {
	return &aesCodec{s: scramble.MustNewScramble(key)}
}

var _ converter.PayloadCodec = (*aesCodec)(nil)

type aesCodec struct {
	s *scramble.Scramble
}

func (a aesCodec) Encode(payloads []*common.Payload) ([]*common.Payload, error) {
	output := make([]*common.Payload, len(payloads))
	for i, in := range payloads {
		data, err := a.s.Encrypt(in.GetData(), nil)
		if err != nil {
			return nil, err
		}

		p := &common.Payload{
			Data:     data,
			Metadata: make(map[string][]byte, len(in.GetMetadata())),
		}
		for k, v := range in.GetMetadata() {
			enc, err := a.s.Encrypt(v, nil)
			if err != nil {
				return nil, err
			}

			p.Metadata[k] = enc
		}

		output[i] = p
	}

	return output, nil
}

// Decode decrypts the payloads. A payload this codec cannot read — plaintext history
// written before encryption was enabled, or a payload sealed with a rotated key — is
// passed through untouched rather than failing the whole batch, so one unreadable
// payload cannot stall a workflow task.
func (a aesCodec) Decode(payloads []*common.Payload) ([]*common.Payload, error) {
	output := make([]*common.Payload, len(payloads))
	for i, in := range payloads {
		p, err := a.decodeOne(in)
		if err != nil {
			output[i] = in

			continue
		}

		output[i] = p
	}

	return output, nil
}

// decodeOne is all-or-nothing: a payload whose data decrypts but whose metadata does not
// would be unusable, so the caller passes the original through instead.
func (a aesCodec) decodeOne(in *common.Payload) (*common.Payload, error) {
	d, err := a.s.Decrypt(in.GetData(), nil)
	if err != nil {
		return nil, err
	}

	p := &common.Payload{
		Data:     d,
		Metadata: make(map[string][]byte, len(in.GetMetadata())),
	}

	for k, v := range in.GetMetadata() {
		dec, err := a.s.Decrypt(v, nil)
		if err != nil {
			return nil, err
		}

		p.Metadata[k] = dec
	}

	return p, nil
}
