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
	OnlyIfValidationTexts() []string
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
	Valid    bool
	RuleType config.ValidationScope
	Name     string
	Excluded bool
	Errors   []error
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
