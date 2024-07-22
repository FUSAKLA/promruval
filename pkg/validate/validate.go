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
		log.Infof("processing file %d/%d %s", i+1, fileCount, fileName)
		validationReport.FilesCount++
		fileReport := validationReport.NewFileReport(fileName)
		var yamlReader io.Reader
		if strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml") {
			var err error
			yamlReader, err = os.Open(fileName)
			if err != nil {
				validationReport.Failed = true
				fileReport.Valid = false
				fileReport.Errors = []error{fmt.Errorf("cannot read file %s: %w", fileName, err)}
				continue
			}
		} else if strings.HasSuffix(fileName, ".jsonnet") {
			log.Debugf("evaluating jsonnet file %s", fileName)
			jsonnetOutput, err := jsonnetVM.EvaluateFile(fileName)
			if err != nil {
				validationReport.Failed = true
				fileReport.Valid = false
				fileReport.Errors = []error{fmt.Errorf("cannot evaluate jsonnet file %s: %w", fileName, err)}
				continue
			}
			yamlReader = strings.NewReader(jsonnetOutput)
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
			for _, rule := range validationRules {
				if rule.Scope() != config.GroupScope {
					continue
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
				if originalRule.Alert != "" {
					ruleReport = groupReport.NewRuleReport(originalRule.Alert, config.AlertScope)
				} else {
					ruleReport = groupReport.NewRuleReport(originalRule.Record, config.RecordingRuleScope)
				}
				var excludedRules []string
				excludedRulesText, ok := originalRule.Annotations[excludeAnnotationName]
				if ok {
					excludedRules = strings.Split(excludedRulesText, ",")
				}
				disabledValidators := ruleNode.DisabledValidators(disableValidationsComment)
				if err := validator.KnownValidators(config.AllScope, disabledValidators); err != nil {
					ruleReport.Errors = append(ruleReport.Errors, fmt.Errorf("invalid disabled validators: %w", err))
				}
				disabledValidators = append(disabledValidators, groupDisabledValidators...)
				for _, rule := range validationRules {
					if rule.Scope() == config.GroupScope {
						continue
					}
					skipRule := false
					if (rule.Scope() != ruleReport.RuleType) && (rule.Scope() != config.AllRulesScope) {
						skipRule = true
					}
					for _, excludedRuleName := range excludedRules {
						if excludedRuleName == rule.Name() {
							skipRule = true
						}
					}
					if skipRule {
						continue
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
