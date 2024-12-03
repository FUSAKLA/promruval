package report

import (
	"errors"
	"testing"

	"github.com/fusakla/promruval/v3/pkg/config"
	"github.com/stretchr/testify/assert"
)

var report = &ValidationReport{
	Failed:              true,
	Duration:            123,
	ErrorsCount:         5,
	FilesCount:          2,
	FilesExcludedCount:  1,
	GroupsCount:         3,
	GroupsExcludedCount: 1,
	RulesCount:          4,
	RulesExcludedCount:  2,
	ValidationRules: []ValidationRule{
		MockValidationRule{
			name:            "Rule1",
			validationTexts: []string{"Rule1Text1", "Rule1Text2"},
			scope:           config.AllScope,
		},
		MockValidationRule{
			name:            "Rule2",
			validationTexts: []string{"Rule2Text1"},
			scope:           config.GroupScope,
		},
	},
	FilesReports: []*FileReport{
		{
			Name:     "file1.yaml",
			Valid:    false,
			Excluded: false,
			Errors: []error{
				errors.New("file error 1"),
				errors.New("file error 2"),
			},
			GroupReports: []*GroupReport{
				{
					Valid:    false,
					Name:     "Group1",
					Excluded: true,
					Errors: []error{
						errors.New("group error 1"),
					},
					RuleReports: []*RuleReport{
						{
							Valid:    false,
							RuleType: "Alert",
							Name:     "Rule1",
							Excluded: false,
							Errors: []error{
								errors.New("rule error 1"),
								errors.New("rule error 2"),
							},
						},
					},
				},
			},
		},
	},
}

func TestSerializeValidationReport(t *testing.T) {
	// Test case: Full serialization with nested structures.
	t.Run("FullSerialization", func(t *testing.T) {
		// Call SerializeValidationReport.
		serialized, err := report.SerializeValidationReport()
		assert.NoError(t, err)

		// Expected result
		expected := SerializableValidationReport{
			Failed:              true,
			Duration:            123,
			ErrorsCount:         5,
			FilesCount:          2,
			FilesExcludedCount:  1,
			GroupsCount:         3,
			GroupsExcludedCount: 1,
			RulesCount:          4,
			RulesExcludedCount:  2,
			ValidationRules: []SerializableValidationRule{
				{
					Name:            "Rule1",
					ValidationTexts: []string{"Rule1Text1", "Rule1Text2"},
				},
				{
					Name:            "Rule2",
					ValidationTexts: []string{"Rule2Text1"},
				},
			},
			FilesReports: []*SerializableFileReport{
				{
					Name:     "file1.yaml",
					Valid:    false,
					Excluded: false,
					Errors: []string{
						"File error 1",
						"File error 2",
					},
					GroupReports: []*SerializableGroupReport{
						{
							Valid:    false,
							Name:     "Group1",
							Excluded: true,
							Errors: []string{
								"Group error 1",
							},
							RuleReports: []*SerializableRuleReport{
								{
									Valid:    false,
									RuleType: "Alert",
									Name:     "Rule1",
									Excluded: false,
									Errors: []string{
										"Rule error 1",
										"Rule error 2",
									},
								},
							},
						},
					},
				},
			},
		}

		// Validate the result.
		assert.Equal(t, expected, serialized)
	})

	// Test case: Empty report.
	t.Run("EmptyReport", func(t *testing.T) {
		report := &ValidationReport{}

		serialized, err := report.SerializeValidationReport()
		assert.NoError(t, err)

		// Expected result for empty report.
		expected := SerializableValidationReport{
			ValidationRules: []SerializableValidationRule{},
			FilesReports:    []*SerializableFileReport{},
		}
		assert.Equal(t, expected, serialized)
	})

	t.Run("EmptyValidationRules", func(t *testing.T) {
		report := &ValidationReport{
			ValidationRules: []ValidationRule{},
			FilesReports: []*FileReport{
				{
					Name:         "file1.yaml",
					Valid:        false,
					Errors:       []error{},
					GroupReports: []*GroupReport{},
				},
			},
		}

		serialized, err := report.SerializeValidationReport()
		assert.NoError(t, err)

		// Updated expected result.
		expected := SerializableValidationReport{
			ValidationRules: []SerializableValidationRule{},
			FilesReports: []*SerializableFileReport{
				{
					Name:                    "file1.yaml",
					Valid:                   false,
					Excluded:                false,
					Errors:                  []string{},
					HasRuleValidationErrors: false,
					GroupReports:            []*SerializableGroupReport{},
				},
			},
		}
		assert.Equal(t, expected, serialized)
	})

	// Test case: Empty errors.
	t.Run("EmtpyErrors", func(t *testing.T) {
		report := &ValidationReport{
			FilesReports: []*FileReport{
				{
					Errors: []error{},
				},
			},
		}

		serialized, err := report.SerializeValidationReport()
		assert.NoError(t, err)

		// Expected result.
		expected := SerializableValidationReport{
			FilesReports: []*SerializableFileReport{
				{
					Errors:       []string{},
					GroupReports: []*SerializableGroupReport{},
				},
			},
			ValidationRules: []SerializableValidationRule{},
		}
		assert.Equal(t, expected, serialized)
	})
}

