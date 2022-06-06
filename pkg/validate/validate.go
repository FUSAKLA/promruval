package validate

import (
	"fmt"
	"github.com/fusakla/promruval/pkg/config"
	"github.com/fusakla/promruval/pkg/prometheus"
	"github.com/fusakla/promruval/pkg/report"
	"github.com/fusakla/promruval/pkg/validator"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"time"
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

type rulesFile struct {
	Groups []ruleGroup `yaml:"groups"`
}

type ruleGroup struct {
	Name                    string             `yaml:"name"`
	Interval                model.Duration     `yaml:"interval,omitempty"`
	PartialResponseStrategy string             `yaml:"partial_response_strategy,omitempty"`
	Rules                   []rulefmt.RuleNode `yaml:"rules"`
}

func Files(fileNames []string, validationRules []*ValidationRule, excludeAnnotationName string, prometheusClient *prometheus.Client) *report.ValidationReport {
	validationReport := report.NewValidationReport()
	for _, r := range validationRules {
		validationReport.ValidationRules = append(validationReport.ValidationRules, r)
	}
	start := time.Now()
	for _, fileName := range fileNames {
		validationReport.FilesCount++
		fileReport := validationReport.NewFileReport(fileName)
		f, err := os.Open(fileName)
		if err != nil {
			validationReport.Failed = true
			fileReport.Errors = []error{fmt.Errorf("cannot read file %s: %s", fileName, err)}
			continue
		}
		var rf rulesFile
		decoder := yaml.NewDecoder(f)
		decoder.KnownFields(true)
		err = decoder.Decode(&rf)
		if err != nil {
			validationReport.Failed = true
			fileReport.Errors = []error{fmt.Errorf("invalid file %s: %s", fileName, err)}
			continue
		}
		for _, group := range rf.Groups {
			validationReport.GroupsCount++
			groupReport := fileReport.NewGroupReport(group.Name)
			for _, ruleNode := range group.Rules {
				validationReport.RulesCount++
				rule := rulefmt.Rule{
					Record:      ruleNode.Record.Value,
					Alert:       ruleNode.Alert.Value,
					Expr:        ruleNode.Expr.Value,
					For:         ruleNode.For,
					Labels:      ruleNode.Labels,
					Annotations: ruleNode.Annotations,
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
			validationRulesIteration:
				for _, validationRule := range validationRules {
					if (validationRule.scope != ruleReport.RuleType) && (validationRule.scope != config.AllRulesScope) {
						continue
					}
					for _, excludedRuleName := range excludedRules {
						if excludedRuleName == validationRule.Name() {
							continue validationRulesIteration
						}
					}
					for _, v := range validationRule.validators {
						start := time.Now()
						ruleReport.Errors = append(ruleReport.Errors, v.Validate(rule, prometheusClient)...)
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
