package cue

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/encoding/yaml"
)

func CueEncodeYAML(value cue.Value) ([]byte, error) {
	var (
		err  error
		data []byte
	)
	switch value.Kind() {
	case cue.ListKind:
		items, err := value.List()
		if err != nil {
			return nil, err
		}
		data, err = yaml.EncodeStream(items)
		if err != nil {
			return nil, err
		}
	case cue.StructKind:
		data, err = yaml.Encode(value)
		if err != nil {
			return nil, err
		}
	default:
		return nil, nil
	}
	data = append(data, []byte("\n---\n")...)
	return data, nil
}
