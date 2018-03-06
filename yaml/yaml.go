package yaml

import "bytes"

func Chunk(in []byte) [][]byte {
	return bytes.Split(in, []byte("\n---\n"))
}

func IsBasicType(v interface{}) bool {
	// https://github.com/ghodss/yaml/blob/master/yaml.go#L130
	switch v.(type) {
	case nil, string, bool, int, int64, float64, uint64:
		return true
	default:
		return false
	}
}
