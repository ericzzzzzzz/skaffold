/*
Copyright 2022 The Skaffold Authors

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

package kubectl

import (
	"context"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/generate"
	rUtil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/transform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/validate"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"io"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"
)

type Kubectl struct {
	cfg                render.Config
	rCfg               latest.RenderConfig
	configName         string
	namespace          string
	generator          generate.Generator
	labels             map[string]string
	manifestOverrides  map[string]string
	transformer        transform.Transformer
	validator          validate.Validator
	transformAllowlist map[apimachinery.GroupKind]latest.ResourceFilter
	transformDenylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

func New(cfg render.Config, rCfg latest.RenderConfig, labels map[string]string, configName string, ns string, manifestOverrides map[string]string) (Kubectl, error) {
	transformAllowlist, transformDenylist, err := rUtil.ConsolidateTransformConfiguration(cfg)
	generator := generate.NewGenerator(cfg.GetWorkingDir(), rCfg.Generate, "")
	if err != nil {
		return Kubectl{}, err
	}

	var validator validate.Validator
	if rCfg.Validate != nil {
		validator, err = validate.NewValidator(*rCfg.Validate)
		if err != nil {
			return Kubectl{}, err
		}
	}

	var transformer transform.Transformer
	if rCfg.Transform != nil {
		transformer, err = transform.NewTransformer(*rCfg.Transform)
		if err != nil {
			return Kubectl{}, err
		}
	}

	if len(manifestOverrides) > 0 {
		err := transformer.Append(latest.Transformer{Name: "apply-setters", ConfigMap: util.EnvMapToSlice(manifestOverrides, ":")})
		if err != nil {
			return Kubectl{}, err
		}
	}

	return Kubectl{
		cfg:                cfg,
		configName:         configName,
		rCfg:               rCfg,
		namespace:          ns,
		generator:          generator,
		labels:             labels,
		manifestOverrides:  manifestOverrides,
		validator:          validator,
		transformer:        transformer,
		transformAllowlist: transformAllowlist,
		transformDenylist:  transformDenylist,
	}, nil
}

func (r Kubectl) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool) (manifest.ManifestListByConfig, error) {
	_, endTrace := instrumentation.StartTrace(ctx, "Render_KubectlManifests")
	log.Entry(ctx).Infof("starting render process")
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"RendererType": "kubectl",
	})
	// get manifest contents from rawManifests and remoteManifests
	manifests, err := r.generator.Generate(ctx, out)
	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}

	manifests, err = r.transformer.Transform(ctx, manifests)

	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}

	opts := rUtil.GenerateHydratedManifestsOptions{
		TransformAllowList:         r.transformAllowlist,
		TransformDenylist:          r.transformDenylist,
		EnablePlatformNodeAffinity: r.cfg.EnablePlatformNodeAffinityInRenderedManifests(),
		EnableGKEARMNodeToleration: r.cfg.EnableGKEARMNodeTolerationInRenderedManifests(),
		Offline:                    offline,
		KubeContext:                r.cfg.GetKubeContext(),
	}
	manifests, err = rUtil.BaseTransform(ctx, manifests, builds, opts, r.labels, r.namespace)

	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}

	err = r.validator.Validate(ctx, manifests)
	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}

	endTrace()
	manifestListByConfig := manifest.NewManifestListByConfig()
	manifestListByConfig.Add(r.configName, manifests)
	return manifestListByConfig, err
}

func (r Kubectl) ManifestDeps() ([]string, error) {
	return nil, nil
}
