package unmarshaler

import (
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

func unmarshalToNodeAndStruct(value, dstNode *yaml.Node, dstStruct interface{}) error {
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
