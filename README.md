# Promruval

[![Go Report Card](https://goreportcard.com/badge/github.com/fusakla/promruval)](https://goreportcard.com/report/github.com/fusakla/promruval)
[![GitHub actions CI](https://img.shields.io/github/workflow/status/fusakla/promruval/Go/master)](https://github.com/FUSAKLA/promruval/actions?query=branch%3Amaster)
[![Docker Pulls](https://img.shields.io/docker/pulls/fusakla/promruval)](https://hub.docker.com/r/fusakla/promruval)
[![GitHub binaries download](https://img.shields.io/github/downloads/fusakla/promruval/total?label=Prebuilt%20binaries%20downloads)](https://github.com/FUSAKLA/promruval/releases/latest)

![](./promruval.png)

_Prometheus Rule Validator_

Promtool allows users to verify syntactic correctness and test PromQL expressions.
Promruval aims to validate the rules' metadata and expression properties
to match requirements and constraints of the particular Prometheus cluster setup.
User defines his validation rules in simple yaml configuration and passes them to
the promruval which validates specified files with Prometheus rules same way promtool does.
Usually it would be used in the CI pipeline.
You can read a blog post about the motivation and
usage [here](https://fusakla.medium.com/promruval-validating-prometheus-rules-9a29f5dc24d2)
or [watch a lightning talk about it from PromCon](https://www.youtube.com/watch?v=YYSJ--KhlIo&list=PLj6h78yzYM2PZb0QuIkm6ZY-xTuNA5zRO&index=16)
.

### Examples of usage

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

Validations are quite variable, so you can use them as you fit.
Full list of supported validations can be found [here](docs/validations.md).

In case of any missing, please create a feature request!

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
    # What Prometheus rules to validate, possible values are: 'Alert', 'Recording rule', 'All rules'.
    scope: All rules
    # List of validations to be used.
    validations:
      # Name of the validation type. See the /docs/validations.md.
      - type: hasLabels
        # Parameters of the validation.
        params:
          labels: [ "severity" ]
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

#### Validation using live Prometheus instance

Event though these validations are useful, they may be flaky and dangerous for the Prometheus instance.
If you have large number of rules and run the check often the number of queries can be huge or the instance might go
down and your validation
would be flaky.

Therefore, it's recommended to use these check as a warning and do not fail if it does not succeed.

### Disabling validations per rule

If you want to disable particular validation for a certain rule, you can add a comment above it with a list of
validation names to ignore. Alternatively, the comment can be put on its own line _inside_ the `expr` of the rule.
The in-expression comment can be present multiple times.

By default, the comment prefix is `ignore_validations` but can be changed using the `customDisableComment` config option
in [config](#configuration).
Value of the comment should be comma separated list of [validation names](./docs/validations.md)

Example:

```yaml
groups:
  - name: foo
    rules:
      # The following validations will be ignored in the rule that immediately follows.
      # ignore_validations: expressionSelectorsMatchesAnything, expressionDoesNotUseOlderDataThan
      - record: bar
        expr: 1
      # The same validations are disabled for the following rule, but the comments are in the expression.
      - name: baz
        expr: |
          # ignore_validations: expressionSelectorsMatchesAnything
          up{
            # ignore_validations: expressionDoesNotUseOlderDataThan
          }
```

### Disabling rules

If you want to temporarily disable any of the rules for all the tested rules,
you can use the `--disable-rule` flag with value corresponding to the `name`
of the rule you want to disable. Can be passed multiple times.

```bash
promruval validate --config-file examples/validation.yaml --disable-rule check-team-label examples/rules.yaml
```

If you want to disable permanently for some Prometheus rule, you can use the special annotation
`disabled_validation_rules`(can be changed in the [config](#configuration)) that represents comma separated list of
rule names to be skipped for the particular rule.

Example Prometheus rule:

```yaml
groups:
  - name: ...
    rules:
      - alert: ...
        expr: ...
        annotations:
          disabled_validation_rules: team-label-check,title-annotation-check
```

### Readable validation description

If you want more human readable validation summary (for a documentation or generating readable git pages)
you can use the `validation-docs` command, see the [usage](#usage).
It should print out more human readable form than the configuration file is
and supports multiple output formats.

```bash
promruval validation-docs --config-file examples/validation.yaml --output=html
```
