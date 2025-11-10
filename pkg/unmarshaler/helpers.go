package unmarshaler

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func getYamlNodeComments(n yaml.Node, commentPrefix string) []string {
	comments := []string{}
	for _, line := range strings.Split(n.HeadComment, "\n") {
		if !strings.Contains(line, commentPrefix) {
			continue
		}
		comments = append(comments, line)
	}
	return comments
}

func getExpressionComments(expr, commentPrefix string) []string {
	comments := []string{}
	for _, line := range strings.Split(expr, "\n") {
		before, comment, found := strings.Cut(line, "#")
		if !found || strings.TrimSpace(before) != "" {
			continue
		}
		if !strings.Contains(comment, commentPrefix) {
			continue
		}
		comments = append(comments, comment)
	}
	return comments
}

func disabledValidatorsFromComments(comments []string, commentPrefix string) []string {
	commentPrefix += ":"
	disabledValidators := []string{}
	for _, comment := range comments {
		_, csv, found := strings.Cut(comment, commentPrefix)
		if !found {
			continue
		}
		validators := strings.Split(csv, ",")
		for _, v := range validators {
			vv := strings.TrimSpace(v)
			disabledValidators = append(disabledValidators, vv)
		}
	}
	return disabledValidators
}

func unmarshalToNodeAndStruct(value, dstNode *yaml.Node, dstStruct interface{}, knownFields []string) error {
	if t := reflect.TypeOf(dstStruct); t == nil || t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("BUG: dstStruct is not a pointer to a struct: %T", dstStruct)
	}
	// Since yaml/v3 Node.Decode doesn't support setting decode options like KnownFields (see https://github.com/go-yaml/yaml/issues/460)
	// we need to check the fields manually, thus the function requires a list of known fields.
	switch {
	case value.Kind == yaml.MappingNode:
		m := map[string]any{}
		if err := value.Decode(m); err != nil {
			return err
		}
		for k := range m {
			if !slices.Contains(knownFields, k) {
				if knownFields == nil {
					// Make the error message more readable:
					knownFields = []string{}
				}
				return fmt.Errorf("line %d: unknown field %q when unmarshaling into %T, supported fields are: %q", value.Line, k, dstStruct, knownFields)
			}
		}
	case value.Kind == yaml.DocumentNode && len(value.Content) == 1:
		return unmarshalToNodeAndStruct(value.Content[0], dstNode, dstStruct, knownFields)
	case value.IsZero():
		// ok, empty input
	case value.Kind == yaml.ScalarNode && value.ShortTag() == "!!null":
		// ok, literal null or nothing
	default:
		b, err := yaml.Marshal(value)
		return errors.Join(err, fmt.Errorf("not a YAML mapping on line %d: %q", value.Line, b))
	}
	if dstNode != nil {
		err := value.Decode(dstNode)
		if err != nil {
			return err
		}
	}
	return value.Decode(dstStruct)
}

// mustListStructYamlFieldNames returns a list of yaml field names for the given struct.
// Fields that have the "omitempty" option in their yaml tag are never returned.
func mustListStructYamlFieldNames(s interface{}, ignoreFields []string) []string {
	y, err := yaml.Marshal(s)
	if err != nil {
		fmt.Println("failed to marshal", err)
		panic(err)
	}
	m := map[string]any{}
	if err := yaml.Unmarshal(y, m); err != nil {
		fmt.Println("failed to unmarshal", err)
		panic(err)
	}
	names := []string{}
	for k := range m {
		if slices.Contains(ignoreFields, k) {
			continue
		}
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
