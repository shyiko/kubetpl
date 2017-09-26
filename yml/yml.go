package yml

import "bytes"

func UnmarshalSlice(in []byte, cb func(in []byte) error) error {
	chunks := bytes.Split(in, []byte("\n---\n"))
	for _, chunk := range chunks {
		if err := cb(chunk); err != nil {
			return err
		}
	}
	return nil
}
