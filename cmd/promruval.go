package main

import (
	"fmt"
	"github.com/fusakla/promruval/pkg/config"
	"github.com/fusakla/promruval/pkg/report"
	"github.com/fusakla/promruval/pkg/validate"
	"github.com/fusakla/promruval/pkg/validator"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"time"
)

var (
	// Set using goreleaser ldflags during build, see https://goreleaser.com/environment/#using-the-mainversion
	version = "dev"
	commit  = "none"
	date    = time.Now().Format("2006-01-02")
	builtBy = os.Getenv("USER")

	app = kingpin.New("promruval", "Prometheus rules validation tool.")

	versionCmd = app.Command("version", "Print version and build information.")

	validateCmd            = app.Command("validate", "Validate Prometheus rule files using validation rules from config file.")
	validateConfigFile     = validateCmd.Flag("config-file", "Path to validation config file.").Short('c').Required().ExistingFile()
	filePaths              = validateCmd.Arg("path", "File paths to be validated, can be passed as a glob.").Required().Strings()
	disabledRules          = validateCmd.Flag("disable-rule", "Allows to disable any validation rules by it's name. Can be passed multiple times.").Short('d').Strings()
	validationOutputFormat = validateCmd.Flag("output", "Format of the output.").Short('o').PlaceHolder("[text,json,yaml]").Default("text").Enum("text", "json", "yaml")
	color                  = validateCmd.Flag("color", "Use color output.").Bool()

	docsCmd          = app.Command("validation-docs", "Print human readable form of the validation rules from config file.")
	docsConfigFile   = docsCmd.Flag("config-file", "Path to validation config file.").Short('c').Required().ExistingFile()
	docsOutputFormat = docsCmd.Flag("output", "Format of the output.").Short('o').PlaceHolder("[text,markdown,html]").Default("text").Enum("text", "markdown", "html")
)

func loadConfigFile(configFilePath string) (*config.Config, error) {
	configFile, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}

	validationConfig := &config.Config{}
	decoder := yaml.NewDecoder(configFile)
	decoder.KnownFields(true)
	if err := decoder.Decode(validationConfig); err != nil {
		return nil, fmt.Errorf("loading config file: %w", err)
	}
	return validationConfig, nil
}

func validationRulesFromConfig(config *config.Config) ([]*validate.ValidationRule, error) {
	var validationRules []*validate.ValidationRule
rulesIteration:
	for _, rule := range config.ValidationRules {
		for _, disabledRule := range *disabledRules {
			if disabledRule == rule.Name {
				continue rulesIteration
			}
		}
		newRule := validate.NewValidationRule(rule.Name, rule.Scope)
		for _, validatorConfig := range rule.Validations {
			newValidator, err := validator.NewFromConfig(validatorConfig)
			if err != nil {
				return nil, fmt.Errorf("loading validator config: %w", err)
			}
			if newValidator == nil {
				continue
			}
			newRule.AddValidator(newValidator)
		}
		validationRules = append(validationRules, newRule)
	}
	return validationRules, nil
}

func exitWithError(err error) {
	fmt.Printf("Error: %v\n", err)
	os.Exit(1)
}

func main() {

	currentCommand := kingpin.MustParse(app.Parse(os.Args[1:]))

	switch currentCommand {
	case versionCmd.FullCommand():
		fmt.Printf("Version: %s\nBuild date: %s\nBuild commit: %s\nBuilt by: %s", version, date, commit, builtBy)
	case docsCmd.FullCommand():
		validationConfig, err := loadConfigFile(*docsConfigFile)
		if err != nil {
			exitWithError(err)
		}
		validationRules, err := validationRulesFromConfig(validationConfig)
		if err != nil {
			exitWithError(err)
		}
		var reportRules []report.ValidationRule
		for _, r := range validationRules {
			reportRules = append(reportRules, r)
		}
		output, err := report.ValidationDocs(reportRules, *docsOutputFormat)
		if err != nil {
			exitWithError(err)
		}
		fmt.Println(output)
	case validateCmd.FullCommand():
		var filesToBeValidated []string
		for _, path := range *filePaths {
			paths, err := filepath.Glob(path)
			if err != nil {
				exitWithError(err)
			}
			for _, p := range paths {
				filesToBeValidated = append(filesToBeValidated, p)
			}
		}

		validationConfig, err := loadConfigFile(*validateConfigFile)
		if err != nil {
			exitWithError(err)
		}
		validationRules, err := validationRulesFromConfig(validationConfig)
		if err != nil {
			exitWithError(err)
		}

		excludeAnnotation := "disabled_validation_rules"
		if validationConfig.CustomExcludeAnnotation != "" {
			excludeAnnotation = validationConfig.CustomExcludeAnnotation
		}
		validationReport := validate.Files(filesToBeValidated, validationRules, excludeAnnotation)
		switch *validationOutputFormat {
		case "text":
			fmt.Println(validationReport.AsText(2, *color))
		case "json":
			fmt.Println(validationReport.AsJSON())
		case "yaml":
			fmt.Println(validationReport.AsYaml())
		}
		if validationReport.Failed {
			os.Exit(1)
		}
	}
}
