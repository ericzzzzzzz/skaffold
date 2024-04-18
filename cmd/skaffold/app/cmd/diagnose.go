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

	enableTemplating bool
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
			{Value: &enableTemplating, Name: "enable-templating", DefValue: false, Usage: "Render supported templated fields with golang template engine"},
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
	if enableTemplating {
		if err := applyTemplates(configs); err != nil {
			return err
		}
	}
	buf, err := yaml.MarshalWithSeparator(configs)
	if err != nil {
		return fmt.Errorf("marshalling configuration: %w", err)
	}

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

func applyTemplates(in interface{}) error {
	if in == nil {
		return nil
	}
	o := reflect.Indirect(reflect.ValueOf(in))

	switch o.Kind() {
	case reflect.Struct:
		for i := 0; i < o.NumField(); i++ {
			next := reflect.Indirect(o.Field(i))
			field := o.Type().Field(i)
			if !containTemplateTag(field) {
				// maybe nested fields contain template tag
				if next.Kind() == reflect.Struct || next.Kind() == reflect.Slice || next.Kind() == reflect.Map || next.Kind() == reflect.Array {
					if err := applyTemplates(next.Addr().Interface()); err != nil {
						return err
					}
				} else if next.Kind() == reflect.Interface {
					if err := applyTemplates(next.Interface()); err != nil {
						return err
					}
				}
				continue
			}

			if next.Kind() == reflect.String {
				updated, err := util.ExpandEnvTemplate(next.String(), nil)
				if err != nil {
					return err
				}
				next.SetString(updated)
			} else if next.Kind() == reflect.Slice || next.Kind() == reflect.Array {
				for i := 0; i < next.Len(); i++ {
					nnext := reflect.Indirect(next.Index(i))
					// string and *string
					if nnext.Kind() == reflect.String {
						updated, err := util.ExpandEnvTemplate(nnext.String(), nil)
						if err != nil {
							return err
						}
						nnext.SetString(updated)
					} else {
						return fmt.Errorf("unsupported type: %v.%v.%v template tag can only be used in the following types: string, *string, []string, []*string, map[any]string, map[any]*string",
							o.Type(), next.Type(), nnext.Type())
					}
				}
			} else if next.Kind() == reflect.Map {
				for _, key := range next.MapKeys() {
					nnext := next.MapIndex(key)
					if nnext.Kind() == reflect.Ptr && nnext.Elem().Kind() == reflect.String {
						nnext = nnext.Elem()
						updated, err := util.ExpandEnvTemplate(nnext.String(), nil)
						if err != nil {
							return err
						}
						nnext.SetString(updated)
					} else if nnext.Kind() == reflect.String {
						updated, err := util.ExpandEnvTemplate(nnext.String(), nil)
						if err != nil {
							return err
						}
						next.SetMapIndex(key, reflect.ValueOf(updated))
					} else {
						return fmt.Errorf("unsupported type: %v.%v.%v template tag can only be used in the following types: string, *string, []string, []*string, map[any]string, map[any]*string",
							o.Type(), next.Type(), nnext.Type())
					}
				}
			} else {
				return fmt.Errorf("unsupported type: %v.%v template tag can only be used in the following types: string, *string, []string, []*string, map[any]string, map[any]*string",
					o.Type(), next.Type())
			}

		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < o.Len(); i++ {
			next := reflect.Indirect(o.Index(i))
			if next.Kind() == reflect.Struct || next.Kind() == reflect.Slice || next.Kind() == reflect.Map || next.Kind() == reflect.Array {
				if err := applyTemplates(next.Addr().Interface()); err != nil {
					return err
				}
			} else if next.Kind() == reflect.Interface {
				if err := applyTemplates(next.Interface()); err != nil {
					return err
				}
			}
		}
	case reflect.Map:
		for _, key := range o.MapKeys() {
			next := reflect.Indirect(o.MapIndex(key))
			if next.Kind() == reflect.Struct || next.Kind() == reflect.Slice || next.Kind() == reflect.Map || next.Kind() == reflect.Array {
				if err := applyTemplates(next.Addr().Interface()); err != nil {
					return err
				}
			} else if next.Kind() == reflect.Interface {
				if err := applyTemplates(next.Interface()); err != nil {
					return err
				}
			}
		}
	default:

	}
	return nil
}

func containTemplateTag(sf reflect.StructField) bool {
	v, ok := sf.Tag.Lookup("skaffold")
	if !ok {
		return ok
	}
	split := strings.Split(v, ",")
	return slices.Contains(split, "template")
}
