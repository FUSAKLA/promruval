package unmarshaler

import (
	"slices"
	"strings"

	loki "github.com/grafana/loki/v3/pkg/tool/rules/rwrulefmt"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"gopkg.in/yaml.v3"
)

var (
	// Struct fields marked as omitempty MUST be set to non-default value so they appear in marshalled yaml.
	rulesFileKnownFields         = mustListStructYamlFieldNames(RulesFile{})
	groupsWithCommentKnownFields = mustListStructYamlFieldNames(GroupsWithComment{})
	ruleGroupKnownFields         = mustListStructYamlFieldNames(RuleGroup{})
	ruleNodeKnownFields          = mustListStructYamlFieldNames(rulefmt.RuleNode{Record: yaml.Node{Kind: yaml.SequenceNode}, Alert: yaml.Node{Kind: yaml.SequenceNode}, For: model.Duration(1), Labels: map[string]string{"foo": "bar"}, Annotations: map[string]string{"foo": "bar"}, KeepFiringFor: model.Duration(1)})
)

type RulesFile struct {
	Groups GroupsWithComment `yaml:"groups"`
	// Just so we can unmarshal also PromQL test files but ignore them because it has no Groups
	RuleFiles          interface{} `yaml:"rule_files"`
	EvaluationInterval interface{} `yaml:"evaluation_interval"`
	GroupEvalOrder     interface{} `yaml:"group_eval_order"`
	Tests              interface{} `yaml:"tests"`
	// Loki only
	Namespace string `yaml:"namespace"`
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
	return unmarshalToNodeAndStruct(value, &r.node, &r.RulesFile, rulesFileKnownFields)
}

func (r *RulesFileWithComment) DisabledValidators(commentPrefix string) []string {
	return disabledValidatorsFromComments(slices.Concat(getYamlNodeComments(r.node, commentPrefix), r.groupsComments), commentPrefix)
}

type GroupsWithComment struct {
	node   yaml.Node
	Groups []RuleGroupWithComment `yaml:"groups"`
}

func (g *GroupsWithComment) UnmarshalYAML(value *yaml.Node) error {
	return unmarshalToNodeAndStruct(value, &g.node, &g.Groups, groupsWithCommentKnownFields)
}

func (g *GroupsWithComment) DisabledValidators(commentPrefix string) []string {
	return disabledValidatorsFromComments(getYamlNodeComments(g.node, commentPrefix), commentPrefix)
}

type RuleGroup struct {
	Name        string            `yaml:"name"`
	Interval    model.Duration    `yaml:"interval"`
	QueryOffset model.Duration    `yaml:"query_offset"`
	Rules       []RuleWithComment `yaml:"rules"`
	Limit       int               `yaml:"limit"`

	// Thanos only
	PartialResponseStrategy string `yaml:"partial_response_strategy"`
	// Cortex/Mimir only
	SourceTenants []string `yaml:"source_tenants"`
	// Loki only
	RWConfigs []loki.RemoteWriteConfig `yaml:"remote_write"`
}

type RuleGroupWithComment struct {
	node yaml.Node
	RuleGroup
}

func (r *RuleGroupWithComment) UnmarshalYAML(value *yaml.Node) error {
	return unmarshalToNodeAndStruct(value, &r.node, &r.RuleGroup, ruleGroupKnownFields)
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
		Record:        r.rule.Record.Value,
		Alert:         r.rule.Alert.Value,
		Expr:          r.rule.Expr.Value,
		For:           r.rule.For,
		Labels:        r.rule.Labels,
		Annotations:   r.rule.Annotations,
		KeepFiringFor: r.rule.KeepFiringFor,
	}
}

func (r *RuleWithComment) UnmarshalYAML(value *yaml.Node) error {
	return unmarshalToNodeAndStruct(value, &r.node, &r.rule, ruleNodeKnownFields)
}

func (r *RuleWithComment) DisabledValidators(commentPrefix string) []string {
	ruleComments := getYamlNodeComments(r.node, commentPrefix)
	exprComments := getExpressionComments(r.rule.Expr.Value, commentPrefix)
	return disabledValidatorsFromComments(slices.Concat(ruleComments, exprComments), commentPrefix)
}
