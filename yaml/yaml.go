package yaml

import (
	"bytes"
)

func Chunk(in []byte) [][]byte {
	return bytes.Split(in, []byte("\n---\n"))
}

func Header(in []byte) []byte {
	var r [][]byte
	for _, line := range bytes.Split(in, []byte("\n")) {
		if !bytes.HasPrefix(line, []byte("#")) && len(line) > 0 {
			break
		}
		r = append(r, line)
	}
	return bytes.Join(r, []byte("\n"))
}

func Footer(in []byte) []byte {
	var r [][]byte
	split := bytes.Split(in, []byte("\n"))
	for i := len(split) - 1; i > -1; i-- {
		line := split[i]
		if !bytes.HasPrefix(line, []byte("#")) && len(line) > 0 {
			break
		}
		r = append(r, line)
	}
	return bytes.Join(reverseInPlace(r), []byte("\n"))
}

func reverseInPlace(in [][]byte) [][]byte {
	for i, j := 0, len(in)-1; i < j; i, j = i+1, j-1 {
		in[i], in[j] = in[j], in[i]
	}
	return in
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
