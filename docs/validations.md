# Supported validations by scopes
All the supported validations are listed here. The validations are grouped by the scope they can be used with.

> If you want some sane default validations, you can look at the [default_validation.yaml](./default_validation.yaml). Those should be a good starting point for your own configuration and applicable to most of the use cases.

- [Supported validations by scopes](#supported-validations-by-scopes)
  - [Groups](#groups)
    - [`hasValidSourceTenants`](#hasvalidsourcetenants)
    - [`hasAllowedEvaluationInterval`](#hasallowedevaluationinterval)
    - [`hasValidPartialResponseStrategy`](#hasvalidpartialresponsestrategy)
    - [`maxRulesPerGroup`](#maxrulespergroup)
    - [`hasValidLimit`](#hasvalidlimit)
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
      - [`expressionDoesNotUseMetrics`](#expressiondoesnotusemetrics)
      - [`expressionDoesNotUseLabels`](#expressiondoesnotuselabels)
      - [`expressionDoesNotUseOlderDataThan`](#expressiondoesnotuseolderdatathan)
      - [`expressionDoesNotUseRangeShorterThan`](#expressiondoesnotuserangeshorterthan)
      - [`expressionDoesNotUseIrate`](#expressiondoesnotuseirate)
      - [`validFunctionsOnCounters`](#validfunctionsoncounters)
      - [`rateBeforeAggregation`](#ratebeforeaggregation)
      - [`expressionCanBeEvaluated`](#expressioncanbeevaluated)
      - [`expressionUsesExistingLabels`](#expressionusesexistinglabels)
      - [`expressionSelectorsMatchesAnything`](#expressionselectorsmatchesanything)
      - [`expressionWithNoMetricName`](#expressionwithnometricname)
      - [`expressionIsWellFormatted`](#expressioniswellformatted)
    - [Other](#other)
      - [`hasSourceTenantsForMetrics`](#hassourcetenantsformetrics)
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
  - [Recording rules validators](#recording-rules-validators)



## Groups
Usage of the following validations is limited to the `Group` scope.

:warning: Can be used only with the `Group` scope.

### `hasValidSourceTenants`

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
  regexp: ".*"
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

#### `expressionDoesNotUseMetrics`

Fails if the rule expression uses metrics matching any of the metric name fully anchored(will be surrounded by `^...$`) regexps.
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

#### `expressionCanBeEvaluated`

> Queries live prometheus instance, requires the `prometheus` config to be set.

This validation runs the expression against the actual Prometheus instance and checks if it ends with error.
Possibly you can set maximum allowed query execution time and maximum number of resulting time series.

```yaml
params:
  timeSeriesLimit: 100
  evaluationDurationLimit: 1m
```

#### `expressionUsesExistingLabels`

> Queries live prometheus instance, requires the `prometheus` config to be set.

Fails if any used label is not present in the configured Prometheus instance.

#### `expressionSelectorsMatchesAnything`

> Queries live prometheus instance, requires the `prometheus` config to be set.

Verifies if any of the selectors in the expression (eg `up{foo="bar"}`) matches actual data in the configured Prometheus
instance.

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

### Other

#### `hasSourceTenantsForMetrics`

Fails, if the rule uses metric, that matches the specified regular expression for any tenant, but does not have the tenant configured in the  `source_tenants` of the rule group option the rule belongs to.
> If you use Mimir, and know, that the metrics are coming from specific tenants, you can make sure the tenants are configured in the rule group `source_tenants` option.

```yaml
params:
  sourceTenants:
    <tenant_name>: <metric_name_regexp> # The regexp will be fully anchored (surrounded by ^...$)
    # Example:
    # k8s: "kube_.*|container_.*"
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

Fails if annotation value is not a valid URL. If `resolveURL` is enabled, tries to make an HTTP request to the specified
URL and fails if the request does not succeed or returns 404 HTTP status code.
> It's common practice to link a playbook with guide how to solve the alert in the alert itself.
> This way you can verify it's a working URL and possibly if it really exists.

```yaml
params:
  annotation: "playbook"
  resolveUrl: true
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

## Recording rules validators
Validators that can be used on `Recording rule` scope.
