# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

- Fixed: :warning: **Unmarshalling of the rule files is strict again**, this behavior was unintentionally brought when adding support for yaml comments.
- Changed: :warning: **Renamed `hasValidPartialStrategy` to `hasValidPartialResponseStrategy`** as it was documented so it is actually a fix
- Changed: :warning: **Disallow special rule file fields of Thanos, Mimir or Loki by default**
           To enable them, you need to set some of the new flags described below
- Changed: The Prometheus results cache format has changed to reduce it's size and improve performance. **Delete the old cache file** before upgrade.
           Also now if the cache contains time of creation and URL of the Prometheus it has data for. From now on, if the URL does not match, the case is pruned.
- Added: :rocket: Support for validation of rule files in the [Jsonnet](https://jsonnet.org/) format.
- Added: New flags `--support-thanos`, `--support-mimir`, `--support-loki` to enable special rule file fields of Thanos, Mimir or Loki
- Added: :tada: **Support for validation of Loki rules!** Now you can validate Loki rules as well. First two validators are:
   - `expressionIsValidLogQL` to check if the expression is a valid LogQL query
   - `logQlExpressionUsesRangeAggregation` to check if the LogQL expression uses range aggregation
   - `logQlExpressionUsesFiltersFirst` to check if the LogQL expression uses filters first in the query since it is more efficient
- Added: Support for alert field `keep_firing_for`
- Added: Support for the `query_offset` field in the rule group
- Added: New validator `expressionIsValidPromQL` to check if the expression is a valid PromQL query
- Added: Support for bearer token authentication in the `prometheus` section of the config using the `bearerTokenFile` field or by specifying the `PROMETHEUS_BEARER_TOKEN` env variable.
- Added: `maximumMatchingSeries` option to the `expressionSelectorsMatchesAnything` validator to check maximum number of series any selector can match.
- Added: new config options to the prometheus section of config:
  - `queryOffset`: Specify offset(delay) of the query (useful for consistency if using remote write for example).
  - `queryLookback`: How long into the past to look in queries supporting time range (just metadata queries for now).
- Added: New validator `alertNameMatchesRegexp` to check if the alert name matches the regexp.
- Added: New validator `groupNameMatchesRegexp` to check if the rule group name matches the regexp.
- Added: New validator `recordedMetricNameMatchesRegexp` to check if the recorded metric name matches the regexp.
- Fixed: Loading glob patterns in the file paths to rules
- Fixed: Params of the `expressionCanBeEvaluated` validator were ignored, this is now fixed.
- Updated: Prometheus and other dependencies
- CI: Updated Github actions for golangcilint and goreleaser

## [2.14.1]
- Fixed: error message in the `hasSourceTenantsForMetrics` validator
- Updated: indirect Go dependency google.golang.org/protobuf bumped to v1.33.0

## [2.14.0]
- Added: new param `defaultTenant` to the `hasSourceTenantsForMetrics` validator to set the default tenant for the rule group.

## [2.13.0] - 2024-03-08
- Added: support for disabling validation using comments in yaml rules per rule group and for the whole file.
- Docs: Updated/improved documentation on how to disable validations and validation rules.

## [2.12.0] - 2024-03-07
- Fixed resolving of the path in `paramsFromFile`. Formerly it was resolved from the current working directory, now it must be a relative path, that will be resolved from the config file location.
- Fixed: tenants in the `hasSourceTenantsForMetrics` validator are now sorted in the human readable output.

## [2.11.0] - 2024-03-07
- :warning: CHANGED: Params of the `hasSourceTenantsForMetrics` validator (again FACEPALM). Now the tenant can have multiple regexp matchers.
  See its [docs](docs/validations.md#hassourcetenantsformetrics).

## [2.10.0] - 2024-03-05
- Fixed error messages for the `hasSourceTenantsForMetrics` and `expressionDoesNotUseIrate` validators.

- Added new option `paramsFromFile` to the validators, so the params can be loaded from a YAML file.
  Example promruval config:
  ```yaml
  - type: labelHasAllowedValue
    paramsFromFile: ./examples/allowed_values_params.yaml
  ```

  Content of `./examples/allowed_values_params.yaml`:
  ```yaml
  label: "severity"
  allowedValues: ["info", "warning", "critical"]
  ```

- Added new config option `additionalDetails` to all validators providing  possibility to add custom details about the error and how to solve it.
  Those will be appended to the validator error message in a parenthesis if provided.
  Example configuration:
  ```yaml
  - name: expressionDoesNotUseIrate
    additionalDetails: "Just do as I say!"
  ```
  Example output:
  ```yaml
  expressionDoesNotUseIrate: you should not use the `irate` function in rules, for more info see https://www.robustperception.io/avoid-irate-in-alerts/ (Just do as I say!)
  ```

- :warning: CHANGED: Params of the `hasSourceTenantsForMetrics` validator. See its [docs](docs/validations.md#hassourcetenantsformetrics).

## [v2.9.0] - 2024-03-02
- Added new `All rules` validator `expressionIsWellFormatted` to check if rules are well formatted as `promtool promql format` would do.
- Added new `Group` validator `maxRulesPerGroup` to check if the number of rules in the group is not exceeding the limit.
- Added new `Group` validator `hasAllowedLimit` to check if the rule group has the `limit` lower than the limit and possibility to enforce it to be configured.
- Added new `Alert` validator `validateLabelTemplates` to check if the alert labels are valid Go templates.
- Added new `Alert` validator `keepFiringForIsNotLongerThan` to check if the alert does not have `keep_firing_for` longer then limit.
- Updated all deps to the latest versions.
- Upgraded go to 1.22
- Update all Github actions in CI
- Made lint more strict and fixed all the issues

### :warning: CHANGED validator scopes
  From now on each validator has restricted scopes it can be used with since they may not make sense in some contexts.
  The documentation in [docs/validations.md](docs/validations.md) was updated and is structured based on the scopes.
  > If you were using the `All rules` scope, you may need to update your configuration and split the rules by the scopes.

## [v2.8.1] - 2024-02-29
- Fixed param validation of the `hasAllowedEvaluationInterval` validator, if the `maximum` was not set.

## [v2.8.0] - 2024-02-29
- Added new param `ignoreTemplatedValues` to the `labelHasAllowedValue` validator to ignore templated values in the label.
- Added new validation rule scope `Group` to validate the rule group itself (not the rules in it).
- Added new `Group` scope validator `hasAllowedEvaluationInterval` to check if the rule group has the `interval` in the configured range and possibility to enforce it to be configured.
- Added new `Group` scope validator `hasValidPartialResponseStrategy` to check if the rule group has valid `partial_response_strategy` and possibility to enforce it to be configured.
- CHANGED: The validator `allowedSourceTenants` is now allowed only in the `Group` scope validation rules.
- Fixed marking empty rule files (or those with all the content commented out) with an error saying EOF, from now on such files are ignored.

## [v2.7.1] - 2024-02-01
- Upgrade all dependencies
- Fix: `promruval version` now works without specifying `--config-file`

## [v2.7.0] - 2023-12-06
- Added new validator `expressionDoesNotUseMetrics`, see its [docs](docs/validations.md#expressiondoesnotusemetrics).
- Added new validator `hasSourceTenantsForMetrics`, see its [docs](docs/validations.md#hassourcetenantsformetrics).
- Improved the HTML output of human readable validation description.
- Added examples of the human-readable validation descriptions to the examples dir.
- Refactored the validation so it can use also group to validate the context of the rule.
- Improve linting and fix all the linting issues.
- Added new validator `hasValidSourceTenants`, see its [docs](docs/validations.md#hasvalidsourcetenants).
- Improved wording in the human readable validation output.

## [v2.6.0] - 2023-12-06
- Added new validator `expressionWithNoMetricName`, see its [docs](docs/validations.md#expressionwithnometricname). Thanks @tizki !
- Upgrade to go 1.21
- Upgrade all dependencies

## [v2.5.0] - 2023-04-29

- Upgrade all dependencies
- Upgrade to Go 1.19
- Support `keep_firing_for` in alert rule
- Support `source_tenants` in rule group used by Cortex/Mimir
- Add linting to CI

## [v2.4.1] - 2023-01-10

- Fixed installation instructions in README.
- Upgraded prometheus dependency and to avoid installation issues using `go install`.

## [v2.4.0] - 2023-01-10

- [#30](https://github.com/FUSAKLA/promruval/pull/30)
  - Upgrade Go to 1.19.
  - :warning: CHANGE - go.mod version bumped to match the project major version, if you use promruval as a library,
                       make sure to change the package to `github.com/fusakla/promruval/v2`.
  - :warning: CHANGE - Updated README.md installation instructions, to install latest version use `go install github.com/fusakla/promruval/v2`.


## [v2.3.1] - 2022-06-29

- [#27](https://github.com/FUSAKLA/promruval/pull/27)
  - typos and wording in validator messages were corrected

## [v2.3.0] - 2022-06-07

- [#25](https://github.com/FUSAKLA/promruval/pull/25)
  - Delete forgotten debug print :ashamed:
  - Redirect logging to stderr
  - Log progress
  - Fix e2e test

- [#26](https://github.com/FUSAKLA/promruval/pull/26)
  - Allow disabling validations in comments inside the `expr` of rules.
    This is useful when you generate the prometheus rule files from a system
    that doesn't support YAML comments, e.g. jsonnet.

## [v2.2.0] - 2022-06-07

- [#24](https://github.com/FUSAKLA/promruval/pull/24) Support disabling validators per rule using comments in yaml,
  see [the docs](./README.md#disabling-validations-per-rule)

## [v2.1.1] - 2022-06-06

- [#23](https://github.com/FUSAKLA/promruval/pull/23) Fix unmarshall of `expressionDoesNotUseOlderDataThan` params.

## [v2.1.0] - 2022-06-06

- [#22](https://github.com/FUSAKLA/promruval/pull/22) Upgrade Prometheus dependencies to support newest PromQL features

## [v2.0.1] - 2022-06-06

- [#21](https://github.com/FUSAKLA/promruval/pull/21) Fix `validFunctionsOnCounters` and `rateBeforeAggregation`
  validators

## [v2.0.0] - 2022-06-03

No actual breaking changes, but a lot of new features and configuration options so why not a major release :)

### Changed

- [#16](https://github.com/FUSAKLA/promruval/pull/16) Upgraded yaml.v3 library to mitigate CVE-2022-28948
- [#15](https://github.com/FUSAKLA/promruval/pull/15) Upgraded to Go 1.18
- [#9](https://github.com/FUSAKLA/promruval/pull/9) Upgraded to Go 1.17

### Added

- [#10](https://github.com/FUSAKLA/promruval/pull/10) New validator `validateAnnotationTemplates` for more info
  see [the docs](docs/validations.md#validateannotationtemplates)
- [#11](https://github.com/FUSAKLA/promruval/pull/11) New validator `forIsNotLongerThan` for more info
  see [the docs](docs/validations.md#forisnotlongerthan)
- [#12](https://github.com/FUSAKLA/promruval/pull/12) New validator `expressionDoesNotUseIrate` for more info
  see [the docs](docs/validations.md#expressiondoesnotuseirate)
- [#13](https://github.com/FUSAKLA/promruval/pull/13) New validator `validFunctionsOnCounters` for more info
  see [the docs](docs/validations.md#validfunctionsoncounters)
- [#14](https://github.com/FUSAKLA/promruval/pull/14) New validator `rateBeforeAggregation` for more info
  see [the docs](docs/validations.md#ratebeforeaggregation)
- [#17](https://github.com/FUSAKLA/promruval/pull/17) New
  validators: [`nonEmptyLabels`](docs/validations.md#nonemptylabels)
  , [`exclusiveLabels`](docs/validations.md#exclusivelabels)
- [#18](https://github.com/FUSAKLA/promruval/pull/18) Added e2e tests
- [#19](https://github.com/FUSAKLA/promruval/pull/19)
    - Added support for validations using live Prometheus and added checks:
        - [expressionCanBeEvaluated](/docs/validations.md#expressioncanbeevaluated)
        - [expressionUsesExistingLabels](/docs/validations.md#expressionusesexistinglabels)
        - [expressionSelectorsMatchesAnything](/docs/validations.md#expressionselectorsmatchesanything)
    - Added `--debug` flag and more logging.
    - New `prometheus` section to the [root configuration](README.md#configuration) allowing to use validation against
      live prometheus instance.
    - Added caching of Prometheus data is used. Default cache file is `./.promruval_cache.json`.
    - Added new flag `--enabled-rule` to enable only named validation rules.
- [#20](https://github.com/FUSAKLA/promruval/pull/20)
    - Flag `--config-file` can be now [passed multiple times](./README.md#configuration-composition) allowing config
      composition.
    - Accept `partial_response_strategy` field in rule group to be able to validate thanos rules.

### Fixed

- [#12](https://github.com/FUSAKLA/promruval/pull/12) Fixed the `annotationIsValidURL` to be more strict in parsing URL
  and to actually use the `resolve_url` configuration.

## [v1.3.2] - 2020-12-09

### Fixed

- [#8](https://github.com/FUSAKLA/promruval/pull/8) Fixed severe bugs in loading some of validator configurations.

## [v1.3.1] - 2020-12-08

### Fixed

- [#7](https://github.com/FUSAKLA/promruval/pull/7) Fixed typos in label and annotations checks `hasAnyOfAnnotations`
  and `hasAnyOfLabels`.

## [v1.3.0] - 2020-12-08

### Added

- New parameter `commaSeparatedValue` for the `annotationHasAllowedValue` validator supporting annotations with a comma
  separated values.

## [v1.2.0] - 2020-11-29

### Added

- New parameter `commaSeparatedValue` for the `labelHasAllowedValue` validator supporting labels with a comma separated
  values.
- Added new validation check [`annotationIsValidPromQL`](docs/validations.md#annotationisvalidpromql)
  to verify if rule annotation contains valid PromQL expression.

### Fixed

- Switch back to official PromQL package to parse queries.

## [v1.1.0] - 2020-11-20

### Changed

- Switched to
  the [Prometheus Duration format](https://prometheus.io/docs/prometheus/latest/querying/basics/#time-durations)
  allowing usage ot `d`, `w` and `y`.

## [v1.0.0] - 2020-11-20

### Changed

- **Breaking:** The `scope` configuration values has changed:
    - `AllRules` -> `All rules`
    - `RecordingRules` -> `Recording rules`

### Added

- Added support for special rule annotation containing names of validation rules
  that should be skipped for the rule. Default annotation name is `disabled_validation_rules`.
- Added new command [`validation-docs`](README.md#readable-validation-description) to print out human readable
  description of the validation config.
- Added docs for all supported validations in [docs/validations.md](docs/validations.md).
- Added new `version` command that prints out version and build metadata.
- Added new validator [`expressionDoesNotUseRangeShorterThan`](docs/validations.md#expressiondoesnotuserangeshorterthan)
  .
- Support searching in `expr` in label presence validators.
- Added short flags.

### Fixed

- Fixed issue when unmarshalling errors of rule files were not printed.

## [v0.1.3] - 2020-11-16

### Fixed

- Fixed docker build using goreleaser in CI.

## [v0.1.2] - 2020-11-16

### Fixed

- Fixed docker release in CI.

## [v0.1.1] - 2020-11-16

First public release.

## [v0.1.0] - 2020-11-15

Initial release.
