package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/fusakla/promruval/v3/pkg/config"
	"go.uber.org/atomic"
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
		Failed:              atomic.NewBool(false),
		Duration:            atomic.NewDuration(0),
		ErrorsCount:         atomic.NewInt32(0),
		FilesCount:          atomic.NewInt32(0),
		FilesExcludedCount:  atomic.NewInt32(0),
		GroupsCount:         atomic.NewInt32(0),
		GroupsExcludedCount: atomic.NewInt32(0),
		RulesCount:          atomic.NewInt32(0),
		RulesExcludedCount:  atomic.NewInt32(0),
		FilesReports:        []*FileReport{},
		ValidationRules:     []ValidationRule{},
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
	Failed      *atomic.Bool     `json:"report_failed" yaml:"report_failed"`
	Duration    *atomic.Duration `json:"duration" yaml:"duration"`
	ErrorsCount *atomic.Int32    `json:"errors_count" yaml:"errors_count"`

	FilesCount         *atomic.Int32 `json:"files_count" yaml:"files_count"`
	FilesExcludedCount *atomic.Int32 `json:"excluded_files_count" yaml:"excluded_files_count"`

	GroupsCount         *atomic.Int32 `json:"groups_count" yaml:"groups_count"`
	GroupsExcludedCount *atomic.Int32 `json:"excluded_groups_count" yaml:"excluded_groups_count"`

	RulesCount         *atomic.Int32 `json:"rules_count" yaml:"rules_count"`
	RulesExcludedCount *atomic.Int32 `json:"excluded_rules_count" yaml:"excluded_rules_count"`

	ValidationRules []ValidationRule `json:"validation_rules" yaml:"validation_rules"`

	FilesReports []*FileReport `json:"files_reports" yaml:"files_reports"`

	mu sync.Mutex `json:"-" yaml:"-"`
}

func (r *ValidationReport) NewFileReport(name string) *FileReport {
	newReport := &FileReport{
		Name:                    name,
		Valid:                   atomic.NewBool(true),
		Excluded:                atomic.NewBool(false),
		Errors:                  []*Error{},
		HasRuleValidationErrors: atomic.NewBool(false),
		GroupReports:            []*GroupReport{},
	}

	r.mu.Lock()
	r.FilesReports = append(r.FilesReports, newReport)
	r.mu.Unlock()
	return newReport
}

// Sort sorts all reports (files, groups, rules) for predictable output.
// This method is thread-safe and uses mutex protection to prevent race conditions
// with concurrent modifications from NewFileReport, NewGroupReport, and NewRuleReport.
func (r *ValidationReport) Sort() {
	r.mu.Lock()
	defer r.mu.Unlock()

	slices.SortFunc(r.FilesReports, func(a, b *FileReport) int {
		return strings.Compare(a.Name, b.Name)
	})
	for _, fileReport := range r.FilesReports {
		// Lock each FileReport to protect its GroupReports slice
		fileReport.mu.Lock()
		slices.SortFunc(fileReport.GroupReports, func(a, b *GroupReport) int {
			return strings.Compare(a.Name, b.Name)
		})
		for _, groupReport := range fileReport.GroupReports {
			// Lock each GroupReport to protect its RuleReports slice
			groupReport.mu.Lock()
			slices.SortFunc(groupReport.RuleReports, func(a, b *RuleReport) int {
				return strings.Compare(a.Name, b.Name)
			})
			groupReport.mu.Unlock()
		}
		fileReport.mu.Unlock()
	}
}

type FileReport struct {
	Name                    string         `json:"file_name" yaml:"file_name"`
	Valid                   *atomic.Bool   `json:"valid" yaml:"valid"`
	Excluded                *atomic.Bool   `json:"excluded" yaml:"excluded"`
	Errors                  []*Error       `json:"errors" yaml:"errors"`
	HasRuleValidationErrors *atomic.Bool   `json:"has_rule_validation_errors" yaml:"has_rule_validation_errors"`
	GroupReports            []*GroupReport `json:"group_reports" yaml:"group_reports"`

	mu sync.Mutex `json:"-" yaml:"-"`
}

func (r *FileReport) NewGroupReport(name string) *GroupReport {
	newReport := &GroupReport{
		Name:        name,
		Valid:       atomic.NewBool(true),
		Excluded:    atomic.NewBool(false),
		RuleReports: []*RuleReport{},
		Errors:      []*Error{},
	}

	r.mu.Lock()
	r.GroupReports = append(r.GroupReports, newReport)
	r.mu.Unlock()
	return newReport
}

// AddError safely adds an error to the file report.
func (r *FileReport) AddError(err *Error) {
	r.mu.Lock()
	r.Errors = append(r.Errors, err)
	r.mu.Unlock()
}

