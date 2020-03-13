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

type GradleDistribution struct {
	LayerContributor libpak.DependencyLayerContributor
	Logger           bard.Logger
}

func (g GradleDistribution) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	g.LayerContributor.Logger = g.Logger

	return g.LayerContributor.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
		g.Logger.Body("Expanding to %s", layer.Path)
		if err := crush.ExtractZip(artifact, layer.Path, 1); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to expand Gradle: %w", err)
		}

		layer.Build = true
		layer.Cache = true
		return layer, nil
	})
}

func (GradleDistribution) Name() string {
	return "gradle"
}

type Gradle struct {
	Logger bard.Logger
}

func (Gradle) Detect(context libcnb.DetectContext, result *libcnb.DetectResult) error {
	files := []string{
		filepath.Join(context.Application.Path, "build.gradle"),
		filepath.Join(context.Application.Path, "build.gradle.kts"),
	}

	for _, f := range files {
		_, err := os.Stat(f)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return fmt.Errorf("unable to determine if %s exists: %w", f, err)
		}

		result.Pass = true
		result.Plans = append(result.Plans, libcnb.BuildPlan{
			Provides: []libcnb.BuildPlanProvide{
				{Name: "gradle"},
				{Name: "jvm-application"},
			},
			Requires: []libcnb.BuildPlanRequire{
				{Name: "gradle"},
				{Name: "jdk"},
			},
		})
	}

	return nil
}

func (Gradle) CachePath() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("unable to determine user home directory: %w", err)
	}

	return filepath.Join(u.HomeDir, ".gradle"), nil
}

func (Gradle) DefaultArguments() []string {
	return []string{"--no-daemon", "-x", "test", "build"}
}

func (Gradle) DefaultTarget() string {
	return filepath.Join("build", "libs", "*.[jw]ar")
}

func (Gradle) Distribution(layersPath string) string {
	return filepath.Join(layersPath, "gradle", "bin", "gradle")
}

func (g Gradle) DistributionLayer(resolver libpak.DependencyResolver, cache libpak.DependencyCache, plan *libcnb.BuildpackPlan) (libcnb.LayerContributor, error) {
	dep, err := resolver.Resolve("gradle", "")
	if err != nil {
		return nil, fmt.Errorf("unable to find depdency: %w", err)
	}

	return GradleDistribution{
		LayerContributor: libpak.NewDependencyLayerContributor(dep, cache, plan),
		Logger:           g.Logger,
	}, nil
}

func (Gradle) Participate(resolver libpak.PlanEntryResolver) (bool, error) {
	_, ok, err := resolver.Resolve("gradle")
	if err != nil {
		return false, fmt.Errorf("unable to resolve gradle plan entry: %w", err)
	}

	return ok, nil
}

func (Gradle) Wrapper() string {
	return "gradlew"
}
