package validate

import (
	"errors"
	"fmt"
	"io"
	log "log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/fusakla/promruval/v3/pkg/config"
	"github.com/fusakla/promruval/v3/pkg/prometheus"
	"github.com/fusakla/promruval/v3/pkg/report"
	"github.com/fusakla/promruval/v3/pkg/unmarshaler"
	"github.com/fusakla/promruval/v3/pkg/validationrule"
	"github.com/fusakla/promruval/v3/pkg/validator"
	"github.com/google/go-jsonnet"
	"github.com/prometheus/prometheus/model/rulefmt"
	"gopkg.in/yaml.v3"
)

func validateWithDetails(v validationrule.ValidatorWithDetails, group unmarshaler.RuleGroup, rule rulefmt.Rule, prometheusClient *prometheus.Client) []*report.Error {
	var reportedError *report.Error
	validatorName := v.Name()
	additionalDetails := v.AdditionalDetails()
	validationErrors := v.Validate(group, rule, prometheusClient)
	errs := make([]*report.Error, 0, len(validationErrors))
	for _, err := range validationErrors {
		if additionalDetails != "" {
			reportedError = report.NewErrorf("%s: %w (%s)", validatorName, err, additionalDetails)
		} else {
			reportedError = report.NewErrorf("%s: %w", validatorName, err)
		}
		errs = append(errs, reportedError)
	}
	return errs
}

func validateFile(fileName string, validationRules []*validationrule.ValidationRule, excludeAnnotationName, disableValidationsComment string, prometheusClient *prometheus.Client, jsonnetVM *jsonnet.VM, validationReport *report.ValidationReport, disableParallelization bool) (groupsCount, rulesCount int, err error) {
	fileReport := validationReport.NewFileReport(fileName)
	var yamlReader io.Reader
	groupsCount = 0
	rulesCount = 0

	switch {
	case strings.HasSuffix(fileName, ".jsonnet"):
		log.Debug("evaluating jsonnet file", "file", fileName)
		jsonnetOutput, err := jsonnetVM.EvaluateFile(fileName)
		if err != nil {
			fileReport.Valid.Store(false)
			fileReport.AddError(report.NewErrorf("cannot evaluate jsonnet file %s: %w", fileName, err))
			return groupsCount, rulesCount, err
		}
		yamlReader = strings.NewReader(jsonnetOutput)
	default:
		var err error
		yamlReader, err = os.Open(fileName)
		if err != nil {
			fileReport.Valid.Store(false)
			fileReport.AddError(report.NewErrorf("cannot read file %s: %w", fileName, err))
			return groupsCount, rulesCount, err
		}
	}
	var rf unmarshaler.RulesFileWithComment
	decoder := yaml.NewDecoder(yamlReader)
	decoder.KnownFields(true)
	err = decoder.Decode(&rf)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return groupsCount, rulesCount, nil
		}
		fileReport.Valid.Store(false)
		fileReport.AddError(report.NewErrorf("invalid file %s: %w", fileName, err))
		return groupsCount, rulesCount, err
	}
	fileDisabledValidators := rf.DisabledValidators(disableValidationsComment)
	allGroupsDisabledValidators := rf.Groups.DisabledValidators(disableValidationsComment)
	for _, group := range rf.Groups.Groups {
		groupsCount++
		groupReport := fileReport.NewGroupReport(group.Name)
		groupDisabledValidators := group.DisabledValidators(disableValidationsComment)
		if err := validator.KnownValidators(config.AllScope, groupDisabledValidators); err != nil {
			groupReport.AddError(report.NewErrorf("invalid disabled validators: %w", err))
		}
		groupDisabledValidators = slices.Concat(groupDisabledValidators, fileDisabledValidators, allGroupsDisabledValidators)

		var groupWg sync.WaitGroup
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
					log.Debug("skipping validation because onlyIf results with errors", "file", fileName, "group", group.Name, "validator", v, "errors", errs)
					continue groupValidationLoop
				}
			}
			for _, v := range rule.Validators() {
				if slices.Contains(groupDisabledValidators, v.Name()) {
					continue
				}
				groupWg.Add(1)
				go func(validator validationrule.ValidatorWithDetails) {
					defer groupWg.Done()
					errs := validateWithDetails(validator, group.RuleGroup, rulefmt.Rule{}, prometheusClient)
					if len(errs) > 0 {
						groupReport.AddErrors(errs)
					}
				}(v)
				if disableParallelization {
					groupWg.Wait()
				}
			}
		}
		groupWg.Wait()
		if len(groupReport.Errors) > 0 {
			fileReport.Valid.Store(false)
			groupReport.Valid.Store(false)
		}
		for _, ruleNode := range group.Rules {
			rulesCount++
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
				ruleReport.AddError(report.NewErrorf("invalid disabled validators: %w", err))
			}
			disabledValidators = append(disabledValidators, groupDisabledValidators...)

			var ruleWg sync.WaitGroup
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
							log.Debug("skipping validation because onlyIf results with errors", "file", fileName, "group", group.Name, "validator", v, "errors", errs)
							continue ruleValidationLoop
						}
					} else {
						log.Debug("skipping onlyIf validation because it is not applicable", "file", fileName, "group", group.Name, "validator_scope", validator.Scope(v.Name()), "rule_scope", ruleNode.Scope())
					}
				}
				for _, v := range rule.Validators() {
					validatorName := v.Name()
					if slices.Contains(disabledValidators, validatorName) {
						continue
					}
					ruleWg.Add(1)
					go func(validator validationrule.ValidatorWithDetails, grp unmarshaler.RuleGroup, rule rulefmt.Rule, vName string) {
						defer ruleWg.Done()
						validationStart := time.Now()
						errs := validateWithDetails(validator, grp, rule, prometheusClient)
						if len(errs) > 0 {
							ruleReport.AddErrors(errs)
						}
						log.Debug("validation completed", "file", fileName, "group", group.Name, "validator", vName, "duration", time.Since(validationStart))
					}(v, group.RuleGroup, originalRule, validatorName)
					if disableParallelization {
						ruleWg.Wait()
					}
				}
			}
			ruleWg.Wait()
			if len(ruleReport.Errors) > 0 {
				fileReport.Valid.Store(false)
				groupReport.Valid.Store(false)
				ruleReport.Valid.Store(false)
			}
		}
	}
	return groupsCount, rulesCount, nil
}

