package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fusakla/promruval/v3/pkg/config"
	"gopkg.in/yaml.v3"
)

type ValidationRule interface {
	Name() string
	Scope() config.ValidationScope
	ValidationTexts() []string
}

type SerializableValidationRule struct {
	Name            string
	ValidationTexts []string
}

func NewValidationReport() *ValidationReport {
	return &ValidationReport{
		Failed:          false,
		FilesReports:    []*FileReport{},
		ValidationRules: []ValidationRule{},
	}
}

type ValidationReport struct {
	Failed      bool
	Duration    time.Duration
	ErrorsCount int

	FilesCount         int
	FilesExcludedCount int

	GroupsCount         int
	GroupsExcludedCount int

	RulesCount         int
	RulesExcludedCount int

	ValidationRules []ValidationRule

	FilesReports []*FileReport
}

type SerializableValidationReport struct {
	Failed      bool
	Duration    time.Duration
	ErrorsCount int

	FilesCount         int
	FilesExcludedCount int

	GroupsCount         int
	GroupsExcludedCount int

	RulesCount         int
	RulesExcludedCount int

	ValidationRules []SerializableValidationRule

	FilesReports []*SerializableFileReport
}

func (r *ValidationReport) NewFileReport(name string) *FileReport {
	newReport := FileReport{
		Name:         name,
		Valid:        true,
		Errors:       []error{},
		GroupReports: []*GroupReport{},
	}
	r.FilesReports = append(r.FilesReports, &newReport)
	return &newReport
}

type FileReport struct {
	Name                    string
	Valid                   bool
	Excluded                bool
	Errors                  []error
	HasRuleValidationErrors bool
	GroupReports            []*GroupReport
}

type SerializableFileReport struct {
	Name                    string
	Valid                   bool
	Excluded                bool
	Errors                  []string
	HasRuleValidationErrors bool
	GroupReports            []*SerializableGroupReport
}

func (r *FileReport) NewGroupReport(name string) *GroupReport {
	newReport := GroupReport{
		Name:        name,
		Valid:       true,
		RuleReports: []*RuleReport{},
		Errors:      []error{},
	}
	r.GroupReports = append(r.GroupReports, &newReport)
	return &newReport
}

func (r *FileReport) AsText(output *IndentedOutput) {
	if r.Valid {
		return
	}
	output.AddLine("File: " + r.Name)
	output.IncreaseIndentation()
	defer output.DecreaseIndentation()
	output.AddTooPreviousLine(" - INVALID")
	output.WriteErrors(r.Errors)
	for _, group := range r.GroupReports {
		group.AsText(output)
	}
}

type GroupReport struct {
	Valid       bool
	Name        string
	Excluded    bool
	RuleReports []*RuleReport
	Errors      []error
}

type SerializableGroupReport struct {
	Valid       bool
	Name        string
	Excluded    bool
	RuleReports []*SerializableRuleReport
	Errors      []string
}

func (r *GroupReport) NewRuleReport(name string, ruleType config.ValidationScope) *RuleReport {
	newReport := RuleReport{
		Name:     name,
		Valid:    true,
		RuleType: ruleType,
		Errors:   []error{},
	}
	r.RuleReports = append(r.RuleReports, &newReport)
	return &newReport
}

func (r *GroupReport) AsText(output *IndentedOutput) {
	if r.Valid {
		return
	}
	output.AddLine("Group: " + r.Name)
	output.IncreaseIndentation()
	defer output.DecreaseIndentation()
	if r.Excluded {
		output.AddLine("Skipped")
		return
	}
	if r.Errors != nil {
		output.AddLine("Group level errors:")
		output.IncreaseIndentation()
		output.WriteErrors(r.Errors)
		output.DecreaseIndentation()
	}
	if len(r.RuleReports) == 0 {
		output.AddLine("No rules")
		return
	}
	for _, rule := range r.RuleReports {
		rule.AsText(output)
	}
}

type RuleReport struct {
	Valid    bool
	RuleType config.ValidationScope
	Name     string
	Excluded bool
	Errors   []error
}

type SerializableRuleReport struct {
	Valid    bool
	RuleType config.ValidationScope
	Name     string
	Excluded bool
	Errors   []string
}

func (r *RuleReport) AsText(output *IndentedOutput) {
	if r.Valid {
		return
	}
	output.AddLine(string(r.RuleType) + ": " + r.Name)
	output.IncreaseIndentation()
	defer output.DecreaseIndentation()
	if r.Excluded {
		output.AddLine("Skipped")
		return
	}
	output.WriteErrors(r.Errors)
}

