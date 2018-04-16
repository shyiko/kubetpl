package processor

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"strings"
)

type kind = string

const (
	kindConfigMap             = "ConfigMap"
	kindSecret                = "Secret"
	kindPod                   = "Pod"
	kindDaemonSet             = "DaemonSet"
	kindDeployment            = "Deployment"
	kindJob                   = "Job"
	kindReplicaSet            = "ReplicaSet"
	kindReplicationController = "ReplicationController"
	kindStatefulSet           = "StatefulSet"
	kindCronJob               = "CronJob"
)

type frozenObjectRef struct {
	kind        kind
	name        string
	updatedName string
}

// kindConfigMap/kindSecret -> kind* -> []path
var pathsToRewrite map[kind]map[kind][]string

func init() {
	configMap := make(map[kind][]string)
	secret := make(map[kind][]string)
	configMap[kindPod] = []string{
		"spec.volumes[*].configMap.name",
	}
	secret[kindPod] = []string{
		"spec.containers[*].env.valueFrom.secretKeyRef.name",
	}
	for _, kind := range []string{
		kindDaemonSet,
		kindDeployment,
		kindJob,
		kindReplicaSet,
		kindReplicationController,
		kindStatefulSet,
	} {
		configMap[kind] = []string{
			"spec.template.spec.volumes[*].configMap.name",
		}
		secret[kind] = []string{
			"spec.template.spec.containers[*].env.valueFrom.secretKeyRef.name",
		}
	}
	configMap[kindCronJob] = []string{
		"spec.jobTemplate.spec.template.spec.volumes[*].configMap.name",
	}
	secret[kindCronJob] = []string{
		"spec.jobTemplate.spec.template.spec.containers[*].env.valueFrom.secretKeyRef.name",
	}
	pathsToRewrite = map[kind]map[kind][]string{
		kindConfigMap: configMap,
		kindSecret:    secret,
	}
}

type FreezeRequest struct {
	Docs    []map[interface{}]interface{}
	Refs    []map[interface{}]interface{}
	Include []string
}

func FreezeInPlace(r FreezeRequest) error {
	var refs []frozenObjectRef
	for _, obj := range append(append([]map[interface{}]interface{}{}, r.Refs...), r.Docs...) {
		if err := validate(obj); err != nil {
			return err
		}
		kind := obj["kind"].(string)
		meta := obj["metadata"].(map[interface{}]interface{})
		name := meta["name"].(string)
		if kind != kindConfigMap && kind != kindSecret {
			continue
		}
		if r.Include != nil && !contains(r.Include, kind+"/"+name) {
			continue
		}
		ref, err := freeze(obj)
		if err != nil {
			return err
		}
		refs = append(refs, ref)
	}
	// making sure all requested kind/name pairs were found
nextAssertion:
	for _, assertion := range r.Include {
		split := strings.SplitN(assertion, "/", 2)
		if len(split) != 2 {
			return fmt.Errorf(`"%s" is not a valid assertion`, assertion)
		}
		kind, name := split[0], split[1]
		for _, ref := range refs {
			if ref.kind == kind && ref.name == name {
				continue nextAssertion
			}
		}
		return fmt.Errorf(`"%s" not found`, assertion)
	}
	// checking for duplicates
	index := make(map[string]bool)
	for _, ref := range refs {
		k := ref.kind + "/" + ref.name
		if index[k] {
			return fmt.Errorf(`Multiple "%s"s found`, k)
		}
		index[k] = true
	}
	// rewriting refs (up until this moment nothing should not have been mutated)
	for _, obj := range r.Docs {
		kind := obj["kind"].(string)
		if kind == kindConfigMap || kind == kindSecret {
			meta := obj["metadata"].(map[interface{}]interface{})
			name := meta["name"].(string)
			for _, ref := range refs {
				if kind == ref.kind && name == ref.name {
					meta["name"] = ref.updatedName
					break
				}
			}
		} else {
			for _, ref := range refs {
				updateRefs(obj, ref)
			}
		}
	}
	return nil
}

func contains(s []string, v string) bool {
	for _, a := range s {
		if a == v {
			return true
		}
	}
	return false
}

func validate(obj map[interface{}]interface{}) error {
	kind, ok := obj["kind"].(string)
	if !ok {
		return errors.New("Malformed object (missing/invalid kind)")
	}
	meta, ok := obj["metadata"].(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf(`Malformed "%s" object (missing/invalid metadata)`, kind)
	}
	_, ok = meta["name"].(string)
	if !ok {
		return fmt.Errorf(`Malformed "%s" object (missing/invalid metadata.name)`, kind)
	}
	return nil
}

func freeze(obj map[interface{}]interface{}) (frozenObjectRef, error) {
	kind := obj["kind"].(string)
	meta := obj["metadata"].(map[interface{}]interface{})
	name := meta["name"].(string)
	snapshot, err := yaml.Marshal(obj)
	if err != nil {
		return frozenObjectRef{}, err
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(snapshot))
	updatedName := fmt.Sprintf("%s-%s", name, sum[:7])
	return frozenObjectRef{kind: kind, name: name, updatedName: updatedName}, nil
}

func updateRefs(obj map[interface{}]interface{}, ref frozenObjectRef) {
	rules, ok := pathsToRewrite[ref.kind]
	if !ok {
		return
	}
	kind := obj["kind"].(string)
	paths, ok := rules[kind]
	if !ok {
		return
	}
	for _, path := range paths {
		d := strings.LastIndex(path, ".")
		last := path[d+1:]
		rr := []interface{}{obj}
		for _, p := range strings.Split(path[0:d], "[*].") {
			var rn []interface{}
			for _, r := range rr {
				m, ok := r.(map[interface{}]interface{})
				if !ok {
					continue
				}
				n := get(m, strings.Split(p, "."))
				if n != nil {
					if slice, ok := n.([]interface{}); ok {
						rn = append(rn, slice...)
					} else {
						rn = append(rn, n)
					}
				}
			}
			rr = rn
		}
		for _, r := range rr {
			m, ok := r.(map[interface{}]interface{})
			if ok && m[last] == ref.name {
				m[last] = ref.updatedName
			}
		}
	}
}

func get(m map[interface{}]interface{}, path []string) interface{} {
	r := m
	var ok bool
	e := len(path) - 1
	for _, p := range path[0:e] {
		if r, ok = r[p].(map[interface{}]interface{}); !ok {
			return nil
		}
	}
	return r[path[e]]
}
