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
)

var (
	app           = kingpin.New("promruval", "Prometheus rules validation tool.")
	configFile    = app.Flag("config-file", "Path to validation config file.").Required().ExistingFile()
	disabledRules = app.Flag("disable-rule", "Allows to disable any validation rules by it's name. Can be passed multiple times.").Strings()

	validateCmd            = app.Command("validate", "Validate Prometheus rule files using validation rules from config file.")
	filePaths              = validateCmd.Arg("path", "File paths to be validated, can be passed as a glob.").Required().Strings()
	validationOutputFormat = validateCmd.Flag("output", "Format of the output.").Default("text").Enum("text", "json", "yaml")
	color                  = validateCmd.Flag("color", "Use color output.").Bool()

	docsCmd          = app.Command("validation-docs", "Print human readable form of the validation rules from config file.")
	docsOutputFormat = docsCmd.Flag("output", "Format of the output.").Default("text").Enum("text", "markdown", "html")
)

func main() {
	currentCommand := kingpin.MustParse(app.Parse(os.Args[1:]))

	configFile, err := os.Open(*configFile)
	if err != nil {
		fmt.Printf("Failed to open config file: %v \n", err)
		os.Exit(1)
	}

	validationConfig := &config.Config{}
	decoder := yaml.NewDecoder(configFile)
	decoder.KnownFields(true)
	if err := decoder.Decode(validationConfig); err != nil {
		fmt.Printf("Failed to load config file: %v \n", err)
		os.Exit(1)
	}

	var validationRules []*validate.ValidationRule
rulesIteration:
	for _, rule := range validationConfig.ValidationRules {
		for _, disabledRule := range *disabledRules {
			if disabledRule == rule.Name {
				continue rulesIteration
			}
		}
		newRule := validate.NewValidationRule(rule.Name, rule.Scope)
		for _, validatorConfig := range rule.Validations {
			newValidator, err := validator.NewFromConfig(validatorConfig)
			if err != nil {
				fmt.Printf("Error loading validtor form confg: %v\n", err)
				os.Exit(1)
			}
			if newValidator == nil {
				continue
			}
			newRule.AddValidator(newValidator)
		}
		validationRules = append(validationRules, newRule)
	}

	var filesToBeValidated []string
	for _, path := range *filePaths {
		paths, err := filepath.Glob(path)
		if err != nil {
			fmt.Printf("Error expanding glob: %v \n", err)
			os.Exit(1)
		}
		for _, p := range paths {
			filesToBeValidated = append(filesToBeValidated, p)
		}
	}

	switch currentCommand {
	case docsCmd.FullCommand():
		output := report.NewIndentedOutput(2, false)
		for _, rule := range validationRules {
			output.AddLine("")
			output.AddLine(rule.Name() + ":")
			output.IncreaseIndentation()
			for _, validatorText := range rule.ValidationTexts() {
				output.AddLine("- " + validatorText)
			}
			output.DecreaseIndentation()
		}
		fmt.Println(output.Text())
	case validateCmd.FullCommand():
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
