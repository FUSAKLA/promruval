# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
 - New parameter `commaSeparatedValue` for the `annotationHasAllowedValue` validator supporting annotations with a comma separated values.

## [v1.2.0] - 2020-11-29
### Added
 - New parameter `commaSeparatedValue` for the `labelHasAllowedValue` validator supporting labels with a comma separated values.
 - Added new validation check [`annotationIsValidPromQL`](docs/validations.md#annotationisvalidpromql)
   to verify if rule annotation contains valid PromQL expression.
### Fixed
 - Switch back to official PromQL package to parse queries.

## [v1.1.0] - 2020-11-20
### Changed
 - Switched to the [Prometheus Duration format](https://prometheus.io/docs/prometheus/latest/querying/basics/#time-durations) allowing usage ot `d`, `w` and `y`.

## [v1.0.0] - 2020-11-20
### Changed
 - **Breaking:** The `scope` configuration values has changed:
     - `AllRules` -> `All rules`
     - `RecordingRules` -> `Recording rules`
     
### Added
 - Added support for special rule annotation containing names of validation rules
   that should be skipped for the rule. Default annotation name is `disabled_validation_rules`.
 - Added new command [`validation-docs`](README.md#readable-validation-description) to print out human readable description of the validation config.
 - Added docs for all supported validations in [docs/validations.md](docs/validations.md).
 - Added new `version` command that prints out version and build metadata.
 - Added new validator [`expressionDoesNotUseRangeShorterThan`](docs/validations.md#expressiondoesnotuserangeshorterthan).
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
