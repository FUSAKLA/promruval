
Validation rules:

  check-severity-label
    - Alert has labels: `severity`
    - Alert label `severity` has one of the allowed values: `info`,`warning`,`critical`
    - Alert if rule has label `severity` with value `info` , it cannot have label `page`
    - Alert expression does not use irate
    - Alert expression does not use data older than `6h0m0s`

  check-team-label
    - Alert has labels: `xxx`
    - Alert label `team` has one of the allowed values: `sre@company.com` (templated values are ignored)

  check-playbook-annotation
    - Alert has any of these annotations: `playbook`,`link`
    - Alert Annotation `link` is a valid URL and does not return HTTP status 404

  check-alert-title
    - Alert has all of these annotations: `title`

  check-prometheus-limitations
    - All rules expression does not use any experimental PromQL functions
    - All rules expression uses underscores as separators in large numbers in PromQL expression. Example: 1_000_000
    - All rules expression does not use data older than `6h0m0s`
    - All rules does not use any of the `cluster`,`locality`,`prometheus-type`,`replica` labels is in its expression

  check-metric-name
    - Alert expression uses metric name in selectors
    - Alert labels are valid templates
    - Alert `keep_firing_for` is not longer than `1h`

  check-groups
    - Group evaluation interval is between `20s` and `106751d23h47m16s854ms` if set
    - Group has at most 10 rules
    - Group does not have higher `limit` configured then 100

  check-formatting
    - All rules expression is well formatted as would `promtool promql format` do or similar online tool such as https://o11y.tools/promqlparser/
    - All rules expression does not do any binary operations between histogram buckets, it can be dangerous because of inconsistency in the data if sent over remote write for example

  check-recording-rules
    - Recording rule Recorded metric name does not match regexp: ^foo_bar$
    - Recording rule Recorded metric name matches regexp: [^:]&#43;:[^:]&#43;:[^:]&#43;

  check-labels-in-expr
    - All rules for metrics matching regexp &#39;^up$&#39;, given lables use only specified values: job: [prometheus]

    - All rules expression does not use labels `app` for metrics matching regexp ^up$ in the expr

  another-checks
    - All rules labels does not have empty values

