package validator

import (
	"testing"

	"github.com/fusakla/promruval/v3/pkg/config"
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
