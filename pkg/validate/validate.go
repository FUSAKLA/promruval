package validate

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/fusakla/promruval/v3/pkg/config"
	"github.com/fusakla/promruval/v3/pkg/prometheus"
	"github.com/fusakla/promruval/v3/pkg/report"
	"github.com/fusakla/promruval/v3/pkg/unmarshaler"
	"github.com/fusakla/promruval/v3/pkg/validationrule"
	"github.com/fusakla/promruval/v3/pkg/validator"
	"github.com/google/go-jsonnet"
	"github.com/prometheus/prometheus/model/rulefmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func validateWithDetails(v validationrule.ValidatorWithDetails, group unmarshaler.RuleGroup, rule rulefmt.Rule, prometheusClient *prometheus.Client) []error {
	var reportedError error
	validatorName := v.Name()
	additionalDetails := v.AdditionalDetails()
	validationErrors := v.Validate(group, rule, prometheusClient)
	errs := make([]error, 0, len(validationErrors))
	for _, err := range validationErrors {
		if additionalDetails != "" {
			reportedError = fmt.Errorf("%s: %w (%s)", validatorName, err, additionalDetails)
		} else {
			reportedError = fmt.Errorf("%s: %w", validatorName, err)
		}
		errs = append(errs, reportedError)
	}
	return errs
}

