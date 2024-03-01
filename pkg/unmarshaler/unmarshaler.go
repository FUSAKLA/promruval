package unmarshaler

import (
	"strings"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"gopkg.in/yaml.v3"
)

type fakeTestFile struct {
	RuleFiles          []yaml.Node `yaml:"rule_files,omitempty"`
	EvaluationInterval yaml.Node   `yaml:"evaluation_interval,omitempty"`
	GroupEvalOrder     []yaml.Node `yaml:"group_eval_order,omitempty"`
	Tests              []yaml.Node `yaml:"tests,omitempty"`
}

type RulesFile struct {
	Groups       []RuleGroup `yaml:"groups"`
	fakeTestFile             // Just so we can unmarshal also PromQL test files but ignore them because it has no Groups
}

type RuleGroup struct {
	Name                    string            `yaml:"name"`
	Interval                model.Duration    `yaml:"interval,omitempty"`
	PartialResponseStrategy string            `yaml:"partial_response_strategy,omitempty"`
	SourceTenants           []string          `yaml:"source_tenants,omitempty"`
	Rules                   []RuleWithComment `yaml:"rules"`
	Limit                   int               `yaml:"limit,omitempty"`
}

type RuleWithComment struct {
	node yaml.Node
	rule rulefmt.RuleNode
}

func (r *RuleWithComment) OriginalRule() rulefmt.Rule {
	return rulefmt.Rule{
		Record:      r.rule.Record.Value,
		Alert:       r.rule.Alert.Value,
		Expr:        r.rule.Expr.Value,
		For:         r.rule.For,
		Labels:      r.rule.Labels,
		Annotations: r.rule.Annotations,
	}
}

func (r *RuleWithComment) UnmarshalYAML(value *yaml.Node) error {
	err := value.Decode(&r.node)
	if err != nil {
		return err
	}
	err = value.Decode(&r.rule)
	if err != nil {
		return err
	}
	return nil
}

func (r *RuleWithComment) DisabledValidators(commentPrefix string) []string {
	commentPrefix += ":"
	var disabledValidators []string
	allComments := strings.Split(r.node.HeadComment, "\n")
	for _, line := range strings.Split(r.rule.Expr.Value, "\n") {
		before, comment, found := strings.Cut(line, "#")
		if !found || strings.TrimSpace(before) != "" {
			continue
		}
		allComments = append(allComments, comment)
	}
	for _, comment := range allComments {
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
