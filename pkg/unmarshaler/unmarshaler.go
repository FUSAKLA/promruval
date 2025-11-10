package unmarshaler

import (
	"slices"
	"strings"

	loki "github.com/grafana/loki/v3/pkg/tool/rules/rwrulefmt"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"gopkg.in/yaml.v3"

	"github.com/fusakla/promruval/v3/pkg/config"
)

var (
	supportLoki   = false
	supportMimir  = false
	supportThanos = false
)

func SupportLoki(support bool) {
	supportLoki = support
}

func SupportMimir(support bool) {
	supportMimir = support
}

func SupportThanos(support bool) {
	supportThanos = support
}

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

func (r *RulesFile) knownFields() []string {
	var ignoredFields []string
	if !supportLoki {
		ignoredFields = append(ignoredFields, "namespace")
	}
	return mustListStructYamlFieldNames(r, ignoredFields)
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
	return unmarshalToNodeAndStruct(value, &r.node, &r.RulesFile, r.RulesFile.knownFields()) //nolint:staticcheck // must be called on the RuleFile so the yaml marshalling works
}

func (r *RulesFileWithComment) DisabledValidators(commentPrefix string) []string {
	return disabledValidatorsFromComments(slices.Concat(getYamlNodeComments(r.node, commentPrefix), r.groupsComments), commentPrefix)
}

type GroupsWithComment struct {
	node   yaml.Node
	Groups []RuleGroupWithComment `yaml:"groups"`
}

func (g *GroupsWithComment) UnmarshalYAML(value *yaml.Node) error {
	return unmarshalToNodeAndStruct(value, &g.node, &g.Groups, mustListStructYamlFieldNames(g, []string{}))
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

func (r *RuleGroup) knownFields() []string {
	var ignoredFields []string
	if !supportLoki {
		ignoredFields = append(ignoredFields, "remote_write")
	}
	if !supportThanos {
		ignoredFields = append(ignoredFields, "partial_response_strategy")
	}
	if !supportMimir {
		ignoredFields = append(ignoredFields, "source_tenants")
	}
	return mustListStructYamlFieldNames(r, ignoredFields)
}

type RuleGroupWithComment struct {
	node yaml.Node
	RuleGroup
}

func (r *RuleGroupWithComment) UnmarshalYAML(value *yaml.Node) error {
	//nolint:staticcheck // the knownFields must be called on the RuleGroup not the RuleGroupWithComment
	return unmarshalToNodeAndStruct(value, &r.node, &r.RuleGroup, r.RuleGroup.knownFields())
}

func (r *RuleGroupWithComment) DisabledValidators(commentPrefix string) []string {
	return disabledValidatorsFromComments(getYamlNodeComments(r.node, commentPrefix), commentPrefix)
}

type RuleWithComment struct {
	node yaml.Node
	rule rulefmt.Rule
}

func (r *RuleWithComment) knownFields() []string {
	// Struct fields marked as omitempty MUST be set to non-default value so they appear in marshalled yaml.
	return mustListStructYamlFieldNames(rulefmt.Rule{Record: "foo", Alert: "bar", For: model.Duration(1), Labels: map[string]string{"foo": "bar"}, Annotations: map[string]string{"foo": "bar"}, KeepFiringFor: model.Duration(1)}, []string{})
}

func (r *RuleWithComment) OriginalRule() rulefmt.Rule {
	return rulefmt.Rule{
		Record:        r.rule.Record,
		Alert:         r.rule.Alert,
		Expr:          r.rule.Expr,
		For:           r.rule.For,
		Labels:        r.rule.Labels,
		Annotations:   r.rule.Annotations,
		KeepFiringFor: r.rule.KeepFiringFor,
	}
}

func (r *RuleWithComment) Scope() config.ValidationScope {
	if r.rule.Alert != "" {
		return config.AlertScope
	}
	return config.RecordingRuleScope
}

func (r *RuleWithComment) UnmarshalYAML(value *yaml.Node) error {
	return unmarshalToNodeAndStruct(value, &r.node, &r.rule, r.knownFields())
}

func (r *RuleWithComment) DisabledValidators(commentPrefix string) []string {
	ruleComments := getYamlNodeComments(r.node, commentPrefix)
	exprComments := getExpressionComments(r.rule.Expr, commentPrefix)
	return disabledValidatorsFromComments(slices.Concat(ruleComments, exprComments), commentPrefix)
}
