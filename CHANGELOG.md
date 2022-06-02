# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