func Files(fileNames []string, validationRules []*validationrule.ValidationRule, excludeAnnotationName, disableValidationsComment string, prometheusClient *prometheus.Client, disableParallelization bool) *report.ValidationReport {
	validationReport := report.NewValidationReport()
	for _, r := range validationRules {
		validationReport.ValidationRules = append(validationReport.ValidationRules, r)
	}

	start := time.Now()
	fileCount := len(fileNames)

	var reportMutex sync.Mutex
	var filesWg sync.WaitGroup

	// Create a jsonnet VM for each goroutine to avoid race conditions
	for i, fileName := range fileNames {
		filesWg.Add(1)
		go func(fileName string, i int) {
			defer filesWg.Done()
			fileStart := time.Now()
			jsonnetVM := jsonnet.MakeVM()
			groupsCount, rulesCount, err := validateFile(fileName, validationRules, excludeAnnotationName, disableValidationsComment, prometheusClient, jsonnetVM, validationReport, disableParallelization)
			if err != nil {
				log.Error("error validating file", "file", fileName, "error", err)
			}

			reportMutex.Lock()
			validationReport.FilesCount.Inc()
			validationReport.GroupsCount.Add(int32(groupsCount))
			validationReport.RulesCount.Add(int32(rulesCount))
			if len(validationReport.FilesReports) > 0 && !validationReport.FilesReports[len(validationReport.FilesReports)-1].Valid.Load() {
				validationReport.Failed.Store(true)
			}
			reportMutex.Unlock()
			if disableParallelization {
				log.Info("finished processing file", "file", fileName, "duration", time.Since(fileStart), "progress", fmt.Sprintf("%d/%d", i+1, fileCount))
			} else {
				log.Info("finished processing file", "file", fileName, "duration", time.Since(fileStart))
			}
		}(fileName, i)
		if disableParallelization {
			filesWg.Wait()
		}
	}
	filesWg.Wait()
	validationReport.Duration.Store(time.Since(start))
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

func Cmd(filePaths []string, mainConfig *config.Config, validationRules []*validationrule.ValidationRule, supportLoki, supportMimir, supportThanos, disableParallelization bool) (*report.ValidationReport, error) {
	var filesToBeValidated []string
	for _, path := range filePaths {
		if strings.HasPrefix(path, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get user home directory: %w", err)
			}
			path = filepath.Join(home, path[2:])
		}

		base, pattern := doublestar.SplitPattern(path)
		paths, err := doublestar.Glob(os.DirFS(base), pattern, doublestar.WithFilesOnly(), doublestar.WithFailOnIOErrors(), doublestar.WithFailOnPatternNotExist())
		if err != nil {
			return nil, fmt.Errorf("failed expanding glob pattern `%s`: %w", path, err)
		}
		for _, p := range paths {
			filesToBeValidated = append(filesToBeValidated, filepath.Join(base, p))
		}
	}

	if supportLoki {
		unmarshaler.SupportLoki(true)
	}

	if supportMimir {
		unmarshaler.SupportMimir(true)
	}

	if supportThanos {
		unmarshaler.SupportThanos(true)
	}

	var err error
	var prometheusClient *prometheus.Client
	if mainConfig.Prometheus != nil && mainConfig.Prometheus.URL != "" {
		prometheusClient, err = prometheus.NewClient(*mainConfig.Prometheus)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize prometheus client: %w", err)
		}
	}

	excludeAnnotation := "disabled_validation_rules"
	if mainConfig.CustomExcludeAnnotation != "" {
		excludeAnnotation = mainConfig.CustomExcludeAnnotation
	}
	disableValidatorsComment := "ignore_validations"
	if mainConfig.CustomDisableComment != "" {
		disableValidatorsComment = mainConfig.CustomDisableComment
	}
	validationReport := Files(filesToBeValidated, validationRules, excludeAnnotation, disableValidatorsComment, prometheusClient, disableParallelization)

	if mainConfig.Prometheus != nil && mainConfig.Prometheus.URL != "" {
		prometheusClient.DumpCache()
	}
	return validationReport, nil
}
