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
	"errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/debugging"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/spf13/cobra"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
)

// for testing
var doDev = runDev

// NewCmdDev describes the CLI command to run a pipeline in development mode.
func NewCmdDev() *cobra.Command {
	return NewCmd("dev").
		WithDescription("Run a pipeline in development mode").
		WithCommonFlags().
		WithHouseKeepingMessages().
		NoArgs(doDev)
}

func runDev(ctx context.Context, out io.Writer) error {
	prune := func() {}
	if opts.Prune() {
		defer func() {
			prune()
		}()
	}

	cleanup := func() {}
	if opts.Cleanup {
		defer func() {
			cleanup()
		}()
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Note: The latest.SkaffoldConfig is used for both latest schema and latestV2 schema because
			// the latest and latestV2 use the same Build struct. Ideally they should be separated.
			err := withRunner(ctx, out, func(r runner.Runner, configs []util.VersionedConfig) error {
				var artifacts []*latest.Artifact
				for _, cfg := range configs {
					artifacts = append(artifacts, cfg.(*latest.SkaffoldConfig).Build.Artifacts...)
				}

				manifest.AddTransform(downloaderTransformer(ctx, artifacts))
				err := r.Dev(ctx, out, artifacts)
				manifestListByConfig := r.DeployManifests()

				cleanup = func() {
					if err := r.Cleanup(context.Background(), out, false, manifestListByConfig, opts.Command); err != nil {
						log.Entry(ctx).Warn("deployer cleanup:", err)
					}
				}

				if r.HasBuilt() {
					prune = func() {
						if err := r.Prune(context.Background(), out); err != nil {
							log.Entry(ctx).Warn("builder cleanup:", err)
						}
					}
				}

				return err
			})
			if err != nil {
				if !errors.Is(err, runner.ErrorConfigurationChanged) {
					return err
				}
				// Otherwise, the skaffold config has changed.
				// just recreate a new runner and restart a dev loop
			}
		}
	}
}

func downloaderTransformer(ctx context.Context, artifacts []*latest.Artifact) manifest.Transform {

	return func(l manifest.ManifestList, builds []graph.Artifact, registries manifest.Registries) (manifest.ManifestList, error) {
		// if downloader defined
		for i, m := range l {
			runtimeObj, _, _ := debugging.DecodeFromYaml(m, nil, nil)

			switch runtimeObj.(type) {
			case *corev1.Pod:
				pod := runtimeObj.(*corev1.Pod)
				container := corev1.Container{Name: "downloader", Image: "sync:2223334", VolumeMounts: []corev1.VolumeMount{{Name: "sync-log", MountPath: "/abccc"}}}
				pod.Spec.InitContainers = append(pod.Spec.InitContainers, container)

				pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{Name: "sync-log", MountPath: "/abccc"})
				configuration, _ := debug.RetrieveImageConfiguration(ctx, &graph.Artifact{ImageName: pod.Spec.Containers[0].Image, Tag: pod.Spec.Containers[0].Image}, map[string]bool{})
				var args []string
				args = append(args, configuration.Entrypoint...)
				args = append(args, configuration.Arguments...)
				pod.Spec.Containers[0].Args = args
				pod.Spec.Containers[0].Command = []string{"/abccc/app-server"}
				if pod.Annotations == nil {
					pod.Annotations = map[string]string{}
				}
				pod.Annotations["skaffold/downloader"] = "auto"
				pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{Name: "sync-log", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})

				yaml, _ := debugging.EncodeAsYaml(pod)
				l[i] = yaml
			case *appsv1.Deployment:
				d := runtimeObj.(*appsv1.Deployment)
				podSpec := d.Spec.Template.Spec
				container := corev1.Container{Name: "downloader", Image: "sync:2223334", VolumeMounts: []corev1.VolumeMount{{Name: "sync-log", MountPath: "/abccc"}}}
				podSpec.InitContainers = append(podSpec.InitContainers, container)
				podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, corev1.VolumeMount{Name: "sync-log", MountPath: "/abccc"})
				//fmt.Println(podSpec.Containers[0].Command)
				//fmt.Println(podSpec.Containers[0].Args)
				configuration, _ := debug.RetrieveImageConfiguration(ctx, &graph.Artifact{ImageName: podSpec.Containers[0].Image, Tag: podSpec.Containers[0].Image}, map[string]bool{})
				var args []string
				args = append(args, configuration.Entrypoint...)
				args = append(args, configuration.Arguments...)
				podSpec.Containers[0].Args = args
				podSpec.Containers[0].Command = []string{"/abccc/app-server"}
				if d.Annotations == nil {
					d.Annotations = map[string]string{}
				}
				d.Annotations["skaffold/downloader"] = "auto"
				podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{Name: "sync-log", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
				d.Spec.Template.Spec = podSpec

				yaml, _ := debugging.EncodeAsYaml(d)
				l[i] = yaml

			}
		}
		return l, nil
	}

}