func (r *ValidationReport) AsText(indentationStep int, color bool) (string, error) {
	output := NewIndentedOutput(indentationStep, color)
	output.AddLine("Validation rules used:")
	output.IncreaseIndentation()
	for _, rule := range r.ValidationRules {
		output.AddLine("")
		output.AddLine(rule.Name() + ":")
		output.IncreaseIndentation()
		for _, check := range rule.ValidationTexts() {
			output.AddLine("- " + check)
		}
		output.DecreaseIndentation()
	}
	output.DecreaseIndentation()
	output.AddLine("\n")
	output.AddLine("Result: ")

	output.IncreaseIndentation()
	for _, file := range r.FilesReports {
		file.AsText(&output)
	}

	output.ResetIndentation()
	output.AddLine("\n")

	if r.Failed {
		output.AddErrorLine("Validation FAILED")
	} else {
		output.AddSuccessLine("Validation PASSED")
	}
	output.AddLine("Statistics:")
	output.IncreaseIndentation()
	output.AddLine("Duration: " + r.Duration.String())
	output.AddLine(renderStatistic("Files", r.FilesCount, r.FilesExcludedCount))
	output.AddLine(renderStatistic("Groups", r.GroupsCount, r.GroupsExcludedCount))
	output.AddLine(renderStatistic("Rules", r.RulesCount, r.RulesExcludedCount))
	return output.Text(), nil
}

func renderStatistic(objectType string, total, excluded int) string {
	return fmt.Sprintf("%s: %d and %d of them excluded", objectType, total, excluded)
}

func (r *ValidationReport) SerializeValidationReport() (SerializableValidationReport, error) {
	if r == nil {
		return SerializableValidationReport{}, fmt.Errorf("ValidationReport is nil")
	}

	SerializedFileReports := []*SerializableFileReport{}
	for _, FileReport := range r.FilesReports {
		StringifiedFileReportErrors := []string{}
		for _, FileReportError := range FileReport.Errors {
			if FileReportError != nil {
				StringifiedFileReportErrors = append(StringifiedFileReportErrors, FileReportError.Error())
			}
		}
		SerializedGroupReports := []*SerializableGroupReport{}
		for _, GroupReport := range FileReport.GroupReports {
			if GroupReport == nil {
				continue
			}
			StringifiedGroupReportErrors := []string{}
			for _, GroupReportError := range GroupReport.Errors {
				if GroupReportError != nil {
					StringifiedGroupReportErrors = append(StringifiedGroupReportErrors, GroupReportError.Error())
				}
			}
			SerializedRuleReports := []*SerializableRuleReport{}
			for _, RuleReport := range GroupReport.RuleReports {
				if RuleReport == nil {
					continue
				}
				StringifiedRuleReportErrors := []string{}
				for _, RuleReportError := range RuleReport.Errors {
					if RuleReportError != nil {
						StringifiedRuleReportErrors = append(StringifiedRuleReportErrors, RuleReportError.Error())
					}
				}
				SerializedRuleReport := SerializableRuleReport{
					Valid:    RuleReport.Valid,
					RuleType: RuleReport.RuleType,
					Name:     RuleReport.Name,
					Excluded: RuleReport.Excluded,
					Errors:   StringifiedRuleReportErrors,
				}
				SerializedRuleReports = append(SerializedRuleReports, &SerializedRuleReport)
			}
			SerializedGroupReport := SerializableGroupReport{
				Valid:       GroupReport.Valid,
				Name:        GroupReport.Name,
				Excluded:    GroupReport.Excluded,
				RuleReports: SerializedRuleReports,
				Errors:      StringifiedGroupReportErrors,
			}
			SerializedGroupReports = append(SerializedGroupReports, &SerializedGroupReport)
		}

		SerializedFileReport := SerializableFileReport{
			Name:                    FileReport.Name,
			Valid:                   FileReport.Valid,
			Excluded:                FileReport.Excluded,
			Errors:                  StringifiedFileReportErrors,
			HasRuleValidationErrors: FileReport.HasRuleValidationErrors,
			GroupReports:            SerializedGroupReports,
		}
		SerializedFileReports = append(SerializedFileReports, &SerializedFileReport)
	}
	SerializedValidationRules := []SerializableValidationRule{}
	for _, ValidationRule := range r.ValidationRules {
		if ValidationRule == nil {
			continue
		}
		SerializedValidationRule := SerializableValidationRule{
			Name:            ValidationRule.Name(),
			ValidationTexts: ValidationRule.ValidationTexts(),
		}
		SerializedValidationRules = append(SerializedValidationRules, SerializedValidationRule)
	}
	SerializedValidationReport := SerializableValidationReport{
		Failed:              r.Failed,
		Duration:            r.Duration,
		ErrorsCount:         r.ErrorsCount,
		FilesCount:          r.FilesCount,
		FilesExcludedCount:  r.FilesExcludedCount,
		GroupsCount:         r.GroupsCount,
		GroupsExcludedCount: r.GroupsExcludedCount,
		RulesCount:          r.RulesCount,
		RulesExcludedCount:  r.RulesExcludedCount,
		ValidationRules:     SerializedValidationRules,
		FilesReports:        SerializedFileReports,
	}
	return SerializedValidationReport, nil
}

func (r *ValidationReport) AsJSON() (string, error) {
	sr, err := r.SerializeValidationReport()
	if err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(sr, "", "  ")
	if err != nil {
		return "", err
	}
	buffer := bytes.NewBuffer(b)
	return buffer.String(), nil
}

func (r *ValidationReport) AsYaml() (string, error) {
	sr, err := r.SerializeValidationReport()
	if err != nil {
		return "", err
	}
	b, err := yaml.Marshal(sr)
	if err != nil {
		return "", err
	}
	buffer := bytes.NewBuffer(b)
	return buffer.String(), nil
}