func TestValidationReport_AsJSON(t *testing.T) {
	t.Run("ValidJSONOutput", func(t *testing.T) {
		// Execute: Generate the JSON string.
		jsonString, err := report.AsJSON()
		assert.NoError(t, err)

		// Expected JSON output.
		expectedJSON := `{
		  "Failed": true,
		  "Duration": 123,
		  "ErrorsCount": 5,
		  "FilesCount": 2,
		  "FilesExcludedCount": 1,
		  "GroupsCount": 3,
		  "GroupsExcludedCount": 1,
		  "RulesCount": 4,
		  "RulesExcludedCount": 2,
		  "ValidationRules": [
		    {
		      "Name": "Rule1",
		      "ValidationTexts": [
		        "Rule1Text1",
		        "Rule1Text2"
		      ]
		    },
		    {
		      "Name": "Rule2",
		      "ValidationTexts": [
		        "Rule2Text1"
		      ]
		    }
		  ],
		  "FilesReports": [
		    {
		      "Name": "file1.yaml",
		      "Valid": false,
		      "Excluded": false,
		      "Errors": [
		        "File error 1",
		        "File error 2"
		      ],
		      "HasRuleValidationErrors": false,
		      "GroupReports": [
		        {
		          "Valid": false,
		          "Name": "Group1",
		          "Excluded": true,
		          "RuleReports": [
		            {
		              "Valid": false,
		              "RuleType": "Alert",
		              "Name": "Rule1",
		              "Excluded": false,
		              "Errors": [
		                "Rule error 1",
		                "Rule error 2"
		              ]
		            }
		          ],
		          "Errors": [
		            "Group error 1"
		          ]
		        }
		      ]
		    }
		  ]
		}`

		// Assertion: Compare the JSON string with the expected value.
		assert.JSONEq(t, expectedJSON, jsonString)
	})
}

func TestValidationReport_AsYaml(t *testing.T) {
	t.Run("ValidYAMLOutput", func(t *testing.T) {
		// Execute: Generate the YAML string.
		yamlString, err := report.AsYaml()
		assert.NoError(t, err)

		// Expected YAML output.
		expectedYAML := `failed: true
duration: 123ns
errorscount: 5
filescount: 2
filesexcludedcount: 1
groupscount: 3
groupsexcludedcount: 1
rulescount: 4
rulesexcludedcount: 2
validationrules:
    - name: Rule1
      validationtexts:
        - Rule1Text1
        - Rule1Text2
    - name: Rule2
      validationtexts:
        - Rule2Text1
filesreports:
    - name: file1.yaml
      valid: false
      excluded: false
      errors:
        - File error 1
        - File error 2
      hasrulevalidationerrors: false
      groupreports:
        - valid: false
          name: Group1
          excluded: true
          rulereports:
            - valid: false
              ruletype: Alert
              name: Rule1
              excluded: false
              errors:
                - Rule error 1
                - Rule error 2
          errors:
            - Group error 1
`

		// Assertion: Compare the YAML string with the expected value.
		assert.YAMLEq(t, expectedYAML, yamlString)
	})
}

// MockValidationRule implements ValidationRule interface for testing.
type MockValidationRule struct {
	name            string
	validationTexts []string
	scope           config.ValidationScope
}

// Name returns the name of the validation rule.
func (m MockValidationRule) Name() string {
	return m.name
}

// ValidationTexts returns the validation texts of the validation rule.
func (m MockValidationRule) ValidationTexts() []string {
	return m.validationTexts
}

// Scope returns the scope of the validation rule.
func (m MockValidationRule) Scope() config.ValidationScope {
	return m.scope
}
