package validator

import (
	"testing"

	"github.com/fusakla/promruval/v3/pkg/config"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"
)

func TestScope(t *testing.T) {
	tests := []struct {
		name          string
		validatorName string
		expectedScope config.ValidationScope
	}{
		{
			name:          "AlertScope",
			validatorName: "alertNameMatchesRegexp",
			expectedScope: config.AlertScope,
		},
		{
			name:          "RecordingRuleScope",
			validatorName: "recordedMetricNameMatchesRegexp",
			expectedScope: config.RecordingRuleScope,
		},
		{
			name:          "GroupScope",
			validatorName: "groupNameMatchesRegexp",
			expectedScope: config.GroupScope,
		},
		{
			name:          "AllRulesScope",
			validatorName: "hasLabels",
			expectedScope: config.AllRulesScope,
		},
		{
			name:          "UnknownValidator",
			validatorName: "unknownValidator",
			expectedScope: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope := Scope(tt.validatorName)
			assert.Equal(t, tt.expectedScope, scope)
		})
	}
}

func TestMatchesScope(t *testing.T) {
	tests := []struct {
		name   string
		rule   rulefmt.Rule
		scope  config.ValidationScope
		result bool
	}{
		{
			name:   "GroupScope",
			rule:   rulefmt.Rule{},
			scope:  config.GroupScope,
			result: true,
		},
		{
			name:   "AllRulesScope",
			rule:   rulefmt.Rule{},
			scope:  config.AllRulesScope,
			result: true,
		},
		{
			name:   "AlertScope with Alert",
			rule:   rulefmt.Rule{Alert: "TestAlert"},
			scope:  config.AlertScope,
			result: true,
		},
		{
			name:   "AlertScope without Alert",
			rule:   rulefmt.Rule{},
			scope:  config.AlertScope,
			result: false,
		},
		{
			name:   "RecordingRuleScope with Record",
			rule:   rulefmt.Rule{Record: "TestRecord"},
			scope:  config.RecordingRuleScope,
			result: true,
		},
		{
			name:   "RecordingRuleScope without Record",
			rule:   rulefmt.Rule{},
			scope:  config.RecordingRuleScope,
			result: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchesScope(tt.rule, tt.scope)
			assert.Equal(t, tt.result, result)
		})
	}
}
