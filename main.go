package main

import (
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/fusakla/promruval/v3/pkg/config"
	"github.com/fusakla/promruval/v3/pkg/extractvalidators"
	"github.com/fusakla/promruval/v3/pkg/report"
	"github.com/fusakla/promruval/v3/pkg/validate"
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
	disableParallelization = validateCmd.Flag("disable-parallelization", "Disable parallelization of validation checks.").Bool()

	docsCmd          = app.Command("validation-docs", "Print human readable form of the validation rules from config file.")
	docsOutputFormat = docsCmd.Flag("output", "Format of the output.").Short('o').PlaceHolder("[text,markdown,html]").Default("text").Enum("text", "markdown", "html")
)

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

	validationConfig, err := config.LoadConfiguration(*validateConfigFiles)
	if err != nil {
		exitWithError(err)
	}

	validationRules, err := extractvalidators.ValidationRulesFromConfig(validationConfig, *disabledRules, *enabledRules)
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

		validationReport, err := validate.Cmd(*filePaths, validationConfig, validationRules, *supportLoki, *supportMimir, *supportThanos, *disableParallelization)
		if err != nil {
			exitWithError(err)
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
