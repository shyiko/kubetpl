package processor

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

type kind = string

const (
	kindConfigMap             = "ConfigMap"
	kindSecret                = "Secret"
	kindPod                   = "Pod"
	kindPodPreset             = "PodPreset"
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
		"spec.initContainers[*].env[*].valueFrom.configMapKeyRef.name",
		"spec.initContainers[*].envFrom[*].configMapRef.name",
		"spec.containers[*].env[*].valueFrom.configMapKeyRef.name",
		"spec.containers[*].envFrom[*].configMapRef.name",
		"spec.volumes[*].configMap.name",
	}
	secret[kindPod] = []string{
		"spec.initContainers[*].env[*].valueFrom.secretKeyRef.name",
		"spec.initContainers[*].envFrom[*].secretRef.name",
		"spec.containers[*].env[*].valueFrom.secretKeyRef.name",
		"spec.containers[*].envFrom[*].secretRef.name",
		"spec.volumes[*].secret.secretName",
	}
	configMap[kindPodPreset] = []string{
		"spec.env[*].valueFrom.configMapKeyRef.name",
		"spec.envFrom[*].configMapRef.name",
		"spec.volumes[*].configMap.name",
	}
	secret[kindPodPreset] = []string{
		"spec.env[*].valueFrom.secretKeyRef.name",
		"spec.envFrom[*].secretRef.name",
		"spec.volumes[*].secret.secretName",
	}
	for _, kind := range []string{
		kindDaemonSet,
		kindDeployment,
		kindJob,
		kindReplicaSet,
		kindReplicationController,
		kindStatefulSet,
	} {
		configMap[kind] = mapWithPrefix(configMap[kindPod], "spec.template.")
		secret[kind] = mapWithPrefix(secret[kindPod], "spec.template.")
	}
	configMap[kindCronJob] = mapWithPrefix(configMap[kindPod], "spec.jobTemplate.spec.template.")
	secret[kindCronJob] = mapWithPrefix(secret[kindPod], "spec.jobTemplate.spec.template.")
	pathsToRewrite = map[kind]map[kind][]string{
		kindConfigMap: configMap,
		kindSecret:    secret,
	}
}

func mapWithPrefix(slice []string, prefix string) []string {
	var s []string
	for _, p := range slice {
		s = append(s, prefix+p)
	}
	return s
}

type FreezeRequest struct {
	Docs    []map[interface{}]interface{}
	Refs    []map[interface{}]interface{}
	Include []string
}

func FreezeInPlace(r FreezeRequest) error {
	var refs []frozenObjectRef
	var includeIndex map[string]bool
	if r.Include != nil {
		includeIndex = make(map[string]bool)
		for _, include := range r.Include {
			includeIndex[include] = true
		}
	}
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
		if includeIndex != nil && !includeIndex[kind+"/"+name] {
			continue
		}
		ref, err := freeze(obj)
		if err != nil {
			return err
		}
		log.Debugf("freeze: freezing %s/%s as %s/%s", ref.kind, ref.name, ref.kind, ref.updatedName)
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
	refIndex := make(map[string]bool)
	for _, ref := range refs {
		k := ref.kind + "/" + ref.name
		if refIndex[k] {
			return fmt.Errorf(`Multiple "%s"s found`, k)
		}
		refIndex[k] = true
	}
	// making sure no ConfigMap/Secret evades freezing (subject to FreezeRequest.Include)
	for _, obj := range r.Docs {
		kind := obj["kind"].(string)
		if kind != kindConfigMap && kind != kindSecret {
			meta := obj["metadata"].(map[interface{}]interface{})
			name := meta["name"].(string)
			for _, ref := range refs {
				if err := traverseRefs(obj, ref, func(node map[interface{}]interface{}, key string, path string) error {
					if v, ok := node[key].(string); ok {
						key := ref.kind + "/" + v
						if !refIndex[key] && (includeIndex == nil || includeIndex[key]) {
							return fmt.Errorf(`Stumbled upon unknown %s reference (in %s/%s).`+
								"\nHave you forgot to --freeze-ref it?"+
								"\n(if --freeze-ref is pointing to a template - "+
								"check that \"# kubetpl:syntax:<template flavor, e.g. $>\" is present)", key, kind, name)
						}
					}
					return nil
				}); err != nil {
					return err
				}
			}
		}
	}
	// rewriting refs (up until this moment nothing should have been mutated)
	for _, obj := range r.Docs {
		kind := obj["kind"].(string)
		meta := obj["metadata"].(map[interface{}]interface{})
		name := meta["name"].(string)
		if kind == kindConfigMap || kind == kindSecret {
			for _, ref := range refs {
				if kind == ref.kind && name == ref.name {
					meta["name"] = ref.updatedName
					break
				}
			}
		} else {
			for _, ref := range refs {
				traverseRefs(obj, ref, func(node map[interface{}]interface{}, key string, path string) error {
					if node[key] == ref.name {
						log.Debugf(`freeze: rewriting %s to %s (%s in %s/%s)`, ref.name, ref.updatedName, path, kind, name)
						node[key] = ref.updatedName
					}
					return nil
				})
			}
		}
	}
	return nil
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

func traverseRefs(
	obj map[interface{}]interface{},
	ref frozenObjectRef,
	cb func(node map[interface{}]interface{}, key string, path string) error,
) error {
	rules, ok := pathsToRewrite[ref.kind]
	if !ok {
		return nil
	}
	kind := obj["kind"].(string)
	paths, ok := rules[kind]
	if !ok {
		return nil
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
			if m, ok := r.(map[interface{}]interface{}); ok {
				if err := cb(m, last, path); err != nil {
					return err
				}
			}
		}
	}
	return nil
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
