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
	"github.com/spf13/cobra"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"strings"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/debugging"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
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
		artifactGraph := graph.ToArtifactGraph(artifacts)
		for _, b := range builds {
			if v, ok := artifactGraph[b.ImageName]; ok {
				if v.DownstreamSync == nil {
					continue
				}
				for i, m := range l {
					runtimeObj, _, _ := debugging.DecodeFromYaml(m, nil, nil)

					switch runtimeObj.(type) {
					case *corev1.Pod:
						pod := runtimeObj.(*corev1.Pod)
						for j, c := range pod.Spec.Containers {
							if c.Image != b.Tag {
								continue
							}
							container := corev1.Container{Name: "downloader", Image: "gcr.io/k8s-skaffold/downloader-helper:v0", VolumeMounts: []corev1.VolumeMount{{Name: "sync-log", MountPath: "/abccc"}}}
							pod.Spec.InitContainers = append(pod.Spec.InitContainers, container)

							pod.Spec.Containers[j].VolumeMounts = append(pod.Spec.Containers[j].VolumeMounts, corev1.VolumeMount{Name: "sync-log", MountPath: "/abccc"})
							configuration, _ := debug.RetrieveImageConfiguration(ctx, &graph.Artifact{ImageName: pod.Spec.Containers[j].Image, Tag: pod.Spec.Containers[j].Image}, map[string]bool{})
							var args []string
							var containerCmd []string
							containerCmd = append(containerCmd, configuration.Entrypoint...)
							containerCmd = append(containerCmd, configuration.Arguments...)
							args = append(args, "--command", strings.Join(containerCmd, ","))

							var targets []string
							for _, entry := range v.DownstreamSync.Entry {
								targets = append(targets, entry.RemoteSrc)
							}
							args = append(args, "--targets", strings.Join(targets, ","))

							excludes := v.DownstreamSync.Excludes
							args = append(args, "--excludes", strings.Join(excludes, ","))

							pod.Spec.Containers[j].Args = args
							pod.Spec.Containers[j].Command = []string{"/abccc/app-server"}
							if pod.Annotations == nil {
								pod.Annotations = map[string]string{}
							}
							pod.Annotations["skaffold/downloader"] = "auto"
							pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{Name: "sync-log", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
							break
						}
						yaml, _ := debugging.EncodeAsYaml(pod)
						l[i] = yaml
					case *appsv1.Deployment:
						d := runtimeObj.(*appsv1.Deployment)
						podSpec := d.Spec.Template.Spec
						for j, c := range podSpec.Containers {
							if c.Image != b.Tag {
								continue
							}
							container := corev1.Container{Name: "downloader", Image: "gcr.io/k8s-skaffold/downloader-helper:v0", VolumeMounts: []corev1.VolumeMount{{Name: "sync-log", MountPath: "/abccc"}}}
							podSpec.InitContainers = append(podSpec.InitContainers, container)
							podSpec.Containers[j].VolumeMounts = append(podSpec.Containers[j].VolumeMounts, corev1.VolumeMount{Name: "sync-log", MountPath: "/abccc"})
							configuration, _ := debug.RetrieveImageConfiguration(ctx, &graph.Artifact{ImageName: podSpec.Containers[0].Image, Tag: podSpec.Containers[0].Image}, map[string]bool{})
							var args []string
							var containerCmd []string
							containerCmd = append(containerCmd, configuration.Entrypoint...)
							containerCmd = append(containerCmd, configuration.Arguments...)
							args = append(args, "--command", strings.Join(containerCmd, ","))

							var targets []string
							for _, entry := range v.DownstreamSync.Entry {
								targets = append(targets, entry.RemoteSrc)
							}
							args = append(args, "--targets", strings.Join(targets, ","))

							excludes := v.DownstreamSync.Excludes
							args = append(args, "--excludes", strings.Join(excludes, ","))

							podSpec.Containers[j].Args = args
							podSpec.Containers[j].Command = []string{"/abccc/app-server"}
							if d.Annotations == nil {
								d.Annotations = map[string]string{}
							}
							d.Annotations["skaffold/downloader"] = "auto"
							podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{Name: "sync-log", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
							d.Spec.Template.Spec = podSpec
							break
						}
						yaml, _ := debugging.EncodeAsYaml(d)
						l[i] = yaml
					}
				}
			}
		}

		return l, nil
	}

}
