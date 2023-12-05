
Validation rules:

  check-severity-label
    - Alert has labels: `severity`
    - Alert label `severity` has one of the allowed values: `info`,`warning`,`critical`
    - Alert if rule has label `severity` with value `info` , it cannot have label `page`
    - Alert expression can be successfully evaluated on the live Prometheus instance
    - Alert expression uses only labels that are actually present in Prometheus
    - Alert expression selectors actually matches any series in Prometheus
    - Alert expression does not use data older than `6h0m0s`

  check-team-label
    - Alert has labels: `xxx`
    - Alert label `team` has one of the allowed values: `sre@company.com`

  check-playbook-annotation
    - Alert has any of these annotations: `playbook`,`link`
    - Alert Annotation `link` is a valid URL and does not return HTTP status 404

  check-alert-title
    - Alert has all of these annotations: `title`

  check-prometheus-limitations
    - All rules expression does not use data older than `6h0m0s`
    - All rules does not use any of the `cluster`,`locality`,`prometheus-type`,`replica` labels is in its expression
    - All rules verifies if the rule group, the rule belongs to, has the required source_tenants configured, according to the mapping of metric names to tenants: `k8s`:`^container_.*|kube_.*$`

  another-checks
    - All rules labels does not have empty values
