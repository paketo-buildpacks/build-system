/*
 * Copyright 2018-2020 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package system

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/crush"
)

type MavenDistribution struct {
	LayerContributor libpak.DependencyLayerContributor
	Logger           bard.Logger
}

func (m MavenDistribution) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	m.LayerContributor.Logger = m.Logger

	return m.LayerContributor.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
		m.Logger.Bodyf("Expanding to %s", layer.Path)
		if err := crush.ExtractTarGz(artifact, layer.Path, 1); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to expand Maven\n%w", err)
		}

		layer.Build = true
		layer.Cache = true
		return layer, nil
	})
}

func (MavenDistribution) Name() string {
	return "maven"
}

type Maven struct {
	Logger bard.Logger
}

func (Maven) CachePath() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("unable to determine user home directory\n%w", err)
	}

	return filepath.Join(u.HomeDir, ".m2"), nil
}

func (Maven) DefaultArguments() []string {
	return []string{"-Dmaven.test.skip=true", "package"}
}

func (Maven) DefaultTarget() string {
	return filepath.Join("target", "*.[jw]ar")
}

func (Maven) Detect(context libcnb.DetectContext, result *libcnb.DetectResult) error {
	file := filepath.Join(context.Application.Path, "pom.xml")
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("unable to determine if %s exists\n%w", file, err)
	}

	result.Pass = true
	result.Plans = append(result.Plans, libcnb.BuildPlan{
		Provides: []libcnb.BuildPlanProvide{
			{Name: "maven"},
			{Name: "jvm-application"},
		},
		Requires: []libcnb.BuildPlanRequire{
			{Name: "maven"},
			{Name: "jdk"},
		},
	})

	return nil
}

func (Maven) Distribution(layersPath string) string {
	return filepath.Join(layersPath, "maven", "bin", "mvn")
}

func (m Maven) DistributionLayer(resolver libpak.DependencyResolver, cache libpak.DependencyCache, plan *libcnb.BuildpackPlan) (libcnb.LayerContributor, error) {
	dep, err := resolver.Resolve("maven", "")
	if err != nil {
		return nil, fmt.Errorf("unable to find depdency\n%w", err)
	}

	return MavenDistribution{
		LayerContributor: libpak.NewDependencyLayerContributor(dep, cache, plan),
		Logger:           m.Logger,
	}, nil
}

func (Maven) Participate(resolver libpak.PlanEntryResolver) (bool, error) {
	_, ok, err := resolver.Resolve("maven")
	if err != nil {
		return false, fmt.Errorf("unable to resolve maven plan entry\n%w", err)
	}

	return ok, nil
}

func (Maven) Wrapper() string {
	return "mvnw"
}
