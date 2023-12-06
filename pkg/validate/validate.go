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
	"github.com/fusakla/promruval/v2/pkg/unmarshaler"
	"github.com/fusakla/promruval/v2/pkg/validationrule"
	"github.com/fusakla/promruval/v2/pkg/validator"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func Files(fileNames []string, validationRules []*validationrule.ValidationRule, excludeAnnotationName string, disableValidationsComment string, prometheusClient *prometheus.Client) *report.ValidationReport {
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
			fileReport.Errors = []error{fmt.Errorf("cannot read file %s: %w", fileName, err)}
			continue
		}
		var rf unmarshaler.RulesFile
		decoder := yaml.NewDecoder(f)
		err = decoder.Decode(&rf)
		if err != nil {
			validationReport.Failed = true
			fileReport.Valid = false
			fileReport.Errors = []error{fmt.Errorf("invalid file %s: %w", fileName, err)}
			continue
		}
		for _, group := range rf.Groups {
			validationReport.GroupsCount++
			groupReport := fileReport.NewGroupReport(group.Name)
			for _, ruleNode := range group.Rules {
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
				if err := validator.KnownValidators(disabledValidators); err != nil {
					ruleReport.Errors = append(ruleReport.Errors, err)
				}
				for _, rule := range validationRules {
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
						for _, err := range v.Validate(group, originalRule, prometheusClient) {
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
