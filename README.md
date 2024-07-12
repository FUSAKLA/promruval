# $$\color{red}Prom\color{black}etheus \ \color{red}ru\color{black}le \ \color{red}val\color{black}idator$$

[![Go Report Card](https://goreportcard.com/badge/github.com/fusakla/promruval)](https://goreportcard.com/report/github.com/fusakla/promruval)
[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/fusakla/promruval/go.yaml)](https://github.com/FUSAKLA/promruval/actions?query=branch%3Amaster)
[![Docker Pulls](https://img.shields.io/docker/pulls/fusakla/promruval)](https://hub.docker.com/r/fusakla/promruval)
[![GitHub binaries download](https://img.shields.io/github/downloads/fusakla/promruval/total?label=Prebuilt%20binaries%20downloads)](https://github.com/FUSAKLA/promruval/releases/latest)

Promtool allows users to verify syntactic correctness and test PromQL expressions.
Promruval aims to validate the rules' metadata and expression properties
to match requirements and constraints of the particular Prometheus cluster setup.
User defines his validation rules in a simple yaml configuration and passes them to
the promruval which validates specified files with Prometheus rules same way promtool does.
Usually it would be used in the CI pipeline.
You can read a blog post about the motivation and
usage [here](https://fusakla.medium.com/promruval-validating-prometheus-rules-9a29f5dc24d2)
or [watch a lightning talk about it from PromCon](https://www.youtube.com/watch?v=YYSJ--KhlIo&list=PLj6h78yzYM2PZb0QuIkm6ZY-xTuNA5zRO&index=16)
.

### Example use-cases

- Make sure the playbook, linked by an alert, is a valid URL and really exists.
- Ensure the range selectors in the `expr` are not lower than three
  times your Prometheus scrape interval.
- Avoid querying more data than is retention of used Prometheus by inspecting
  if the `expr` does not use older data than specified.
- Make sure `expr` does not use any of the specified labels. Useful when using Thanos, to forbid
  usage of external labels when alerting on Prometheus to avoid confusion.
- Ensure alerts has the required labels expected by routing in Alertmanager
  possibly with allowed values.
- Make sure Alerts has the expected annotations for rendering the alert template.
- Forbid usage of some labels or annotations if it got deprecated.
- and many more...

> As a good starting point you can use the [`docs/default_validation.yaml`](docs/default_validation.yaml) which contains
some basic validations that are useful for most of the users.

Validations are quite variable, so you can use them as you fit.

### **👉 Full list of supported validations can be found [here](docs/validations.md).**

In case you would like to add some, please create a feature request!

### Installation

Using [prebuilt binaries](https://github.com/FUSAKLA/promruval/releases/latest),
[Docker image](https://hub.docker.com/r/fusakla/promruval) of build it yourself.

 ```bash
go install github.com/fusakla/promruval/v2@latest
```

or

```
make build
```

### Usage

```bash
$ ./promruval --help-long
usage: promruval --config-file=CONFIG-FILE [<flags>] <command> [<args> ...]

Prometheus rules validation tool.

Flags:
      --help   Show context-sensitive help (also try --help-long and --help-man).
  -c, --config-file=CONFIG-FILE ...
               Path to validation config file. Can be passed multiple times, only validationRules will be reflected from the additional configs.
      --debug  Enable debug logging.

Commands:
  help [<command>...]
    Show help.


  version
    Print version and build information.


  validate [<flags>] <path>...
    Validate Prometheus rule files using validation rules from config file.

    -d, --disable-rule=DISABLE-RULE ...
                                   Allows to disable any validation rules by it's name. Can be passed multiple times.
    -e, --enable-rule=ENABLE-RULE ...
                                   Only enable these validation rules. Can be passed multiple times.
    -o, --output=[text,json,yaml]  Format of the output.
        --color                    Use color output.

  validation-docs [<flags>]
    Print human readable form of the validation rules from config file.

    -o, --output=[text,markdown,html]
      Format of the output.
```

#### Configuration composition

The `--config-file` flag can be passed multiple times. Promruval will append the additional validation rules from the
other configs and override the other configurations. The late wins.
This allows you to use compose configuration for example if you have specific validations for rules.

**Example**:

```bash
rules/
  validations.yaml # Generic validations that apply to all rules
  prometheus/
     validations.yaml # Specific validations for Prometheus rules (different Prometheus URL, shorter data retention, no external labels etc)
     rules.yaml
  thanos/
     validations.yaml # Specific validations for Thanos (different URL, longer retention etc)
     rules.yaml
```

And Promruval would be run as

```bash
promruval validate --config-file ./rules/validation.yaml --config-file ./rules/prometheus/validation.yaml ./rules/prometheus/*.yaml
```

### Configuration

Promruval uses a yaml configuration file to define the validation rules.
Basic structure is:

```yaml
# OPTIONAL Overrides the annotation used for disabling rules.
customExcludeAnnotation: my_disable_annotation

prometheus:
  # URL of the running prometheus instance to be used
  url: https://foo.bar/
  # OPTIONAL Skip TLS verification
  insecureSkipTlsVerify: false
  # OPTIONAL Timeout for any request on the Prometheus instance
  timeout: 30s
  # OPTIONAL: name of the file to save cache of the Prometheus calls for speedup
  cacheFile: .promruval_cache.json
  # OPTIONAL: maximum age how old the cache can be to be used
  maxCacheAge: 1h

validationRules:
  # Name of the validation rule.
  - name: example-validation
    # What Prometheus rules to validate, possible values are: 'Group', 'Alert', 'Recording rule', 'All rules'.
    scope: All rules
    # List of validations to be used.
    validations:
      # Name of the validation type. See the /docs/validations.md.
      - type: hasLabels
        # Additional detaild that will be appended to the default error message. Useful to customize the error message.
        additionalDetails: "We do this because ..."
        # Parameters of the validation. See the /docs/validations.md for details on params of each validation.
        params:
          labels: [ "severity" ]
        # OPTIONAL If you want to load the parameters from a separate file, you can use this option.
        # Its value must be a relative path to the file from the location of the config file.
        # The content of the file must be in the exact form as the expected params would be.
        # The option is mutually exclusive with the `params` option.
        # paramsFromFile: /path/to/file.yaml
      ...
```

For a complete list of supported validations see the [docs/validations.md](docs/validations.md).

If you want to see example configuration see the  [`examples/validation.yaml`](examples/validation.yaml).

### How to run it

If you downloaded the [prebuilt binary](https://github.com/FUSAKLA/promruval/releases/latest) or built it on your own:

```bash
promruval validate --config-file=examples/validation.yaml examples/rules.yaml
```

Or using [Docker image](https://hub.docker.com/r/fusakla/promruval)

```bash
docker run -it -v $PWD:/rules fusakla/promruval validate --config-file=/rules/examples/validation.yaml /rules/examples/rules.yaml
```

### Validation using live Prometheus instance

Event though these validations are useful, they may be flaky and dangerous for the Prometheus instance.
If you have large number of rules and run the check often the number of queries can be huge or the instance might go
down and your validation
would be flaky.

Therefore, it's recommended to use these check as a warning and do not fail if it does not succeed.
Also consider running it rather periodically (for example once per day) instead of running it on every commit in CI.

### Disabling validations
There are three ways you can disable certain validation:
 - [Using cmd line flag](#using-cmd-line-flag)
 - [Using YAML comments](#using-yaml-comments)
 - [Using PromQL expression comments](#using-promql-expression-comments)
 - [Using alert annotation](#using-alert-annotation)

> The last two are useful if you yse for example jsonnet to generate the rules.
> Then you can't use the YAML comments, but you can set the comments in the expression or alert annotations.
> Unfortunately those have limited scope of usage (recording rules cannot have annotations, cannot be disabled on the group or file level).

#### Using cmd line flag
If you want to temporarily disable any of the validation rules for all the tested files,
you can use the `--disable-rule` flag with value corresponding to the `name`
of the validation rule you want to disable. Can be passed multiple times.

Example:
```yaml
# Promruval validation configuration
validationRules:
  - name: check-irate
    scope: Alert
    validations:
      - type: expressionDoesNotUseIrate
```

```bash
promruval validate --config-file examples/validation.yaml --disable-rule check-irate examples/rules.yaml
```

#### Using YAML comments
You can use comments in YAML to disable certain validations. This can be done on the file, group or rule level.
The comment should be in format `# ignore_validations: validationName1, validationName2, ...` where the `validationName`
is the name of the validation as defined in the [docs/validations.md](./docs/validations.md).

> The `ignore_validations` prefix can be changed using the `customDisableComment` config option in the [config](#configuration).

Example:
```yaml
# Disable for the whole file
# ignore_validations: expressionDoesNotUseIrate
groups:
  # Disable only for the following rule group
  # ignore_validations: expressionDoesNotUseIrate
  - name: group1
    partial_response_strategy: abort
    interval: 1m
    limit: 10
    rules:
      # Disable only for the following rule
      # ignore_validations: expressionDoesNotUseIrate
      - record: recorded_metrics
        expr: 1
        labels:
          foo: bar
```

#### Using PromQL expression comments
Same way as in the YAML comments, you can use comments in the PromQL expression to disable certain validations.
The comment should be in the same format `# ignore_validations: validationName1, validationName2, ...` where the `validationName`
is the name of the validation as defined in the [docs/validations.md](./docs/validations.md).
The comment can be present multiple times in the expression and can be anywhere in the expression.

> The `ignore_validations` prefix can be changed using the `customDisableComment` config option in the [config](#configuration).

Example:
```yaml
groups:
  - name: test-group
    rules:
      - alert: test-alert
        expr: |
          # ignore_validations: expressionDoesNotUseIrate
          irate(http_requests_total[5m]) # ignore_validations: expressionDoesNotUseIrate
```

#### Using alert annotation
If you can't(or don't want to) use the comments to disable validations, you can use the special annotation
`disabled_validation_rules`. It represents comma separated list of **validation rule names** to be skipped for the particular alert.
Since annotations are only available for alerts, **this method can be used only for alerts!**

> The `disabled_validation_rules` annotation name can be changed using the `customExcludeAnnotation` config option in the [config](#configuration).

Example:

```yaml
# Promruval validation configuration
validationRules:
  - name: check-irate
    scope: Alert
    validations:
      - type: expressionDoesNotUseIrate
```

```yaml
# Prometheus rule file
groups:
  - name: test-group
    rules:
      - alert: test-alert
        expr: 1
        annotations:
          disabled_validation_rules: check-irate # Will disable the check-irate validation rule check for this alert
```

### Human readable validation description

If you want more human readable validation summary (for a documentation or generating readable git pages)
you can use the `validation-docs` command, see the [usage](#usage).
It should print out more human readable form than the configuration file is
and supports multiple output formats such as `text`, `markdown` and `HTML`.
See the examples for the output for [Markdown](./examples/human_readable.md) and [HTML](./examples/human_readable.html).

```bash
promruval validation-docs --config-file examples/validation.yaml --output=html
```
