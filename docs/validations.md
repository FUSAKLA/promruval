# Supported validations by scopes
All the supported validations are listed here. The validations are grouped by the scope they can be used with.

> If you want some sane default validations, you can look at the [default_validation.yaml](./default_validation.yaml). Those should be a good starting point for your own configuration and applicable to most of the use cases.

> Please note that unless explicitly stated otherwise, all regular expressions used in configuration are fully anchored `^...$` and defaults to match everything `.*`.

- [Supported validations by scopes](#supported-validations-by-scopes)
  - [Groups](#groups)
    - [`hasAllowedSourceTenants`](#hasallowedsourcetenants)
    - [`hasAllowedEvaluationInterval`](#hasallowedevaluationinterval)
    - [`hasValidPartialResponseStrategy`](#hasvalidpartialresponsestrategy)
    - [`maxRulesPerGroup`](#maxrulespergroup)
    - [`hasValidLimit`](#hasvalidlimit)
    - [`groupNameMatchesRegexp`](#groupnamematchesregexp)
    - [`hasAllowedQueryOffset`](#hasallowedqueryoffset)
  - [Universal rule validators](#universal-rule-validators)
    - [Labels](#labels)
      - [`hasLabels`](#haslabels)
      - [`doesNotHaveLabels`](#doesnothavelabels)
      - [`hasAnyOfLabels`](#hasanyoflabels)
      - [`labelMatchesRegexp`](#labelmatchesregexp)
      - [`labelHasAllowedValue`](#labelhasallowedvalue)
      - [`nonEmptyLabels`](#nonemptylabels)
      - [`exclusiveLabels`](#exclusivelabels)
    - [PromQL expression validators](#promql-expression-validators)
      - [`expressionIsValidPromQL`](#expressionisvalidpromql)
      - [`expressionDoesNotUseExperimentalFunctions`](#expressiondoesnotuseexperimentalfunctions)
      - [`expressionDoesNotUseMetrics`](#expressiondoesnotusemetrics)
      - [`expressionDoesNotUseLabels`](#expressiondoesnotuselabels)
      - [`expressionUsesOnlyAllowedLabelsForMetricRegexp`](#expressionusesonlyallowedlabelsformetricregexp)
      - [`expressionDoesNotUseLabelsForMetricRegexp`](#expressiondoesnotuselabelsformetricregexp)
      - [`expressionUsesOnlyAllowedLabelValuesForMetricRegexp`](#expressionusesonlyallowedlabelvaluesformetricregexp)
      - [`expressionDoesNotUseOlderDataThan`](#expressiondoesnotuseolderdatathan)
      - [`expressionDoesNotUseRangeShorterThan`](#expressiondoesnotuserangeshorterthan)
      - [`expressionDoesNotUseIrate`](#expressiondoesnotuseirate)
      - [`validFunctionsOnCounters`](#validfunctionsoncounters)
      - [`rateBeforeAggregation`](#ratebeforeaggregation)
      - [`expressionUsesUnderscoresInLargeNumbers`](#expressionusesunderscoresinlargenumbers)
      - [`expressionWithNoMetricName`](#expressionwithnometricname)
      - [`expressionIsWellFormatted`](#expressioniswellformatted)
      - [`expressionDoesNotUseClassicHistogramBucketOperations`](#expressiondoesnotuseclassichistogrambucketoperations)
    - [PromQL expression validators (using live Prometheus instance)](#promql-expression-validators-using-live-prometheus-instance)
      - [`expressionCanBeEvaluated`](#expressioncanbeevaluated)
      - [`expressionUsesExistingLabels`](#expressionusesexistinglabels)
      - [`expressionSelectorsMatchesAnything`](#expressionselectorsmatchesanything)
    - [LogQL expression validators](#logql-expression-validators)
      - [`expressionIsValidLogQL`](#expressionisvalidlogql)
      - [`logQlExpressionUsesRangeAggregation`](#logqlexpressionusesrangeaggregation)
    - [Other](#other)
      - [`hasSourceTenantsForMetrics`](#hassourcetenantsformetrics)
      - [`doesNotContainTypos`](#doesnotcontaintypos)
  - [Alert validators](#alert-validators)
    - [Labels](#labels-1)
      - [`validateLabelTemplates`](#validatelabeltemplates)
    - [Annotations](#annotations)
      - [`hasAnnotations`](#hasannotations)
      - [`doesNotHaveAnnotations`](#doesnothaveannotations)
      - [`hasAnyOfAnnotations`](#hasanyofannotations)
      - [`annotationMatchesRegexp`](#annotationmatchesregexp)
      - [`annotationHasAllowedValue`](#annotationhasallowedvalue)
      - [`annotationIsValidURL`](#annotationisvalidurl)
      - [`annotationIsValidPromQL`](#annotationisvalidpromql)
      - [`validateAnnotationTemplates`](#validateannotationtemplates)
    - [Other](#other-1)
      - [`forIsNotLongerThan`](#forisnotlongerthan)
      - [`keepFiringForIsNotLongerThan`](#keepfiringforisnotlongerthan)
      - [`alertNameMatchesRegexp`](#alertnamematchesregexp)
  - [Recording rules validators](#recording-rules-validators)
      - [`recordedMetricNameMatchesRegexp`](#recordedmetricnamematchesregexp)
      - [`recordedMetricNameDoesNotMatchRegexp`](#recordedmetricnamedoesnotmatchregexp)



## Groups
Usage of the following validations is limited to the `Group` scope.

:warning: Can be used only with the `Group` scope.

### `hasAllowedSourceTenants`

Fails if the rule group has other than than the configured source tenants.
> If using Mimir, you may want to check that only the known tenants are used to avoid typos for example.

```yaml
params:
  allowedSourceTenants: [ "foo", "bar" ]
```

### `hasAllowedEvaluationInterval`

Fails if the rule group has the `interval` out of the configured range.
By default it will ignore, if the group does not have the interval configured.
You can enforce it to be set by setting the `mustBeSet` to true.
> Useful to avoid using too short or too long evaluation intervals such as `1s` which would most certainly lead to missed evaluation intervals.
> Enforcint the interval to be configured per group can force the user to think about how often they really need the rules to be evaluated.

```yaml
params:
  minimum: "0s"
  maximum: <duration> # Optional, default is infinity
  mustBeSet: false
```

### `hasValidPartialResponseStrategy`

Fails if the rule group has invalid value of the `partial_response_strategy` option, if set.
To enforce the `partial_response_strategy` to be set, set the `mustBeSet` to true.

```yaml
params:
  mustBeSet: false
```

### `maxRulesPerGroup`

Fails if the rule group has more rules than the specified limit.
> Since the rules in one rule group are evaluated sequentially, it's a good practice to split the rules to smaller groups.
> This way the evaluation will be parallelized and the evaluation time will be shorter.

```yaml
params:
  limit: 10
```

### `hasValidLimit`

Fails if the rule group has the `limit` option set higher, then the specified limit.
If not set at all, it will fail also, since the default limit is 0 meaning unlimited.
> It's a good practice to limit the number of alerts in the group to avoid overloading the Alertmanager of event receivers, which can rate-limit.
> In case of recording rules, can help to avoid generating huge amount of time series.

```yaml
params:
  limit: 10
```

### `groupNameMatchesRegexp`

Fails if the group name does not match the specified regular expression.

```yaml
params:
  regexp: "[A-Z]\s+" # defaults to ""
```

### `hasAllowedQueryOffset`

Fails if the rule group has the `query_offset` out of the configured range.

```yaml
params:
  minimum: <duration>
  maximum: <duration> # Optional, default is infinity
```

## Universal rule validators
Validators that can be used on  `All rules`, `Recording rule` and `Alert` scopes.

### Labels

#### `hasLabels`

Fails if rule does not have all the specified labels. Is `searchInExpr` is set, the labels are also looked for in the
rules `expr`.
> Make sure every alert has all the labels required for it to be correctly routed by the Alertmanager.

```yaml
params:
  labels: [ "foo", "bar" ]
  searchInExpr: true
```

#### `doesNotHaveLabels`

Fails if rule has any of specified labels. Is `searchInExpr` is set, the labels are also looked for in the rules `expr`.
> In case of deprecating some old well-known labels used formerly for routing for example,
> you can make sure no one will use them by mistake again.

```yaml
params:
  labels: [ "foo", "bar" ]
  searchInExpr: true
```

#### `hasAnyOfLabels`

Fails if rule does not have any of specified labels.

```yaml
params:
  labels: [ "foo", "bar" ]
```

#### `labelMatchesRegexp`

Fails if rule label does not match the specified regular expression.
> If you for example use a `team` label containing email of the specific team,
> you can use a regular expression to verify its form.

```yaml
params:
  label: "foo"
  regexp: ".*" # defaults to ""
```

#### `labelHasAllowedValue`

Fails if rule label value is not one of the allowed values. If the `commaSeparatedValue` is set to true, the label value
to true, the label value is split by a comma, and the distinct values are checked if valid.
Since the labels can be templated, but Promruval cannot tell if the resulting value will be valid,
there is the `ignoreTemplatedValues` option, that allows you to ignore the templated values.

> It's quite common to have well known severities for alerts which can be important even in the
> Alertmanager routing tree. Ths is how you can make sure only the well-known severities are used.

```yaml
params:
  label: "foo"
  allowedValues: [ "foo", "bar" ]
  commaSeparatedValue: true
  ignoreTemplatedValues: false
```

#### `nonEmptyLabels`

Fails if any label has empty value. It has no effect and is dropped by Prometheus.

#### `exclusiveLabels`

Fails if the rule has the first label and also the second one.
You can also optionally specify event the value of those labels.

Example: If alert has label `severity` with value `critical` cannot have label `page` with value `true`

```yaml
params:
  firstLabel: "severity"
  firstLabelValue: "critical" # Optional, if not set, only presence of the label excludes the second label
  secondLabel: "page"
  secondLabelValue: "true" # Optional, if set, fails only if also the second label value matches
```
### PromQL expression validators

#### `expressionIsValidPromQL`

Fails if the expression is not a valid PromQL query.

#### `expressionDoesNotUseExperimentalFunctions`

Fails if the rule expression uses any of the experimental PromQL functions.

#### `expressionDoesNotUseMetrics`

Fails if the rule expression uses metrics matching any of the metric name regexps. Empty list does not match anything. All regexps within the list will be fully anchored (surrounded by `^...$`), empty string (`""`) **won't be converted to all match (`.*`)**.
> If you want to avoid using some metrics in the rules, you can use this validation to make sure it won't happen.

```yaml
params:
  metricNameRegexps: [ "foo_bar.*", "foo_baz" ]
```

#### `expressionDoesNotUseLabels`

Fails if the rule uses any of specified labels in its `expr` label matchers, aggregations or joins.
> If using Thanos, users has to know if the rule is evaluated by Prometheus or Thanos,
> but Prometheus cannot use the external labels. This way you can make sure it won't happen.

```yaml
params:
  labels: [ "foo", "bar" ]
```

#### `expressionUsesOnlyAllowedLabelsForMetricRegexp`

Fails if the rule uses any labels beside those listed in `allowedLabels`, in combination with given metric regexp in its `expr` label matchers, aggregations or joins. If the metric name is omitted in the query, or matched using regexp or any negative matcher on the `__name__` label, the rule will be skipped.

The check rather ignores validation of labels, where it cannot be sure if they are targeting only the metric in question, like aggregations by labels on top of vector matching expression where the labels might come from the other part of the expr.

> If using kube-state-metrics for exposing labels information about K8S objects (kube_*_labels) only those labels whitelisted by kube-state-metrics admin will be available.
> Might be useful to check that users does not use any other in their expressions.

```yaml
params:
  metricNameRegexp: "kube_pod_labels"
  allowedLabels: [ "pod", "cluster", "app", "team" ]
```

#### `expressionDoesNotUseLabelsForMetricRegexp`

Fails if the rule uses any labels listed in `labels`, in combination with given metric regexp in its `expr` label matchers, aggregations or joins. If the metric name is omitted in the query, or matched using regexp or any negative matcher on the `__name__` label, the rule will be skipped.

The check rather ignores validation of labels, where it cannot be sure if they are targeting only the metric in question, like aggregations by labels on top of vector matching expression where the labels might come from the other part of the expr.

> Might be useful to make sure users does not use labels which are subject to change.

```yaml
params:
  metricNameRegexp: "kube_.+"
  labels: [ "job", "cluster", "app", "instance" ]
```

#### `expressionUsesOnlyAllowedLabelValuesForMetricRegexp`
Fails if the metrics matching given regexp uses label selectors for given labels does not match at least one of given values. Regexp match in label selector (`=~`) is evaluated as a regexp against a given list of allowed label values. Negative regexp matches (`!~`) are ignored.

```yaml
params:
  metricNameRegexp: "kube_pod_labels"
  allowedLabelValues:
    cluster: ['kube1', 'kube2', 'kube3']
    team: ['sre', 'backend']
```

#### `expressionDoesNotUseOlderDataThan`

Fails if the rule `expr` uses older data than specified limit in Prometheus duration syntax. Checks even in sub-queries
and offsets.
> Useful to avoid writing queries which expects longer data retention than the Prometheus actually has.

```yaml
params:
  limit: "12h"
```

#### `expressionDoesNotUseRangeShorterThan`

Fails if the rule `expr` uses shorter range than specified limit in the Prometheus duration format.
> Useful to avoid using shorter range than twice of the scrape interval.

```yaml
params:
  limit: "1m"
```

#### `expressionDoesNotUseIrate`

Fails if the rule `expr` uses the `irate` function as discouraged
in https://prometheus.io/docs/prometheus/latest/querying/functions/#irate.
> It's not recommended to use `irate` function in the rules.

#### `validFunctionsOnCounters`

Fails if the expression uses a `rate` or `increase` function on a metric that does not end with the `_total` suffix.
> It's a common mistake to use the `rate` or `increase` function on a metric that is not a counter.
> This validation can help to avoid it.

#### `rateBeforeAggregation`

Fails if aggregation function is used before the `rate` or `increase` functions.
> Avoid common mistake of using aggregation function before the `rate` or `increase` function.

#### `expressionUsesUnderscoresInLargeNumbers`

Fails if the query containes numbers higher than 1000 without using underscores as separators for better readability.
Ignores numbers in the `10e2` and duration format.

#### `expressionWithNoMetricName`

Fails if an expression doesn't use an explicit metric name (also if used as `__name__` label) in all its selectors(eg `up{foo="bar"}`).
> Such queries may be very expensive and can lead to performance issues.

#### `expressionIsWellFormatted`

Fails if the expression is not well formatted PromQL as would `promtool promql format` do.
It does remove the comments from the expression before the validation, since the PromQL prettifier drops them, so this should avoid false positive diffs.
But if you want to ignore the expressions with comments, you can set the `ignoreComments` to true.
> Useful to make sure the expressions are formatted in a consistent way.

```yaml
params:
  showExpectedForm: true # Optional, will show how the query should be formatted
  skipExpressionsWithComments: true # Optional, will skip the expressions with comments
```

#### `expressionDoesNotUseClassicHistogramBucketOperations`

Fails if the expression does any binary operation between bucket metrics of a classical histogram.

> There are situations when the classic histogram is not atomic (for example remote write), this it may result in unexpected results.
> This calculation is often used to calculate SLOs a a difference between the `+Inf` bucket and one of the buckets which is the SLO threshold.
> To avoid this issue, it's recommended to calculate such differences before sending the data over the remote write for example.

### PromQL expression validators (using live Prometheus instance)

All these validations require the ` prometheus` sectiong in the config to be set.

#### `expressionCanBeEvaluated`

> Queries live prometheus instance, requires the `prometheus` config to be set.

This validation runs the expression against the actual Prometheus instance and checks if it ends with error.
Possibly you can set maximum allowed query execution time and maximum number of resulting time series.

```yaml
params:
  timeSeriesLimit: 100 # Optional, maximum series returned by the query
  evaluationDurationLimit: 1m # Optional, maximum duration of the query evaluation
```

#### `expressionUsesExistingLabels`

> Queries live prometheus instance, requires the `prometheus` config to be set.

Fails if any used label is not present in the configured Prometheus instance.

#### `expressionSelectorsMatchesAnything`

> Queries live prometheus instance, requires the `prometheus` config to be set.

Verifies if any of the selectors in the expression (eg `up{foo="bar"}`) matches actual data in the configured Prometheus
instance.

```yaml
params:
  maximumMatchingSeries: 1000 # Optional, maximum number of matching series for single selector used in expression
```

### LogQL expression validators

#### `expressionIsValidLogQL`

Fails if the expression is not a valid LogQL query.

#### `logQlExpressionUsesRangeAggregation`

Fails if the LogQL expression does not use any [range aggregation function](https://grafana.com/docs/loki/latest/query/metric_queries/#log-range-aggregations), which is required if used in rules.


### Other

#### `hasSourceTenantsForMetrics`

Fails, if the rule uses metric, that matches the specified regular expression for any tenant, but does not have the tenant configured in the  `source_tenants` of the rule group option the rule belongs to.
> If you use Mimir, and know, that the metrics are coming from specific tenants, you can make sure the tenants are configured in the rule group `source_tenants` option.

```yaml
params:
  defaultTenant: <tenant_name> # Optional, if set, the tenant that will be assumed if the group does not have the `source_tenants` option set
  sourceTenants:
    <tenant_name>:
      - regexp: <metric_name_regexp>
        negativeRegexp: <metric_name_regexp> # Optional, metrics matching the regexp will be excluded from the check, defaults to ""
        description: <description> # Optional, will be shown in the validator output human-readable description
  # Example:
  # k8s:
  #   - regexp: "kube_.*|container_.*"
  #     description: "Metrics from KSM"
  #   - regexp: "container_.*"
  #     description: "Metrics from cAdvisor"
  #   - regexp: "kafka_.*"
  #   - regexp: "node_.*"
  #     description: "Node exporter metrics provided by the k8s infrastructure team"
  # kafka:
  #   - regexp: "kafka_.*"
  #     negativeRegexp: "kafka_(consumer|producer)_.*"
  #     description: "Metrics from Kafka"
```

#### `doesNotContainTypos`

Fails, if any of the well-known labels, annotations or its values does contains a typo (cace sensitive).
Typo is identified by computing the [Levenshtein distance](https://en.wikipedia.org/wiki/Levenshtein_distance) between the actual and well-known value.
You can specify either maximum distance or maximum ratio of the distance to the length of the actual value. If the distance is lower then the threshold, the validation will fail assuming it is a typo.

```yaml
params:
  # Only maxLevenshteinDistance or maxDifferenceRatio can be set, not both
  maxLevenshteinDistance: <int> #  Optional, configures maximum distance, below is considered a typo
  maxDifferenceRatio: <float> # Optional, configures maximum ratio (0-1) of the distance to the length of the actual value, below is considered a typo
  wellKnownAnnotations: ['playbook', 'dashboard', 'title', 'description'] # Optional, well-known values to check rule annotations names
  wellKnownRuleLabels: ['do_not_inhibit']  # Optional, well-known values to check rule labels names
  wellKnownSeriesLabels: ['pod', 'locality', 'cluster']  # Optional, well-known values to check series labels names
```

## Alert validators
Validators that can be used on `Alert` scope.

### Labels

#### `validateLabelTemplates`

Fails if the label contains invalid Go template.

### Annotations

#### `hasAnnotations`

Fails if rule does not have all the specified annotations.
> Alertmanager templates often expects some specific annotations, so they can be rendered correctly.
> Make sure all alerts has those!

```yaml
params:
  annotations: [ "foo", "bar" ]
```

#### `doesNotHaveAnnotations`

Fails if rule has any of specified annotations.

```yaml
params:
  annotations: [ "foo", "bar" ]
```

#### `hasAnyOfAnnotations`

Fails if rule does not have any of specified annotations.

```yaml
params:
  annotations: [ "foo", "bar" ]
```

#### `annotationMatchesRegexp`

Fails if rule annotation value does not match the specified regular expression.

```yaml
params:
  annotation: "foo"
  regexp: ".*"
```

#### `annotationHasAllowedValue`

Fails if rule annotation value is not one of the allowed values.

```yaml
params:
  annotation: "foo"
  allowedValues: [ "foo", "bar" ]
  commaSeparatedValue: true
```

#### `annotationIsValidURL`

Fails if annotation value is not a valid URL. If `resolveUrl` is enabled, tries to make an HTTP request to the specified
URL and fails if the request does not succeed or returns 404 HTTP status code.
> It's common practice to link a playbook with guide how to solve the alert in the alert itself.
> This way you can verify it's a working URL and possibly if it really exists.

If `asTemplate` is enabled, the annotation is parsed as a [Go text template](
https://pkg.go.dev/text/template). If the parsing fails (which can happen when
incorrect syntax is used, for example) the validation fails immediately.
Otherwise all templated parts of the annotation are replaced with an empty
string and the result must be a valid URL. Note that the template is never
executed, it is just parsed.
> This works best when the path and/or query parameters of the URL are
> templated; when the whole schema or hostname part of the URL is templated,
> the validation will fail.

`asTemplate` and `resolveUrl` cannot be both enabled at the same time.

```yaml
params:
  annotation: "playbook"
  resolveUrl: true
  asTemplate: false
```

#### `annotationIsValidPromQL`

Fails if the rule specified annotations does not contain valid PromQL if present.

```yaml
params:
  annotation: "foo"
```

#### `validateAnnotationTemplates`

Fails if the annotation contains invalid Go template.

### Other

#### `forIsNotLongerThan`

Fails if the alert uses longer `for` than the specified limit.
> Too long `for` makes the alerts more fragile.

```yaml
params:
  limit: "1h"
```

#### `keepFiringForIsNotLongerThan`

Fails if the alert uses longer `keep_firing_for` than the specified limit.

```yaml
params:
  limit: "1h"
```

#### `alertNameMatchesRegexp`

Fails if the alert name does not match the specified regular expression.

```yaml
params:
  regexp: "[A-Z]\s+"
```

## Recording rules validators
Validators that can be used on `Recording rule` scope.

#### `recordedMetricNameMatchesRegexp`

Fails if the name of the recorded metric does not match the specified regular expression.

```yaml
params:
  regexp: "[^:]+:[^:]+:[^:]+" # defaults to ""
```

#### `recordedMetricNameDoesNotMatchRegexp`

Fails if the name of the recorded metric matches the specified regular expression.

```yaml
params:
  regexp: "^foo_bar$" # defaults to ""
```
