# Promruval
![CircleCI build status](https://img.shields.io/circleci/build/github/FUSAKLA/promruval/master)
[![Docker Pulls](https://img.shields.io/docker/pulls/fusakla/promruval)](https://hub.docker.com/r/fusakla/promruval)
[![GitHub binaries download](https://img.shields.io/github/downloads/fusakla/promruval/total?label=Prebuilt%20binaries%20downloads)](https://github.com/FUSAKLA/promruval/releases/latest)

![](./promruval.png)

_Prometheus Rule Validator_


Tool for validation of Prometheus rules metadata.

Promtool allows user to verify syntactic correctness and test PromQL expressions.
Promruval aims to validate the rules' metadata.

This is useful for making sure the labels, you use for routing alerts in Alertmanager,
has the allowed values you expect in the routing.
Same for severities or avoiding typos.

### Examples of usage
 - Make sure the playbook linked by an alert is valid URL and really exist.
 - Avoid querying more data than the retention of used Prometheus by checking expr
   does not use older data than specified. 
 - Prevent expr to use any of specified labels. Useful if using Thanos to forbid
   usage of external labels when alerting on Prometheus to avoid confusion for users.
 - Ensure alerts has required labels for routing in Alertmanager possibly with allowed values.
 - Make sure Alerts has the expected annotations for rendering the alert template.
 - Forbid usage of some labels or annotations if it got deprecated. 
 - And many more...
 
Validations are quite variable, so you can use them as you fit.
In case of any missing, please create a feature request,
and I'd be happy to add it if reasonable.
 
### Install
Using [prebuilt binaries](https://github.com/FUSAKLA/promruval/releases/latest), [Docker image](https://hub.docker.com/r/fusakla/promruval) of build it yourself.
 ```bash
go get github.com/fusakla/proruval 
```
or 
```
make build
```

### Configuration
Promruval uses yaml configuration file to define the validation rules.
See the [`examples/validation.yaml`](examples/validation.yaml) for example.

### How to use it
If you downloaded the [prebuilt binary](https://github.com/FUSAKLA/promruval/releases/latest) or built it on your own:
```bash
promruval validate --config-file=examples/validation.yaml examples/rules.yaml
```

Or using [Docker image](https://hub.docker.com/r/fusakla/promruval)
```bash
docker run -it -v $PWD:/rules fusakla/promruval validate --config-file=/rules/examples/validation.yaml /rules/examples/rules.yaml
```

### Disabling rules
If you want to temporarily disable any of the rules, you can use the `--disable-rule` flag
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
