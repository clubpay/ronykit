package flow

import (
	"github.com/clubpay/ronykit/flow/internal/scramble"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"golang.org/x/sync/errgroup"
)

func EncryptedDataConverter(key string) converter.DataConverter {
	return converter.NewCodecDataConverter(
		converter.GetDefaultDataConverter(),
		converter.NewZlibCodec(converter.ZlibCodecOptions{}),
		&aesCodec{s: scramble.NewScramble(key)},
	)
}

var _ converter.PayloadCodec = (*aesCodec)(nil)

type aesCodec struct {
	s *scramble.Scramble
}

func (a aesCodec) Encode(payloads []*common.Payload) ([]*common.Payload, error) {
	output := make([]*common.Payload, len(payloads))
	errG := &errgroup.Group{}
	for i := range payloads {
		errG.Go(func(idx int) func() error {
			return func() error {
				p := &common.Payload{
					Data:     a.s.Encrypt(payloads[idx].Data, nil),
					Metadata: make(map[string][]byte, len(payloads[idx].Metadata)),
				}
				for k, v := range payloads[i].Metadata {
					p.Metadata[k] = a.s.Encrypt(v, nil)
				}
				output[idx] = p

				return nil
			}
		}(i))
	}
	err := errG.Wait()
	if err != nil {
		return nil, err
	}

	return output, nil
}

func (a aesCodec) Decode(payloads []*common.Payload) ([]*common.Payload, error) {
	output := make([]*common.Payload, len(payloads))
	errG := &errgroup.Group{}
	for i := range payloads {
		errG.Go(func(idx int) func() error {
			return func() error {
				d, err := a.s.Decrypt(payloads[idx].Data, nil)
				if err != nil {
					output[idx] = payloads[idx]

					return nil
				}
				p := &common.Payload{
					Data:     d,
					Metadata: make(map[string][]byte, len(payloads[idx].Metadata)),
				}
				for k, v := range payloads[i].Metadata {
					d, err = a.s.Decrypt(v, nil)
					if err != nil {
						return err
					}
					p.Metadata[k] = d
				}
				output[idx] = p

				return nil
			}
		}(i))
	}
	err := errG.Wait()
	if err != nil {
		return nil, err
	}

	return output, nil
}
