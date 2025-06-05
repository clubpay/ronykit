package flow

import (
	"net/http"

	"go.temporal.io/sdk/converter"
)

func EncryptedDataConverter() converter.DataConverter {
	return converter.NewCodecDataConverter(
		converter.GetDefaultDataConverter(),
		converter.NewZlibCodec(converter.ZlibCodecOptions{}),
		converter.NewRemotePayloadCodec(converter.RemotePayloadCodecOptions{
			Endpoint:      "",
			ModifyRequest: nil,
			Client:        http.Client{},
		}),
	)
}