// AddErrors safely adds multiple errors to the file report.
func (r *FileReport) AddErrors(errs []*Error) {
	if len(errs) == 0 {
		return
	}
	r.mu.Lock()
	r.Errors = append(r.Errors, errs...)
	r.mu.Unlock()
}

func (r *FileReport) AsText(output *IndentedOutput) {
	if r.Valid.Load() {
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
	Valid       *atomic.Bool  `json:"valid" yaml:"valid"`
	Name        string        `json:"group_name" yaml:"group_name"`
	Excluded    *atomic.Bool  `json:"excluded" yaml:"excluded"`
	RuleReports []*RuleReport `json:"rule_reports" yaml:"rule_reports"`
	Errors      []*Error      `json:"errors" yaml:"errors"`

	mu sync.Mutex `json:"-" yaml:"-"`
}

func (r *GroupReport) NewRuleReport(name string, ruleType config.ValidationScope) *RuleReport {
	newReport := &RuleReport{
		Name:     name,
		Valid:    atomic.NewBool(true),
		RuleType: ruleType,
		Excluded: atomic.NewBool(false),
		Errors:   []*Error{},
	}

	r.mu.Lock()
	r.RuleReports = append(r.RuleReports, newReport)
	r.mu.Unlock()
	return newReport
}

// AddError safely adds an error to the group report.
func (r *GroupReport) AddError(err *Error) {
	r.mu.Lock()
	r.Errors = append(r.Errors, err)
	r.mu.Unlock()
}

// AddErrors safely adds multiple errors to the group report.
func (r *GroupReport) AddErrors(errs []*Error) {
	if len(errs) == 0 {
		return
	}
	r.mu.Lock()
	r.Errors = append(r.Errors, errs...)
	r.mu.Unlock()
}

func (r *GroupReport) AsText(output *IndentedOutput) {
	if r.Valid.Load() {
		return
	}
	output.AddLine("Group: " + r.Name)
	output.IncreaseIndentation()
	defer output.DecreaseIndentation()
	if r.Excluded.Load() {
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
	Valid    *atomic.Bool           `json:"valid" yaml:"valid"`
	RuleType config.ValidationScope `json:"rule_type" yaml:"rule_type"`
	Name     string                 `json:"name" yaml:"name"`
	Excluded *atomic.Bool           `json:"excluded" yaml:"excluded"`
	Errors   []*Error               `json:"errors" yaml:"errors"`

	mu sync.Mutex `json:"-" yaml:"-"`
}

func (r *RuleReport) AsText(output *IndentedOutput) {
	if r.Valid.Load() {
		return
	}
	output.AddLine(string(r.RuleType) + ": " + r.Name)
	output.IncreaseIndentation()
	defer output.DecreaseIndentation()
	if r.Excluded.Load() {
		output.AddLine("Skipped")
		return
	}
	output.WriteErrors(r.Errors)
}

// AddError safely adds an error to the rule report.
func (r *RuleReport) AddError(err *Error) {
	r.mu.Lock()
	r.Errors = append(r.Errors, err)
	r.mu.Unlock()
}

// AddErrors safely adds multiple errors to the rule report.
func (r *RuleReport) AddErrors(errs []*Error) {
	if len(errs) == 0 {
		return
	}
	r.mu.Lock()
	r.Errors = append(r.Errors, errs...)
	r.mu.Unlock()
}

func (r *ValidationReport) AsText(indentationStep int, color bool) (string, error) {
	r.Sort()

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

	if r.Failed.Load() {
		output.AddErrorLine("Validation FAILED")
	} else {
		output.AddSuccessLine("Validation PASSED")
	}
	output.AddLine("Statistics:")
	output.IncreaseIndentation()
	output.AddLine("Duration: " + r.Duration.Load().String())
	output.AddLine(renderStatistic("Files", int(r.FilesCount.Load()), int(r.FilesExcludedCount.Load())))
	output.AddLine(renderStatistic("Groups", int(r.GroupsCount.Load()), int(r.GroupsExcludedCount.Load())))
	output.AddLine(renderStatistic("Rules", int(r.RulesCount.Load()), int(r.RulesExcludedCount.Load())))
	return output.Text(), nil
}

func renderStatistic(objectType string, total, excluded int) string {
	return fmt.Sprintf("%s: %d and %d of them excluded", objectType, total, excluded)
}

func (r *ValidationReport) AsJSON() (string, error) {
	r.Sort()

	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	buffer := bytes.NewBuffer(b)
	return buffer.String(), nil
}

func (r *ValidationReport) AsYaml() (string, error) {
	r.Sort()

	b, err := yaml.Marshal(r)
	if err != nil {
		return "", err
	}
	buffer := bytes.NewBuffer(b)
	return buffer.String(), nil
}
