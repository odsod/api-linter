// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aep-dev/api-linter/internal"
	"github.com/aep-dev/api-linter/lint"
	lint_v2 "github.com/aep-dev/api-linter/lint/v2"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

type cli struct {
	ConfigPath                string
	FormatType                string
	OutputPath                string
	ExitStatusOnLintFailure   bool
	VersionFlag               bool
	ProtoImportPaths          []string
	ProtoFiles                []string
	ProtoDescPath             []string
	EnabledRules              []string
	DisabledRules             []string
	ListRulesFlag             bool
	DebugFlag                 bool
	IgnoreCommentDisablesFlag bool
}

// ExitForLintFailure indicates that a problem was found during linting.
//
//lint:ignore ST1012 modifying this variable name is a breaking change.
var ExitForLintFailure = errors.New("found problems during linting")

func newCli(args []string) *cli {
	// Define flag variables.
	var cfgFlag string
	var fmtFlag string
	var outFlag string
	var setExitStatusOnLintFailure bool
	var versionFlag bool
	var protoImportFlag []string
	var protoDescFlag []string
	var ruleEnableFlag []string
	var ruleDisableFlag []string
	var listRulesFlag bool
	var debugFlag bool
	var ignoreCommentDisablesFlag bool

	// Register flag variables.
	fs := pflag.NewFlagSet("api-linter", pflag.ExitOnError)
	fs.StringVar(&cfgFlag, "config", "", "The linter config file.")
	fs.StringVar(&fmtFlag, "output-format", "", "The format of the linting results.\nSupported formats include \"yaml\", \"json\",\"github\" and \"summary\" table.\nYAML is the default.")
	fs.StringVarP(&outFlag, "output-path", "o", "", "The output file path.\nIf not given, the linting results will be printed out to STDOUT.")
	fs.BoolVar(&setExitStatusOnLintFailure, "set-exit-status", false, "Return exit status 1 when lint errors are found.")
	fs.BoolVar(&versionFlag, "version", false, "Print version and exit.")
	fs.StringArrayVarP(&protoImportFlag, "proto-path", "I", nil, "The folder for searching proto imports.\nMay be specified multiple times; directories will be searched in order.\nThe current working directory is always used.")
	fs.StringArrayVar(&protoDescFlag, "descriptor-set-in", nil, "The file containing a FileDescriptorSet for searching proto imports.\nMay be specified multiple times.")
	fs.StringArrayVar(&ruleEnableFlag, "enable-rule", nil, "Enable a rule with the given name.\nMay be specified multiple times.")
	fs.StringArrayVar(&ruleDisableFlag, "disable-rule", nil, "Disable a rule with the given name.\nMay be specified multiple times.")
	fs.BoolVar(&listRulesFlag, "list-rules", false, "Print the rules and exit.  Honors the output-format flag.")
	fs.BoolVar(&debugFlag, "debug", false, "Run in debug mode. Panics will print stack.")
	fs.BoolVar(&ignoreCommentDisablesFlag, "ignore-comment-disables", false, "If set to true, disable comments will be ignored.\nThis is helpful when strict enforcement of AIPs are necessary and\nproto definitions should not be able to disable checks.")

	// Parse flags.
	err := fs.Parse(args)
	if err != nil {
		panic(err)
	}

	return &cli{
		ConfigPath:                cfgFlag,
		FormatType:                fmtFlag,
		OutputPath:                outFlag,
		ExitStatusOnLintFailure:   setExitStatusOnLintFailure,
		ProtoImportPaths:          append(protoImportFlag, "."),
		ProtoDescPath:             protoDescFlag,
		EnabledRules:              ruleEnableFlag,
		DisabledRules:             ruleDisableFlag,
		ProtoFiles:                fs.Args(),
		VersionFlag:               versionFlag,
		ListRulesFlag:             listRulesFlag,
		DebugFlag:                 debugFlag,
		IgnoreCommentDisablesFlag: ignoreCommentDisablesFlag,
	}
}

func (c *cli) lint(rules lint.RuleRegistry, rulesV2 lint_v2.RuleRegistry, configs lint.Configs) error {
	// Print version and exit if asked.
	if c.VersionFlag {
		fmt.Printf("api-linter %s\n", internal.Version)
		return nil
	}

	if c.ListRulesFlag {
		return outputRules(c.FormatType)
	}

	// Pre-check if there are files to lint.
	if len(c.ProtoFiles) == 0 {
		return fmt.Errorf("no file to lint")
	}
	// Read linter config and append it to the default.
	if c.ConfigPath != "" {
		config, err := lint.ReadConfigsFromFile(c.ConfigPath)
		if err != nil {
			return err
		}
		configs = append(configs, config...)
	}
	// Add configs for the enabled rules.
	configs = append(configs, lint.Config{
		EnabledRules: c.EnabledRules,
	})
	// Add configs for the disabled rules.
	configs = append(configs, lint.Config{
		DisabledRules: c.DisabledRules,
	})

	// V1 Pipeline
	resultsV1, err := c.runV1(rules, configs)
	if err != nil {
		return err
	}

	// V2 Pipeline
	// Note: We convert V1 configs to V2 configs. Since they are identical in structure,
	// we just cast or copy. For PoC, let's assume they are compatible or just use empty.
	var configsV2 lint_v2.Configs
	for _, cfg := range configs {
		configsV2 = append(configsV2, lint_v2.Config{
			EnabledRules:  cfg.EnabledRules,
			DisabledRules: cfg.DisabledRules,
		})
	}
	resultsV2, err := c.runV2(rulesV2, configsV2)
	if err != nil {
		// Log V2 failure but don't stop V1?
		// Actually, let's return error for now to be safe.
		return fmt.Errorf("V2 pipeline failed: %w", err)
	}

	// Merge Results
	results := mergeResponses(resultsV1, resultsV2)

	// Determine the output for writing the results.
	// Stdout is the default output.
	w := os.Stdout
	if c.OutputPath != "" {
		var err error
		w, err = os.Create(c.OutputPath)
		if err != nil {
			return err
		}
		defer w.Close()
	}

	// Determine the format for printing the results.
	// YAML format is the default.
	marshal := getOutputFormatFunc(c.FormatType)

	// Print the results.
	b, err := marshal(results)
	if err != nil {
		return err
	}
	if _, err = w.Write(b); err != nil {
		return err
	}

	// Return error on lint failure which subsequently
	// exits with a non-zero status code
	if c.ExitStatusOnLintFailure && anyProblemsInMerged(results) {
		return ExitForLintFailure
	}

	return nil
}

func anyProblemsInMerged(results []interface{}) bool {
	for _, r := range results {
		if ur, ok := r.(*unifiedResponse); ok {
			if len(ur.Problems) > 0 {
				return true
			}
		}
	}
	return false
}

var outputFormatFuncs = map[string]formatFunc{
	"yaml": yaml.Marshal,
	"yml":  yaml.Marshal,
	"json": json.Marshal,
	"github": func(i interface{}) ([]byte, error) {
		switch v := i.(type) {
		case []lint.Response:
			return formatGitHubActionOutput(v), nil
		default:
			return json.Marshal(v)
		}
	},
	"summary": func(i interface{}) ([]byte, error) {
		switch v := i.(type) {
		case []lint.Response:
			return printSummaryTable(v)
		case listedRules:
			return v.printSummaryTable()
		default:
			return json.Marshal(v)
		}
	},
}

type formatFunc func(interface{}) ([]byte, error)

func getOutputFormatFunc(formatType string) formatFunc {
	if f, found := outputFormatFuncs[strings.ToLower(formatType)]; found {
		return f
	}
	return yaml.Marshal
}
