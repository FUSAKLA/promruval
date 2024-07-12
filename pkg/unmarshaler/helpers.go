package unmarshaler

import (
	"fmt"
	"slices"
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
	// Since yaml/v3 Node.Decode doesn't support setting decode options like KnownFields (see https://github.com/go-yaml/yaml/issues/460)
	// we need to check the fields manually, thus the function requires a list of known fields.
	if value.Kind == yaml.MappingNode {
		m := map[string]any{}
		if err := value.Decode(m); err != nil {
			return err
		}
		for k := range m {
			if !slices.Contains(knownFields, k) {
				return fmt.Errorf("unknown field %q when unmarshalling the %T, only supported fields are: %s", k, dstStruct, strings.Join(knownFields, ","))
			}
		}
	}
	err := value.Decode(dstNode)
	if err != nil {
		return err
	}
	err = value.Decode(dstStruct)
	if err != nil {
		return err
	}
	return nil
}

// mustListStructYamlFieldNames returns a list of yaml field names for the given struct.
func mustListStructYamlFieldNames(s interface{}) []string {
	y, err := yaml.Marshal(s)
	if err != nil {
		fmt.Println("failed to marshal", err)
		panic(err)
	}
	m := map[string]any{}
	if err := yaml.Unmarshal(y, m); err != nil {
		fmt.Println("failed to marshal", err)
		panic(err)
	}
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	return names
}
