package config

import (
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

func checkYAMLNode(node *yaml.Node, path string, t reflect.Type, errs *Errors) {
	if node == nil {
		return
	}
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) > 0 {
			checkYAMLNode(node.Content[0], path, t, errs)
		}
	case yaml.MappingNode:
		checkYAMLMapping(node, path, t, errs)
	case yaml.SequenceNode:
		checkYAMLSequence(node, path, t, errs)
	}
}

func checkYAMLMapping(node *yaml.Node, path string, t reflect.Type, errs *Errors) {
	t = deref(t)
	seen := make(map[string]int)

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		key := keyNode.Value

		if prevLine, ok := seen[key]; ok {
			errs.Add(fmt.Errorf("%s: duplicate key %q (first at line %d, again at line %d)",
				path, key, prevLine, keyNode.Line))
		}
		seen[key] = keyNode.Line

		if t != nil && t.Kind() == reflect.Struct {
			fields := yamlFieldTypes(t)
			fieldType, ok := fields[key]
			if !ok {
				errs.Add(fmt.Errorf("%s: unknown field %q at line %d", path, key, keyNode.Line))
				continue
			}
			checkYAMLNode(valNode, joinPath(path, key), fieldType, errs)
		} else if t != nil && t.Kind() == reflect.Map {
			checkYAMLNode(valNode, joinPath(path, key), t.Elem(), errs)
		}
	}
}

func checkYAMLSequence(node *yaml.Node, path string, t reflect.Type, errs *Errors) {
	t = deref(t)
	var elemType reflect.Type
	if t != nil && t.Kind() == reflect.Slice {
		elemType = t.Elem()
	}
	for i, item := range node.Content {
		checkYAMLNode(item, fmt.Sprintf("%s[%d]", path, i), elemType, errs)
	}
}

func yamlFieldTypes(t reflect.Type) map[string]reflect.Type {
	t = deref(t)
	fields := make(map[string]reflect.Type)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("yaml")
		if tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			name = strings.ToLower(f.Name)
		}
		fields[name] = f.Type
	}
	return fields
}

func deref(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func joinPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}
