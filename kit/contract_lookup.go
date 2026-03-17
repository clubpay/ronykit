package kit

import "github.com/clubpay/ronykit/kit/errors"

const contractLookupSeparator = "::"

func contractLookupKey(serviceName, contractID string) string {
	return serviceName + contractLookupSeparator + contractID
}

func resolveContract(contracts map[string]Contract, serviceName, contractID string) (Contract, error) {
	if serviceName == "" || contractID == "" {
		return nil, errors.Wrap(
			ErrContractNotFound,
			errors.New("invalid execute arguments (service=%q, contract=%q)", serviceName, contractID),
		)
	}

	c := contracts[contractLookupKey(serviceName, contractID)]
	if c == nil {
		return nil, errors.Wrap(
			ErrContractNotFound,
			errors.New("service=%q contract=%q", serviceName, contractID),
		)
	}

	return c, nil
}
