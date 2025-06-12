package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fusakla/promruval/v3/pkg/config"
	"gopkg.in/yaml.v3"
)

type ValidationRule interface {
	Name() string
	Scope() config.ValidationScope
	ValidationTexts() []string
	OnlyIfValidationTexts() []string
	json.Marshaler
	yaml.Marshaler
}

func NewValidationReport() *ValidationReport {
	return &ValidationReport{
		Failed:          false,
		FilesReports:    []*FileReport{},
		ValidationRules: []ValidationRule{},
	}
}

func NewErrorf(format string, args ...any) *Error {
	return &Error{
		error: fmt.Errorf(format, args...),
	}
}

func NewError(msg string) *Error {
	return &Error{
		error: errors.New(msg),
	}
}

type Error struct {
	error
}

func (e *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

func (e *Error) MarshalYAML() (any, error) {
	return e.String(), nil
}

func (e *Error) String() string {
	if e.error == nil {
		return "null"
	}
	return e.Error()
}

type ValidationReport struct {
	Failed      bool          `json:"report_failed" yaml:"report_failed"`
	Duration    time.Duration `json:"duration" yaml:"duration"`
	ErrorsCount int           `json:"errors_count" yaml:"errors_count"`

	FilesCount         int `json:"files_count" yaml:"files_count"`
	FilesExcludedCount int `json:"excluded_files_count" yaml:"excluded_files_count"`

	GroupsCount         int `json:"groups_count" yaml:"groups_count"`
	GroupsExcludedCount int `json:"excluded_groups_count" yaml:"excluded_groups_count"`

	RulesCount         int `json:"rules_count" yaml:"rules_count"`
	RulesExcludedCount int `json:"excluded_rules_count" yaml:"excluded_rules_count"`

	ValidationRules []ValidationRule `json:"validation_rules" yaml:"validation_rules"`

	FilesReports []*FileReport `json:"files_reports" yaml:"files_reports"`
}

func (r *ValidationReport) NewFileReport(name string) *FileReport {
	newReport := FileReport{
		Name:         name,
		Valid:        true,
		Errors:       []*Error{},
		GroupReports: []*GroupReport{},
	}
	r.FilesReports = append(r.FilesReports, &newReport)
	return &newReport
}

type FileReport struct {
	Name                    string         `json:"file_name" yaml:"file_name"`
	Valid                   bool           `json:"valid" yaml:"valid"`
	Excluded                bool           `json:"excluded" yaml:"excluded"`
	Errors                  []*Error       `json:"errors" yaml:"errors"`
	HasRuleValidationErrors bool           `json:"has_rule_validation_errors" yaml:"has_rule_validation_errors"`
	GroupReports            []*GroupReport `json:"group_reports" yaml:"group_reports"`
}

func (r *FileReport) NewGroupReport(name string) *GroupReport {
	newReport := GroupReport{
		Name:        name,
		Valid:       true,
		RuleReports: []*RuleReport{},
		Errors:      []*Error{},
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
	Valid       bool          `json:"valid" yaml:"valid"`
	Name        string        `json:"group_name" yaml:"group_name"`
	Excluded    bool          `json:"excluded" yaml:"excluded"`
	RuleReports []*RuleReport `json:"rule_reports" yaml:"rule_reports"`
	Errors      []*Error      `json:"errors" yaml:"errors"`
}

func (r *GroupReport) NewRuleReport(name string, ruleType config.ValidationScope) *RuleReport {
	newReport := RuleReport{
		Name:     name,
		Valid:    true,
		RuleType: ruleType,
		Errors:   []*Error{},
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
	if len(r.Errors) > 0 {
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
	Valid    bool                   `json:"valid" yaml:"valid"`
	RuleType config.ValidationScope `json:"rule_type" yaml:"rule_type"`
	Name     string                 `json:"name" yaml:"name"`
	Excluded bool                   `json:"excluded" yaml:"excluded"`
	Errors   []*Error               `json:"errors" yaml:"errors"`
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
	validationText, err := ValidationDocs(r.ValidationRules, "text")
	if err != nil {
		return "", err
	}
	output.AddLine(validationText)
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

func (r *ValidationReport) AsJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	buffer := bytes.NewBuffer(b)
	return buffer.String(), nil
}

func (r *ValidationReport) AsYaml() (string, error) {
	b, err := yaml.Marshal(r)
	if err != nil {
		return "", err
	}
	buffer := bytes.NewBuffer(b)
	return buffer.String(), nil
}
