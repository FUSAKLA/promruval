# Supported validations

### `hasLabels`
Fails if rule does not have all the specified labels. Is `searchInExpr` is set, the labels
are also looked for in the rules `expr`. 
> Make sure every alert has all the labels required for it to be correctly routed by the Alertmanager. 
```yaml
params:
  labels: ["foo", "bar"]
  searchInExpr: true
```

### `hasAnnotations`
Fails if rule does not have all the specified annotations.
> Alertmanager templates often expects some specific annotations, so they can be rendered correctly.
> Make sure all alerts has those!
```yaml
params:
  annotations: ["foo", "bar"]
```

### `doesNotHaveLabels`
Fails if rule has any of specified labels. Is `searchInExpr` is set, the labels
are also looked for in the rules `expr`. 
> In case of deprecating some old well-known labels used formerly for routing for example,
>  you can make sure no one will use them by mistake again.
```yaml
params:
  labels: ["foo", "bar"]
  searchInExpr: true
```

### `doesNotHaveAnnotations`
Fails if rule has any of specified annotations.
```yaml
params:
  annotations: ["foo", "bar"]
```

### `hasAnyOfLabels`
Fails if rule does not have any of specified labels.
```yaml
params:
  labels: ["foo", "bar"]
```

### `hasAnyOfAnnotations`
Fails if rule does not have any of specified annotations.
```yaml
params:
  annotations: ["foo", "bar"]
```

### `labelMatchesRegexp`
Fails if rule label does not match the specified regular expression.
> If you for example use a `team` label containing email of the specific team,
> you can use a regular expression to verify its form. 
```yaml
params:
  label: "foo"
  regexp: ".*"
```


### `annotationMatchesRegexp`
Fails if rule annotation value does not match the specified regular expression.
```yaml
params:
  annotation: "foo"
  regexp: ".*"
```


### `labelHasAllowedValue`
Fails if rule label value is not one of the allowed values. If the `commaSeparatedValue` is set to true, the label value
to true, the label vaue is split by a comma, and the distinct values are check if valid.
> It's quite common to have well known severities for alerts which can be important even in the
> Alertmanager routing tree. Ths is how you can make sure only the well-known severities are used.
```yaml
params:
  label: "foo"
  allowedValues: ["foo", "bar"]
  commaSeparatedValue: true
```

### `annotationHasAllowedValue`
Fails if rule annotation value is not one of the allowed values.
```yaml
params:
  annotation: "foo"
  allowedValues: ["foo", "bar"]
  commaSeparatedValue: true
```

### `annotationIsValidURL`
Fails if annotation value is not a valid URL.
If `resolveURL` is enabled, tries to make an HTTP request to the specified URL
and fails if the request does not succeed or returns 404 HTTP status code.
> It's common practise to link a playbook with guide how to solve the alert in the alert itself.
> This way you can verify it's a working URL and possibly if it really exists.
```yaml
params:
  annotation: "playbook"
  resolveUrl: true
```

### `expressionDoesNotUseLabels`
Fails if the rule uses any of specified labels in its `expr` label matchers, aggregations or joins.
> If using Thanos, users has to know if the rule is evaluated by Prometheus or Thanos, 
> but Prometheus cannot use the external labels. This way you can make sure it won't happen.
```yaml
params:
  labels: ["foo", "bar"]
```

### `expressionDoesNotUseOlderDataThan`
Fails if the rule `expr` uses older data than specified limit in Prometheus duration syntax.
Checks even in subqueries and offsets.
> Useful to avoid writing queries which expects longer data retention than the Prometheus actually has.
```yaml
params:
  limit: "12h"
```

### `expressionDoesNotUseRangeShorterThan`
Fails if the rule `expr` uses shorter range than specified limit in the Prometheus duration format.
>Useful to avoid using shorter range than twice of the scrape interval. 
```yaml
params:
  limit: "1m"
```

### `annotationIsValidPromQL`
Fails if the rule specified annotations does not contain valid PromQL if present. 
```yaml
params:
  annotation: "foo"
```
