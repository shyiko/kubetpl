package processor

import (
	"encoding/base64"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"strings"
)

var kubetplDataFromFile = "kubetpl/data-from-file"

func ReplaceDataFromFileInPlace(
	obj map[interface{}]interface{},
	read func(file string) (string, []byte, error),
) (bool, error) {
	if obj["kind"] != "ConfigMap" && obj["kind"] != "Secret" {
		return false, nil
	}
	if obj[kubetplDataFromFile] == nil {
		return false, nil
	}
	var entries []string
	switch v := obj[kubetplDataFromFile].(type) {
	case []interface{}:
		for _, entry := range v {
			entries = append(entries, fmt.Sprintf("%v", entry))
		}
	default:
		entries = []string{fmt.Sprintf("%v", v)}
	}
	var r []fileEntry
	for _, entry := range entries {
		if entry == "" {
			continue
		}
		split := strings.SplitN(entry, "=", 2)
		if len(split) == 1 {
			split = []string{"", split[0]}
		}
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
		log.Debugf(`kubetpl/data-from-file: loading %s`, e.value)
		key, value, err := read(e.value)
		if err != nil {
			return false, err
		}
		if e.key != "" { // key override
			key = e.key
		}
		if obj["kind"] == "Secret" {
			data[key] = base64.StdEncoding.EncodeToString(value)
		} else {
			data[key] = string(value)
		}
	}
	delete(obj, kubetplDataFromFile)
	return len(r) != 0, nil
}

type fileEntry struct {
	key, value string
}
