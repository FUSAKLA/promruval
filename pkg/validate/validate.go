package validate

import (
	"github.com/fusakla/promruval/pkg/config"
	"github.com/fusakla/promruval/pkg/report"
	"github.com/fusakla/promruval/pkg/validator"
	"github.com/prometheus/prometheus/pkg/rulefmt"
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

func Files(fileNames []string, validationRules []*ValidationRule, excludeAnnotationName string) *report.ValidationReport {
	validationReport := report.NewValidationReport()
	for _, r := range validationRules {
		validationReport.ValidationRules = append(validationReport.ValidationRules, r)
	}
	start := time.Now()
	for _, fileName := range fileNames {
		validationReport.FilesCount++
		fileReport := validationReport.NewFileReport(fileName)
		rgs, errs := rulefmt.ParseFile(fileName)
		if len(errs) > 0 {
			validationReport.Failed = true
			fileReport.Valid = false
			fileReport.Errors = errs
			continue
		}
		for _, group := range rgs.Groups {
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
						ruleReport.Errors = append(ruleReport.Errors, v.Validate(rule)...)
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
