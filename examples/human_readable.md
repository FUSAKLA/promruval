
Validation rules:

  check-severity-label
    - Alert has labels: `severity`
    - Alert label `severity` has one of the allowed values: `info`,`warning`,`critical`
    - Alert if rule has label `severity` with value `info` , it cannot have label `page`
    - Alert expression can be successfully evaluated on the live Prometheus instance
    - Alert expression uses only labels that are actually present in Prometheus
    - Alert expression does not use irate
    - Alert expression selectors actually matches any series in Prometheus
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
    - All rules expression does not use data older than `6h0m0s`
    - All rules does not use any of the `cluster`,`locality`,`prometheus-type`,`replica` labels is in its expression

  check-source-tenants
    - All rules rule group, the rule belongs to, has the required `source_tenants` configured, according to the mapping of metric names to tenants: 
        `k8s`:   `^container_.*$` (Metrics from cAdvisor)
        `k8s`:   `^kube_.*$` (Metrics from KSM)
        `mysql`:   `^mysql_.*$` (MySQL metrics from the MySQL team)

  check-metric-name
    - Alert expression uses metric name in selectors
    - Alert labels are valid templates
    - Alert `keep_firing_for` is not longer than `1h`

  check-groups
    - Group does not have other `source_tenants` than: `tenant1`, `tenant2`, `k8s`
    - Group evaluation interval is between `20s` and `106751d23h47m16s854ms` if set
    - Group has valid partial_response_strategy (one of `warn` or `abort`) if set
    - Group has at most 10 rules
    - Group does not have higher `limit` configured then 100

  check-formatting
    - All rules expression is well formatted as would `promtool promql format` do or similar online tool such as https://o11y.tools/promqlparser/

  another-checks
    - All rules labels does not have empty values

