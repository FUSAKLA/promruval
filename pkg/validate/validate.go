package validate

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/fusakla/promruval/v2/pkg/config"
	"github.com/fusakla/promruval/v2/pkg/prometheus"
	"github.com/fusakla/promruval/v2/pkg/report"
	"github.com/fusakla/promruval/v2/pkg/validator"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func NewValidationRule(name string, scope config.ValidationScope) *ValidationRule {
	return &ValidationRule{
		name:       name,
		scope:      scope,
		validators: []validator.Validator{},
	}
}

type ValidationRule struct {
	name       string
	scope      config.ValidationScope
	validators []validator.Validator
}

func (r *ValidationRule) AddValidator(newValidator validator.Validator) {
	r.validators = append(r.validators, newValidator)
}

func (r *ValidationRule) Name() string {
	return r.name
}

func (r *ValidationRule) Scope() string {
	return string(r.scope)
}

func (r *ValidationRule) ValidationTexts() []string {
	var validationTexts []string
	for _, v := range r.validators {
		validationTexts = append(validationTexts, v.String())
	}
	return validationTexts
}

type fakeTestFile struct {
	RuleFiles          []yaml.Node `yaml:"rule_files,omitempty"`
	EvaluationInterval yaml.Node   `yaml:"evaluation_interval,omitempty"`
	GroupEvalOrder     []yaml.Node `yaml:"group_eval_order,omitempty"`
	Tests              []yaml.Node `yaml:"tests,omitempty"`
}

type rulesFile struct {
	Groups []ruleGroup `yaml:"groups"`
	fakeTestFile
}

type ruleGroup struct {
	Name                    string            `yaml:"name"`
	Interval                model.Duration    `yaml:"interval,omitempty"`
	PartialResponseStrategy string            `yaml:"partial_response_strategy,omitempty"`
	SourceTenants           []string          `yaml:"source_tenants,omitempty"`
	Rules                   []ruleWithComment `yaml:"rules"`
}

type ruleWithComment struct {
	node yaml.Node
	rule rulefmt.RuleNode
}

func (r *ruleWithComment) UnmarshalYAML(value *yaml.Node) error {
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

func (r *ruleWithComment) disabledValidators(commentPrefix string) ([]string, error) {
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
			if !validator.KnownValidatorName(vv) {
				return disabledValidators, fmt.Errorf("unknown validator name `%s` in the `%s` comment", vv, commentPrefix)
			}
			disabledValidators = append(disabledValidators, vv)
		}
	}
	return disabledValidators, nil
}

func Files(fileNames []string, validationRules []*ValidationRule, excludeAnnotationName string, disableValidationsComment string, prometheusClient *prometheus.Client) *report.ValidationReport {
	validationReport := report.NewValidationReport()
	for _, r := range validationRules {
		validationReport.ValidationRules = append(validationReport.ValidationRules, r)
	}
	start := time.Now()
	fileCount := len(fileNames)
	for i, fileName := range fileNames {
		log.Infof("processing file %d/%d %s", i+1, fileCount, fileName)
		validationReport.FilesCount++
		fileReport := validationReport.NewFileReport(fileName)
		f, err := os.Open(fileName)
		if err != nil {
			validationReport.Failed = true
			fileReport.Valid = false
			fileReport.Errors = []error{fmt.Errorf("cannot read file %s: %s", fileName, err)}
			continue
		}
		var rf rulesFile
		decoder := yaml.NewDecoder(f)
		err = decoder.Decode(&rf)
		if err != nil {
			validationReport.Failed = true
			fileReport.Valid = false
			fileReport.Errors = []error{fmt.Errorf("invalid file %s: %s", fileName, err)}
			continue
		}
		for _, group := range rf.Groups {
			validationReport.GroupsCount++
			groupReport := fileReport.NewGroupReport(group.Name)
			for _, ruleNode := range group.Rules {
				rule := rulefmt.Rule{
					Record:      ruleNode.rule.Record.Value,
					Alert:       ruleNode.rule.Alert.Value,
					Expr:        ruleNode.rule.Expr.Value,
					For:         ruleNode.rule.For,
					Labels:      ruleNode.rule.Labels,
					Annotations: ruleNode.rule.Annotations,
				}
				var ruleReport *report.RuleReport
				if rule.Alert != "" {
					ruleReport = groupReport.NewRuleReport(rule.Alert, config.AlertScope)
				} else {
					ruleReport = groupReport.NewRuleReport(rule.Record, config.RecordingRuleScope)
				}
				var excludedRules []string
				excludedRulesText, ok := rule.Annotations[excludeAnnotationName]
				if ok {
					excludedRules = strings.Split(excludedRulesText, ",")
				}
				disabledValidators, err := ruleNode.disabledValidators(disableValidationsComment)
				if err != nil {
					ruleReport.Errors = append(ruleReport.Errors, err)
				}
				for _, validationRule := range validationRules {
					skipRule := false
					if (validationRule.scope != ruleReport.RuleType) && (validationRule.scope != config.AllRulesScope) {
						skipRule = true
					}
					for _, excludedRuleName := range excludedRules {
						if excludedRuleName == validationRule.Name() {
							skipRule = true
						}
					}
					if skipRule {
						continue
					}
					for _, v := range validationRule.validators {
						skipValidator := false
						validatorName := reflect.TypeOf(v).Elem().Name()
						for _, dv := range disabledValidators {
							if validatorName == dv {
								skipValidator = true
							}
						}
						if skipValidator {
							continue
						}
						for _, err := range v.Validate(rule, prometheusClient) {
							ruleReport.Errors = append(ruleReport.Errors, fmt.Errorf("%s: %w", validatorName, err))
						}
						log.Debugf("validation of file %s group %s using \"%s\" took %s", fileName, group.Name, v, time.Since(start))
					}
					if len(ruleReport.Errors) > 0 {
						validationReport.Failed = true
						fileReport.Valid = false
						groupReport.Valid = false
						ruleReport.Valid = false
					}
				}
			}
		}
	}
	validationReport.Duration = time.Since(start)
	return validationReport
}
