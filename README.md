# Promruval
![GitHub actions CI](https://img.shields.io/github/workflow/status/fusakla/promruval/Go/master)
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
You can read a blog post about the motivation and usage [here](https://fusakla.medium.com/promruval-validating-prometheus-rules-9a29f5dc24d2) or [watch a lightning talk about it from PromCon](https://www.youtube.com/watch?v=YYSJ--KhlIo&list=PLj6h78yzYM2PZb0QuIkm6ZY-xTuNA5zRO&index=16).

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
go get github.com/fusakla/promruval
```
or 
```
make build
```


### Usage
```bash
$ promruval --help-long
usage: promruval [<flags>] <command> [<args> ...]

Prometheus rules validation tool.

Flags:
  --help  Show context-sensitive help (also try --help-long and --help-man).

Commands:
  help [<command>...]
    Show help.

  version
    Print version and build information.

  validate --config-file=CONFIG-FILE [<flags>] <path>...
    Validate Prometheus rule files using validation rules from config file.

    -c, --config-file=CONFIG-FILE  Path to validation config file.
    -d, --disable-rule=DISABLE-RULE ...  
                                   Allows to disable any validation rules by it's name. Can be passed multiple times.
    -o, --output=[text,json,yaml]  Format of the output.
        --color                    Use color output.

  validation-docs --config-file=CONFIG-FILE [<flags>]
    Print human readable form of the validation rules from config file.

    -c, --config-file=CONFIG-FILE  Path to validation config file.
    -o, --output=[text,markdown,html]  
                                   Format of the output.
```


### Configuration
Promruval uses a yaml configuration file to define the validation rules.
Basic structure is:
```yaml
# OPTIONAL Overrides the annotation used for disabling rules.
customExcludeAnnotation: my_disable_annotation

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
          labels: ["severity"]
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
