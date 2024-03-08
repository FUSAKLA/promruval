package unmarshaler

import (
	"slices"
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
	Groups       GroupsWithComment `yaml:"groups"`
	fakeTestFile                   // Just so we can unmarshal also PromQL test files but ignore them because it has no Groups
}

type RulesFileWithComment struct {
	node           yaml.Node
	groupsComments []string
	RulesFile
}

func (r *RulesFileWithComment) UnmarshalYAML(value *yaml.Node) error {
	for _, field := range value.Content {
		if field.Kind == yaml.ScalarNode && field.Value == "groups" {
			r.groupsComments = strings.Split(field.HeadComment, "\n")
		}
	}
	return unmarshalToNodeAndStruct(value, &r.node, &r.RulesFile)
}

func (r *RulesFileWithComment) DisabledValidators(commentPrefix string) []string {
	return disabledValidatorsFromComments(slices.Concat(getYamlNodeComments(r.node, commentPrefix), r.groupsComments), commentPrefix)
}

type GroupsWithComment struct {
	node   yaml.Node
	Groups []RuleGroupWithComment
}

func (g *GroupsWithComment) UnmarshalYAML(value *yaml.Node) error {
	return unmarshalToNodeAndStruct(value, &g.node, &g.Groups)
}

func (g *GroupsWithComment) DisabledValidators(commentPrefix string) []string {
	return disabledValidatorsFromComments(getYamlNodeComments(g.node, commentPrefix), commentPrefix)
}

type RuleGroup struct {
	Name                    string            `yaml:"name"`
	Interval                model.Duration    `yaml:"interval,omitempty"`
	PartialResponseStrategy string            `yaml:"partial_response_strategy,omitempty"`
	SourceTenants           []string          `yaml:"source_tenants,omitempty"`
	Rules                   []RuleWithComment `yaml:"rules"`
	Limit                   int               `yaml:"limit,omitempty"`
}

type RuleGroupWithComment struct {
	node yaml.Node
	RuleGroup
}

func (r *RuleGroupWithComment) UnmarshalYAML(value *yaml.Node) error {
	return unmarshalToNodeAndStruct(value, &r.node, &r.RuleGroup)
}

func (r *RuleGroupWithComment) DisabledValidators(commentPrefix string) []string {
	return disabledValidatorsFromComments(getYamlNodeComments(r.node, commentPrefix), commentPrefix)
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
	return unmarshalToNodeAndStruct(value, &r.node, &r.rule)
}

func (r *RuleWithComment) DisabledValidators(commentPrefix string) []string {
	ruleComments := getYamlNodeComments(r.node, commentPrefix)
	exprComments := getExpressionComments(r.rule.Expr.Value, commentPrefix)
	return disabledValidatorsFromComments(slices.Concat(ruleComments, exprComments), commentPrefix)
}
