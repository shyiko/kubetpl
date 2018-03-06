package tpl

import (
	"fmt"
	"strings"
)

var kubetplDataFromFile = "kubetpl/data-from-file"

func ReplaceDataFromFileInPlace(obj map[interface{}]interface{}, load func(file string) (string, []byte, error)) (bool, error) {
	if obj["kind"] != "ConfigMap" && obj["kind"] != "Secret" { // todo: error?
		return false, nil
	}
	if obj[kubetplDataFromFile] == nil {
		return false, nil
	}
	var entries []string
	switch v := obj[kubetplDataFromFile].(type) {
	case string:
		entries = []string{v}
	case []interface{}:
		for _, v0 := range v {
			entries = append(entries, v0.(string)) // fixme: panic
		}
	default:
		return false, fmt.Errorf("%s expects a list of strings", kubetplDataFromFile)
	}
	var r []fileEntry
	for _, e := range entries {
		split := strings.Split(e, "=")
		if len(split) == 1 {
			split = []string{"", split[0]}
		}
		//split := append(strings.Split(e, "="), e)
		key, value := strings.TrimSpace(split[0]), strings.TrimSpace(split[1])
		if value != "" {
			r = append(r, fileEntry{key, value})
		}
	}
	data, ok := obj["data"].(map[interface{}]interface{})
	if !ok {
		data = make(map[interface{}]interface{})
		obj["data"] = data
	}
	for _, e := range r {
		defkey, value, err := load(e.value)
		if err != nil {
			return false, err
		}
		key := e.key
		if key == "" {
			key = defkey
		}
		data[key] = string(value)
	}
	delete(obj, kubetplDataFromFile)
	return len(r) != 0, nil
}

type fileEntry struct {
	key, value string
}
