package dotenv

import "gopkg.in/ini.v1"

func Parse(data []byte) (map[string]string, error) {
	f, err := ini.Load(data)
	if err != nil {
		return nil, err
	}
	section, err := f.GetSection("")
	if err != nil {
		panic(err)
	}
	return section.KeysHash(), nil
}
