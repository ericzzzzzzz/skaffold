/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"fmt"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/diagnose"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/parser"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	schemaUtil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
	"github.com/spf13/cobra"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
)

var (
	yamlOnly   bool
	outputFile string
	// for testing
	getRunContext = runcontext.GetRunContext
	getCfgs       = parser.GetAllConfigs
)

// NewCmdDiagnose describes the CLI command to diagnose skaffold.
func NewCmdDiagnose() *cobra.Command {
	return NewCmd("diagnose").
		WithDescription("Run a diagnostic on Skaffold").
		WithExample("Search for configuration issues and print the effective configuration", "diagnose").
		WithExample("Print the effective skaffold.yaml configuration for given profile", "diagnose --yaml-only --profile PROFILE").
		WithCommonFlags().
		WithFlags([]*Flag{
			{Value: &yamlOnly, Name: "yaml-only", DefValue: false, Usage: "Only prints the effective skaffold.yaml configuration"},
			{Value: &outputFile, Name: "output", Shorthand: "o", DefValue: "", Usage: "File to write diagnose result"},
		}).
		NoArgs(doDiagnose)
}

func doDiagnose(ctx context.Context, out io.Writer) error {
	// force absolute path resolution during diagnose
	opts.MakePathsAbsolute = util.Ptr(true)
	configs, err := getCfgs(ctx, opts)
	if err != nil {
		return err
	}
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}

	if !yamlOnly {
		if err := printArtifactDiagnostics(ctx, out, configs); err != nil {
			return err
		}
	}
	// remove the dependency config references since they have already been imported and will be marshalled together.
	for i := range configs {
		configs[i].(*latest.SkaffoldConfig).Dependencies = nil
	}
	traverse(configs)
	buf, err := yaml.MarshalWithSeparator(configs)
	if err != nil {
		return fmt.Errorf("marshalling configuration: %w", err)
	}
	//r := reflect.ValueOf(configs)
	//kind := r.Kind()

	out.Write(buf)

	return nil
}

func printArtifactDiagnostics(ctx context.Context, out io.Writer, configs []schemaUtil.VersionedConfig) error {
	runCtx, err := getRunContext(ctx, opts, configs)
	if err != nil {
		return fmt.Errorf("getting run context: %w", err)
	}
	for _, c := range configs {
		config := c.(*latest.SkaffoldConfig)
		fmt.Fprintln(out, "Skaffold version:", version.Get().GitCommit)
		fmt.Fprintln(out, "Configuration version:", config.APIVersion)
		fmt.Fprintln(out, "Number of artifacts:", len(config.Build.Artifacts))

		if err := diagnose.CheckArtifacts(ctx, runCtx, out); err != nil {
			return fmt.Errorf("running diagnostic on artifacts: %w", err)
		}

		output.Blue.Fprintln(out, "\nConfiguration")
	}
	return nil
}

func traverse(in interface{}) {
	if in == nil {
		return
	}
	o := reflect.ValueOf(in)
	//fmt.Println(o.Kind())
	if o.Kind() == reflect.Ptr {
		o = o.Elem()
	}

	switch o.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < o.Len(); i++ {
			if o.Index(i).CanAddr() && o.Index(i).Kind() != reflect.Interface && o.Index(i).Kind() != reflect.Ptr {
				traverse(o.Index(i).Addr().Interface())
			} else {
				traverse(o.Index(i).Interface())
			}
		}
	case reflect.Struct:
		for i := 0; i < o.NumField(); i++ {
			next := o.Field(i)
			// string
			// *string
			// []string
			if next.Kind() == reflect.String {
				field := o.Type().Field(i)
				if a := field.Tag.Get("skaffold"); a != "" {
					split := strings.Split(a, ",")
					if slices.Contains(split, "template") {
						fmt.Println(field.Name)
						updated, _ := util.ExpandEnvTemplate(next.String(), nil)
						next.SetString(updated)
					}
				}
			} else if next.Kind() == reflect.Slice && next.Type().Elem().Kind() == reflect.String {
				_ = o.Type().Field(i)
			} else if next.CanAddr() && (next.Kind() == reflect.Struct || next.Kind() == reflect.Slice || next.Kind() == reflect.Map || next.Kind() == reflect.Array) {
				traverse(next.Addr().Interface())
			} else {
				traverse(next.Interface())
			}

		}
	case reflect.Map:
		for _, key := range o.MapKeys() {
			traverse(o.MapIndex(key).Addr().Interface())
		}
	default:

	}
}