func Files(fileNames []string, validationRules []*validationrule.ValidationRule, excludeAnnotationName, disableValidationsComment string, prometheusClient *prometheus.Client) *report.ValidationReport {
	validationReport := report.NewValidationReport()
	for _, r := range validationRules {
		validationReport.ValidationRules = append(validationReport.ValidationRules, r)
	}
	jsonnetVM := jsonnet.MakeVM()
	start := time.Now()
	fileCount := len(fileNames)
	for i, fileName := range fileNames {
		log.WithFields(log.Fields{
			"file":     fileName,
			"progress": fmt.Sprintf("%d/%d", i+1, fileCount),
		}).Info("processing file")
		validationReport.FilesCount++
		fileReport := validationReport.NewFileReport(fileName)
		var yamlReader io.Reader
		switch {
		case strings.HasSuffix(fileName, ".jsonnet"):
			log.Debugf("evaluating jsonnet file %s", fileName)
			jsonnetOutput, err := jsonnetVM.EvaluateFile(fileName)
			if err != nil {
				validationReport.Failed = true
				fileReport.Valid = false
				fileReport.Errors = []error{fmt.Errorf("cannot evaluate jsonnet file %s: %w", fileName, err)}
				continue
			}
			yamlReader = strings.NewReader(jsonnetOutput)
		default:
			var err error
			yamlReader, err = os.Open(fileName)
			if err != nil {
				validationReport.Failed = true
				fileReport.Valid = false
				fileReport.Errors = []error{fmt.Errorf("cannot read file %s: %w", fileName, err)}
				continue
			}
		}
		var rf unmarshaler.RulesFileWithComment
		decoder := yaml.NewDecoder(yamlReader)
		decoder.KnownFields(true)
		err := decoder.Decode(&rf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			validationReport.Failed = true
			fileReport.Valid = false
			fileReport.Errors = []error{fmt.Errorf("invalid file %s: %w", fileName, err)}
			continue
		}
		fileDisabledValidators := rf.DisabledValidators(disableValidationsComment)
		allGroupsDisabledValidators := rf.Groups.DisabledValidators(disableValidationsComment)
		for _, group := range rf.Groups.Groups {
			validationReport.GroupsCount++
			groupReport := fileReport.NewGroupReport(group.Name)
			groupDisabledValidators := group.DisabledValidators(disableValidationsComment)
			if err := validator.KnownValidators(config.AllScope, groupDisabledValidators); err != nil {
				groupReport.Errors = append(groupReport.Errors, fmt.Errorf("invalid disabled validators: %w", err))
			}
			groupDisabledValidators = slices.Concat(groupDisabledValidators, fileDisabledValidators, allGroupsDisabledValidators)
		groupValidationLoop:
			for _, rule := range validationRules {
				if rule.Scope() != config.GroupScope {
					continue
				}
				for _, v := range rule.OnlyIf() {
					if validator.Scope(v.Name()) != config.GroupScope {
						continue
					}
					if errs := validateWithDetails(v, group.RuleGroup, rulefmt.Rule{}, prometheusClient); len(errs) > 0 {
						log.Debugf("skipping validation of file %s group %s using \"%s\" because onlyIf results with errors: %v", fileName, group.Name, v, errs)
						continue groupValidationLoop
					}
				}
				for _, v := range rule.Validators() {
					if slices.Contains(groupDisabledValidators, v.Name()) {
						continue
					}
					groupReport.Errors = append(groupReport.Errors, validateWithDetails(v, group.RuleGroup, rulefmt.Rule{}, prometheusClient)...)
				}
			}
			if len(groupReport.Errors) > 0 {
				validationReport.Failed = true
				fileReport.Valid = false
				groupReport.Valid = false
			}
			for _, ruleNode := range group.Rules {
				validationReport.RulesCount++
				originalRule := ruleNode.OriginalRule()
				var ruleReport *report.RuleReport
				switch ruleNode.Scope() {
				case config.AlertScope:
					ruleReport = groupReport.NewRuleReport(originalRule.Alert, config.AlertScope)
				case config.RecordingRuleScope:
					ruleReport = groupReport.NewRuleReport(originalRule.Record, config.RecordingRuleScope)
				}
				var excludedRules []string
				excludedRulesText, ok := originalRule.Annotations[excludeAnnotationName]
				if ok {
					excludedRules = generateExcludedRules(excludedRulesText)
				}
				disabledValidators := ruleNode.DisabledValidators(disableValidationsComment)
				if err := validator.KnownValidators(config.AllScope, disabledValidators); err != nil {
					ruleReport.Errors = append(ruleReport.Errors, fmt.Errorf("invalid disabled validators: %w", err))
				}
				disabledValidators = append(disabledValidators, groupDisabledValidators...)
			ruleValidationLoop:
				for _, rule := range validationRules {
					if rule.Scope() == config.GroupScope {
						continue
					}
					if (rule.Scope() != ruleReport.RuleType) && (rule.Scope() != config.AllRulesScope) {
						continue
					}
					for _, excludedRuleName := range excludedRules {
						if excludedRuleName == rule.Name() {
							continue ruleValidationLoop
						}
					}
					for _, v := range rule.OnlyIf() {
						if validator.MatchesScope(originalRule, ruleNode.Scope()) {
							if errs := validateWithDetails(v, group.RuleGroup, originalRule, prometheusClient); len(errs) > 0 {
								log.Debugf("skipping validation of file %s group %s using \"%s\" because onlyIf results with errors: %v", fileName, group.Name, v, errs)
								continue ruleValidationLoop
							}
						} else {
							log.Debugf("skipping onlyIf validation of file %s group %s because it is not applicable: validator scrope: `%s`, rule scope: `%s`", fileName, group.Name, validator.Scope(v.Name()), ruleNode.Scope())
						}
					}
					for _, v := range rule.Validators() {
						validatorName := v.Name()
						if slices.Contains(disabledValidators, validatorName) {
							continue
						}
						ruleReport.Errors = append(ruleReport.Errors, validateWithDetails(v, group.RuleGroup, originalRule, prometheusClient)...)
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

func generateExcludedRules(excludedRulesText string) []string {
	var excludedRules []string
	for _, r := range strings.Split(excludedRulesText, ",") {
		rule := strings.TrimSpace(r)
		if rule != "" {
			excludedRules = append(excludedRules, rule)
		}
	}
	slices.Sort(excludedRules)
	return slices.Compact(excludedRules)
}
