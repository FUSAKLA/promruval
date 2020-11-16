# Promruval
![CircleCI](https://img.shields.io/circleci/build/github/FUSAKLA/promruval/master)

_Prometheus Rule Validator_


Tool for validation of Prometheus rules metadata.

Promtool allows user to verify syntactic correctness and test PromQL expressions.
Promruval aims to validate the rules' metadata.

This is useful for making sure the labels, you use for routing alerts in Alertmanager,
has the allowed values you expect in the routing. 
Same for severities or avoiding typos.

![](./promruval.png)

### Supported validations

| Validation | Example usage |
|------------|---------------|
| Annotation is valid URL (and is resolvable) | Make sure the linked playbooks really exist. |
| Expr does not use older data than specified limit | Avoid querying more data than the retention is.|
| Expr does not use specified labels | When using Thanos, can be useful to forbid usage of external labels when alerting on Prometheus to avoid confusion. |
| Rule has specified labels/annotations | Make sure the alert has a `team` label for routing. |
| Label/Annotation matches specified regexp | Check if `team` label is valid email |
| Label/Annotation has one of the allowed values | Check only allowed severities `info`, `warning` and `critical` are used. |
| Rule has any of specified labels/annotations | If you have 2 annotations relative and absolute to reference playbook. |
 
 
### Install
Download prebuilt binaries, Docker image of build it yourself.
 ```bash
go get github.com/fusakla/proruval 
```

### Configuration
Promruval uses yaml configuration file to define the validation rules.
See the [`examples/validation.yaml`](examples/validation.yaml) for example.

### How to use it
If you downloaded the prebuilt binary or built it on your own:
```bash
promruval validate --config-file=examples/validation.yaml examples/rules.yaml
```

Or using Docker
```bash
docker run -it -v $PWD:/rules fusakla/promruval --config-file=/rules/examples/validation.yaml /rules/examples/rules.yaml
```

### Disabling rules
If you want to temporarily disable any of the rules, you can use the `--dicable-rule` flag
with value corresponding to the `name` of the rule you want to disable. You can pass it multiple times.
```bash
promruval validate --config-file examples/validation.yaml --disable-rule check-team-label examples/rules.yaml
```
 
### Readable validation docs
If you want more readable validation summary (for a documentation for example or generating readable pages)
you can use the `validation-docs` command. It should print out more human readable form than the configuration file is.
```bash
promruval validation-docs --config-file examples/validation.yaml
```
