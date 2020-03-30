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
	"path/filepath"
	"strings"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
)

type Build struct {
	Logger  bard.Logger
	Systems []System
}

func (b Build) Build(context libcnb.BuildContext) (libcnb.BuildResult, error) {
	b.Logger.Title(context.Buildpack)
	result := libcnb.BuildResult{}

	pr := libpak.PlanEntryResolver{Plan: context.Plan}

	dr, err := libpak.NewDependencyResolver(context)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency resolver\n%w", err)
	}

	dc := libpak.NewDependencyCache(context.Buildpack)
	dc.Logger = b.Logger

	for _, s := range b.Systems {
		if ok, err := s.Participate(pr); err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to determine participation\n%w", err)
		} else if !ok {
			continue
		}

		b.Logger.Body(bard.FormatUserConfig("BP_BUILD_ARGUMENTS", "the arguments passed to the build system",
			strings.Join(s.DefaultArguments(), " ")))
		b.Logger.Body(bard.FormatUserConfig("BP_BUILT_MODULE", "the module to find application artifact in", "<ROOT>"))
		b.Logger.Body(bard.FormatUserConfig("BP_BUILT_ARTIFACT", "the built application artifact", s.DefaultTarget()))

		var command string
		wrapper := filepath.Join(context.Application.Path, s.Wrapper())
		if _, err := os.Stat(wrapper); os.IsNotExist(err) {
			command = s.Distribution(context.Layers.Path)

			layer, err := s.DistributionLayer(dr, dc, &result.Plan)
			if err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to create distribution layer\n%w", err)
			}
			result.Layers = append(result.Layers, layer)
		} else if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to stat %s\n%w", wrapper, err)
		} else {
			command = wrapper
		}

		cache, err := s.CachePath()
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to determine cache location\n%w", err)
		}
		c := NewCache(cache)
		c.Logger = b.Logger
		result.Layers = append(result.Layers, c)

		a, err := NewApplication(context.Application.Path, command, s.DefaultArguments(), s.DefaultTarget())
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to create application layer\n%w", err)
		}
		a.Logger = b.Logger
		result.Layers = append(result.Layers, a)
	}

	return result, nil
}
