package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	doublestar "github.com/bmatcuk/doublestar/v4"
	"github.com/fusakla/promruval/v2/pkg/config"
	"github.com/fusakla/promruval/v2/pkg/prometheus"
	"github.com/fusakla/promruval/v2/pkg/report"
	"github.com/fusakla/promruval/v2/pkg/unmarshaler"
	"github.com/fusakla/promruval/v2/pkg/validate"
	"github.com/fusakla/promruval/v2/pkg/validationrule"
	"github.com/fusakla/promruval/v2/pkg/validator"
	log "github.com/sirupsen/logrus"
)

var (
	// Set using goreleaser ldflags during build, see https://goreleaser.com/environment/#using-the-mainversion
	version = "dev"
	commit  = "none"
	date    = time.Now().Format("2006-01-02")
	builtBy = os.Getenv("USER")

	app                 = kingpin.New("promruval", "Prometheus rules validation tool.")
	validateConfigFiles = app.Flag("config-file", "Path to validation config file. Can be passed multiple times, only validationRules will be reflected from the additional configs.").Short('c').ExistingFiles()
	debug               = app.Flag("debug", "Enable debug logging.").Bool()

	versionCmd = app.Command("version", "Print version and build information.")

	validateCmd            = app.Command("validate", "Validate Prometheus rule files in YAML or jsonnet format using validation rules from config file(s).")
	filePaths              = validateCmd.Arg("path", "Rule file paths to be validated (.yaml, .yml or .jsonnet), can use even double star globs or ~. Will be expanded if not done by bash.").Required().Strings()
	disabledRules          = validateCmd.Flag("disable-rule", "Allows to disable any validation rules by it's name. Can be passed multiple times.").Short('d').Strings()
	enabledRules           = validateCmd.Flag("enable-rule", "Only enable these validation rules. Can be passed multiple times.").Short('e').Strings()
	validationOutputFormat = validateCmd.Flag("output", "Format of the output.").Short('o').PlaceHolder("[text,json,yaml]").Default("text").Enum("text", "json", "yaml")
	color                  = validateCmd.Flag("color", "Use color output.").Bool()
	supportLoki            = validateCmd.Flag("support-loki", "Support Loki rules format.").Bool()
	supportMimir           = validateCmd.Flag("support-mimir", "Support Mimir rules format.").Bool()
	supportThanos          = validateCmd.Flag("support-thanos", "Support Thanos rules format.").Bool()

	docsCmd          = app.Command("validation-docs", "Print human readable form of the validation rules from config file.")
	docsOutputFormat = docsCmd.Flag("output", "Format of the output.").Short('o').PlaceHolder("[text,markdown,html]").Default("text").Enum("text", "markdown", "html")
)

func loadConfigFile(configFilePath string) (*config.Config, error) {
	configLoader := config.NewLoader(configFilePath)
	return configLoader.Load()
}

func validationRulesFromConfig(validationConfig *config.Config) ([]*validationrule.ValidationRule, error) {
	var validationRules []*validationrule.ValidationRule
rulesIteration:
	for _, validationRule := range validationConfig.ValidationRules {
		for _, disabledRule := range *disabledRules {
			if disabledRule == validationRule.Name {
				continue rulesIteration
			}
		}
		for _, enabledRule := range *enabledRules {
			if enabledRule != validationRule.Name {
				continue rulesIteration
			}
		}
		newRule := validationrule.New(validationRule.Name, validationRule.Scope)
		for _, validatorConfig := range validationRule.Validations {
			if err := validator.KnownValidators(validationRule.Scope, []string{validatorConfig.ValidatorType}); err != nil {
				return nil, fmt.Errorf("loading config for validator `%s` in the `%s` rule: %w", validatorConfig.ValidatorType, validationRule.Name, err)
			}
			newValidator, err := validator.NewFromConfig(validationRule.Scope, validatorConfig)
			if err != nil {
				return nil, fmt.Errorf("loading config for validator `%s` in the `%s` rule: %w", validatorConfig.ValidatorType, validationRule.Name, err)
			}
			if newValidator == nil {
				continue
			}
			newRule.AddValidator(newValidator, validatorConfig.AdditionalDetails)
		}
		validationRules = append(validationRules, newRule)
	}
	return validationRules, nil
}

