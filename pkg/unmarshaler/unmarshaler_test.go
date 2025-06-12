package unmarshaler

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	loki "github.com/grafana/loki/v3/pkg/tool/rules/rwrulefmt"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestUnmarshalling(t *testing.T) {
	type testCase struct {
		name          string
		input         string
		beforeExecute func()
		afterExecute  func()
		expected      RulesFileWithComment
		error         bool
	}

	testCases := []testCase{
		{
			name: "valid rules file with rule and alert",
			input: `
groups:
  - name: group1
    interval: 10s
    query_offset: 5s
    rules:
      - alert: alert1
        expr: expr1
        for: 10m
        keep_firing_for: 1h
        labels:
          foo: bar
        annotations:
          foo: bar
  - name: group2
    rules:
      - record: record1
        expr: expr1
        labels:
          foo: bar
`,
			expected: RulesFileWithComment{
				RulesFile: RulesFile{
					Groups: GroupsWithComment{
						Groups: []RuleGroupWithComment{
							{
								RuleGroup: RuleGroup{
									Name:        "group1",
									Interval:    model.Duration(time.Second * 10),
									QueryOffset: model.Duration(time.Second * 5),
									Rules: []RuleWithComment{
										{
											rule: rulefmt.Rule{
												Alert:         "alert1",
												Expr:          "expr1",
												For:           model.Duration(time.Minute * 10),
												Labels:        map[string]string{"foo": "bar"},
												Annotations:   map[string]string{"foo": "bar"},
												KeepFiringFor: model.Duration(time.Hour),
											},
										},
									},
								},
							},
							{
								RuleGroup: RuleGroup{
									Name: "group2",
									Rules: []RuleWithComment{
										{
											rule: rulefmt.Rule{
												Record: "record1",
												Expr:   "expr1",
												Labels: map[string]string{"foo": "bar"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			error: false,
		},

		{
			name: "successfully load rule test file",
			input: `
rule_files: []
evaluation_interval: 1m
group_eval_order: ???
tests: []
`,
			expected: RulesFileWithComment{
				RulesFile: RulesFile{
					RuleFiles:          []interface{}{},
					EvaluationInterval: "1m",
					GroupEvalOrder:     "???",
					Tests:              []interface{}{},
				},
			},
			error: false,
		},

		// =================== THANOS =====================
		{
			name: "thanos disallowed fields: partial_response_strategy",
			input: `
groups:
  - name: group1
    partial_response_strategy: warn
    rules:
      - alert: alert1
        expr: expr1
`,
			error: true,
		},
		{
			name: "thanos invalid partial strategy",
			input: `
groups:
  - name: group1
    partial_response_strategy: foo
`,
			error: true,
		},
		{
			name:          "thanos allowed fields",
			beforeExecute: func() { SupportThanos(true) },
			afterExecute:  func() { SupportThanos(false) },
			input: `
groups:
  - name: group1
    partial_response_strategy: warn
`,
			expected: RulesFileWithComment{
				RulesFile: RulesFile{
					Groups: GroupsWithComment{
						Groups: []RuleGroupWithComment{
							{
								RuleGroup: RuleGroup{
									Name:                    "group1",
									PartialResponseStrategy: "warn",
								},
							},
						},
					},
				},
			},
			error: false,
		},

		// =================== LOKI =====================
		{
			name: "loki disallowed fields: namespace",
			input: `
namespace: foo
groups: []
`,
			error: true,
		},
		{
			name: "loki disallowed fields: remote_write",
			input: `
groups:
  - name: group1
    remote_write:
      - url: http://localhost:3100/loki/api/v1/push
`,
			error: true,
		},
		{
			name:          "loki allowed fields",
			beforeExecute: func() { SupportLoki(true) },
			afterExecute:  func() { SupportLoki(false) },
			input: `
namespace: foo
groups:
  - name: group1
    remote_write:
      - url: http://localhost:3100/loki/api/v1/push
`,
			expected: RulesFileWithComment{
				RulesFile: RulesFile{
					Namespace: "foo",
					Groups: GroupsWithComment{
						Groups: []RuleGroupWithComment{
							{
								RuleGroup: RuleGroup{
									Name: "group1",
									RWConfigs: []loki.RemoteWriteConfig{
										{
											URL: "http://localhost:3100/loki/api/v1/push",
										},
									},
								},
							},
						},
					},
				},
			},
			error: false,
		},

		// =================== Mimir =====================
		{
			name: "mimir disallowed fields: source_tenants",
			input: `
groups:
  - name: group1
    source_tenants: ["tenant1"]
`,
			error: true,
		},
		{
			name: "mimir invalid source_tenants",
			input: `
groups:
  - name: group1
    source_tenants: foo
`,
			error: true,
		},
		{
			name:          "mimir allowed fields",
			beforeExecute: func() { SupportMimir(true) },
			afterExecute:  func() { SupportMimir(false) },
			input: `
groups:
  - name: group1
    source_tenants: ["tenant1", "tenant2"]
`,
			expected: RulesFileWithComment{
				RulesFile: RulesFile{
					Groups: GroupsWithComment{
						Groups: []RuleGroupWithComment{
							{
								RuleGroup: RuleGroup{
									Name:          "group1",
									SourceTenants: []string{"tenant1", "tenant2"},
								},
							},
						},
					},
				},
			},
			error: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.beforeExecute != nil {
				tc.beforeExecute()
			}
			var actual RulesFileWithComment
			err := yaml.Unmarshal([]byte(tc.input), &actual)
			if tc.afterExecute != nil {
				tc.afterExecute()
			}
			if tc.error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if diff := cmp.Diff(tc.expected, actual, cmpopts.IgnoreUnexported(RulesFileWithComment{}, GroupsWithComment{}, RuleGroupWithComment{}, RuleWithComment{})); diff != "" {
					t.Errorf("Diff in unmarshalled struct: mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
