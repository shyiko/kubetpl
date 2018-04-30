package processor

import (
	"encoding/base64"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/shyiko/kubetpl/dotenv"
	"strings"
)

var kubetplDataFromFile = "kubetpl/data-from-file"
var kubetplDataFromEnvFile = "kubetpl/data-from-env-file"

func ReplaceDataFromFileInPlace(
	obj map[interface{}]interface{},
	read func(file string) (string, []byte, error),
) (bool, error) {
	if obj["kind"] != "ConfigMap" && obj["kind"] != "Secret" {
		return false, nil
	}
	fromFile := sliceFileEntries(obj, kubetplDataFromFile)
	fromEnvFile := slice(obj, kubetplDataFromEnvFile)
	if len(fromFile) == 0 && len(fromEnvFile) == 0 {
		return false, nil
	}
	if len(fromFile) != 0 && len(fromEnvFile) != 0 {
		return false, fmt.Errorf("%s cannot be combined with %s", kubetplDataFromFile, kubetplDataFromEnvFile)
	}
	data, ok := obj["data"].(map[interface{}]interface{})
	if !ok {
		data = make(map[interface{}]interface{})
		obj["data"] = data
	}
	for _, e := range fromFile {
		log.Debugf(`%s: loading %s`, kubetplDataFromFile, e.value)
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
	for _, e := range fromEnvFile {
		log.Debugf(`%s: loading %s`, kubetplDataFromEnvFile, e)
		_, value, err := read(e)
		if err != nil {
			return false, err
		}
		env, err := dotenv.Parse(value)
		if err != nil {
			return false, fmt.Errorf("%s: %s", e, err.Error())
		}
		for key, value := range env {
			if obj["kind"] == "Secret" {
				data[key] = base64.StdEncoding.EncodeToString([]byte(value))
			} else {
				data[key] = string(value)
			}
		}
	}
	delete(obj, kubetplDataFromFile)
	delete(obj, kubetplDataFromEnvFile)
	return true, nil
}

func sliceFileEntries(obj map[interface{}]interface{}, key string) []fileEntry {
	var r []fileEntry
	for _, entry := range slice(obj, key) {
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
	return r
}

func slice(obj map[interface{}]interface{}, key string) []string {
	if obj[key] == nil {
		return nil
	}
	var r []string
	switch v := obj[key].(type) {
	case []interface{}:
		for _, entry := range v {
			r = append(r, fmt.Sprintf("%v", entry))
		}
	default:
		r = []string{fmt.Sprintf("%v", v)}
	}
	return r
}

type fileEntry struct {
	key, value string
}