func exitWithError(err error) {
	log.Error(err)
	os.Exit(1)
}

func main() {
	currentCommand := kingpin.MustParse(app.Parse(os.Args[1:]))

	if currentCommand == versionCmd.FullCommand() {
		fmt.Printf("Version: %s\nBuild date: %s\nBuild commit: %s\nBuilt by: %s\n", version, date, commit, builtBy)
		return
	}

	if len(*validateConfigFiles) == 0 {
		app.Fatalf("required flag --config-file not provided, try --help")
	}

	mainValidationConfig, err := loadConfigFile((*validateConfigFiles)[0])
	if err != nil {
		exitWithError(err)
	}
	for _, cf := range (*validateConfigFiles)[1:] {
		validationConfig, err := loadConfigFile(cf)
		if err != nil {
			exitWithError(err)
		}
		if validationConfig.Prometheus.URL != "" {
			mainValidationConfig.Prometheus = validationConfig.Prometheus
		}
		if validationConfig.CustomExcludeAnnotation != "" {
			mainValidationConfig.CustomExcludeAnnotation = validationConfig.CustomExcludeAnnotation
		}
		if validationConfig.CustomDisableComment != "" {
			mainValidationConfig.CustomDisableComment = validationConfig.CustomDisableComment
		}
		mainValidationConfig.ValidationRules = append(mainValidationConfig.ValidationRules, validationConfig.ValidationRules...)
	}
	validationRules, err := validationRulesFromConfig(mainValidationConfig)
	if err != nil {
		exitWithError(err)
	}

	switch currentCommand {
	case docsCmd.FullCommand():
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
		log.SetLevel(log.InfoLevel)
		log.SetOutput(os.Stderr)
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, DisableLevelTruncation: true})
		if *debug {
			log.SetLevel(log.DebugLevel)
		}
		var filesToBeValidated []string
		for _, path := range *filePaths {
			if strings.HasPrefix(path, "~/") {
				home, err := os.UserHomeDir()
				if err != nil {
					exitWithError(fmt.Errorf("failed to get user home directory: %w", err))
				}
				path = filepath.Join(home, path[2:])
			}

			base, pattern := doublestar.SplitPattern(path)
			paths, err := doublestar.Glob(os.DirFS(base), pattern, doublestar.WithFilesOnly(), doublestar.WithFailOnIOErrors(), doublestar.WithFailOnPatternNotExist())
			if err != nil {
				exitWithError(fmt.Errorf("failed expanding glob pattern `%s`: %w", path, err))
			}
			for _, p := range paths {
				filesToBeValidated = append(filesToBeValidated, filepath.Join(base, p))
			}
		}

		if *supportLoki {
			unmarshaler.SupportLoki(true)
		}

		if *supportMimir {
			unmarshaler.SupportMimir(true)
		}

		if *supportThanos {
			unmarshaler.SupportThanos(true)
		}

		var prometheusClient *prometheus.Client
		if mainValidationConfig.Prometheus.URL != "" {
			prometheusClient, err = prometheus.NewClient(mainValidationConfig.Prometheus)
			if err != nil {
				exitWithError(fmt.Errorf("failed to initialize prometheus client: %w", err))
			}
		}

		excludeAnnotation := "disabled_validation_rules"
		if mainValidationConfig.CustomExcludeAnnotation != "" {
			excludeAnnotation = mainValidationConfig.CustomExcludeAnnotation
		}
		disableValidatorsComment := "ignore_validations"
		if mainValidationConfig.CustomDisableComment != "" {
			disableValidatorsComment = mainValidationConfig.CustomDisableComment
		}
		validationReport := validate.Files(filesToBeValidated, validationRules, excludeAnnotation, disableValidatorsComment, prometheusClient)

		if mainValidationConfig.Prometheus.URL != "" {
			prometheusClient.DumpCache()
		}

		var output string
		switch *validationOutputFormat {
		case "text":
			output, err = validationReport.AsText(2, *color)
		case "json":
			output, err = validationReport.AsJSON()
		case "yaml":
			output, err = validationReport.AsYaml()
		}
		if err != nil {
			exitWithError(err)
		}
		fmt.Println(output)
		if validationReport.Failed {
			os.Exit(1)
		}
	}
}
