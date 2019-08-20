package main

import (
	"encoding/json"

	"github.com/pkg/errors"
)

func jsonUnmarshal(input []byte, out interface{}) error {
	return errors.WithStack(json.Unmarshal(input, out))
}

// Sets indentation based on `PRETTY_JSON`. Should be used for server responses.
func jsonMarshal(input interface{}) ([]byte, error) {
	if env.conf.PrettyJson {
		bytes, err := json.MarshalIndent(input, "", PRETTY_PRINT_INDENT)
		return bytes, errors.WithStack(err)
	}
	bytes, err := json.Marshal(input)
	return bytes, errors.WithStack(err)
}

func maybeJsonMarshal(value interface{}) ([]byte, error) {
	if value == nil {
		return nil, nil
	}

	out, err := jsonMarshal(value)
	if err != nil {
		return out, errors.Errorf(`JSON marshaling error: %+v`, errors.WithStack(err))
	}

	return out, nil
}
