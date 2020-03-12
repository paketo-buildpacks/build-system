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
	"archive/zip"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buildpacks/libcnb"
	"github.com/magiconair/properties"
	"github.com/mattn/go-shellwords"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/crush"
	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/sherpa"
)

type Application struct {
	ApplicationPath  string
	Command          string
	Crush            crush.Crush
	DefaultArguments []string
	DefaultTarget    string
	Executor         effect.Executor
	LayerContributor libpak.LayerContributor
	Logger           bard.Logger
}

func NewApplication(applicationPath string, command string, defaultArguments []string, defaultTarget string) (Application, error) {
	l, err := sherpa.NewFileListing(applicationPath)
	if err != nil {
		return Application{}, fmt.Errorf("unable to create file listing for %s: %w", applicationPath, err)
	}
	expected := map[string][]sherpa.FileEntry{"files": l}

	return Application{
		ApplicationPath:  applicationPath,
		Command:          command,
		DefaultArguments: defaultArguments,
		DefaultTarget:    defaultTarget,
		Executor:         effect.NewExecutor(),
		LayerContributor: libpak.NewLayerContributor("Compiled Application", expected),
		Logger:           bard.NewLogger(os.Stdout),
	}, nil
}

func (a Application) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	layer, err := a.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {
		a.Logger.Body(bard.FormatUserConfig("BP_BUILD_ARGUMENTS", "the arguments passed to the build system",
			strings.Join(a.DefaultArguments, " ")))
		a.Logger.Body(bard.FormatUserConfig("BP_BUILT_ARTIFACT", "the built application artifact", a.DefaultTarget))
		a.Logger.Body(bard.FormatUserConfig("BP_BUILT_MODULE", "the module to find application artifact in", "<ROOT>"))

		arguments, err := a.ResolveArguments()
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to resolve arguments: %w", err)
		}

		a.Logger.Body("Executing %s %s", filepath.Base(a.Command), strings.Join(arguments, " "))
		if err := a.Executor.Execute(effect.Execution{
			Command: a.Command,
			Args:    arguments,
			Dir:     a.ApplicationPath,
			Stdout:  a.Logger.InfoWriter(),
			Stderr:  a.Logger.InfoWriter(),
		}); err != nil {
			return libcnb.Layer{}, fmt.Errorf("error running build: %w", err)
		}

		artifact, err := a.ResolveArtifact()
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to resolve artifact: %w", err)
		}

		in, err := os.Open(artifact)
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to open %s: %w", artifact, err)
		}
		defer in.Close()

		file := filepath.Join(layer.Path, "application.zip")
		if err := sherpa.CopyFile(in, file); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to copy %s to %s: %w", artifact, file, err)
		}

		layer.Cache = true
		return layer, nil
	})
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to contribute application layer: %w", err)
	}

	a.Logger.Header("Removing source code")
	cs, err := ioutil.ReadDir(a.ApplicationPath)
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to list children of %s: %w", a.ApplicationPath, err)
	}
	for _, c := range cs {
		file := filepath.Join(a.ApplicationPath, c.Name())
		if err := os.RemoveAll(file); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to remove %s: %w", file, err)
		}
	}

	file := filepath.Join(layer.Path, "application.zip")
	in, err := os.Open(file)
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to open %s: %w", file, err)
	}
	defer in.Close()

	if err := a.Crush.ExtractZip(in, a.ApplicationPath, 0); err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to extract %s: %w", file, err)
	}

	return layer, nil
}

func (Application) Name() string {
	return "application"
}

func (a Application) ResolveArguments() ([]string, error) {
	var err error
	arguments := a.DefaultArguments

	if s, ok := os.LookupEnv("BP_BUILD_ARGUMENTS"); ok {
		arguments, err = shellwords.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("unable to parse arguments from %s: %w", s, err)
		}
	}

	return arguments, nil
}

func (a Application) ResolveArtifact() (string, error) {
	pattern := a.DefaultTarget
	if s, ok := os.LookupEnv("BP_BUILT_MODULE"); ok {
		pattern = filepath.Join(s, pattern)
	}
	if s, ok := os.LookupEnv("BP_BUILT_ARTIFACT"); ok {
		pattern = s
	}

	file := filepath.Join(a.ApplicationPath, pattern)
	candidates, err := filepath.Glob(file)
	if err != nil {
		return "", fmt.Errorf("unable to find files with %s: %w", pattern, err)
	}

	if len(candidates) == 1 {
		return candidates[0], nil
	}

	var artifacts []string
	for _, c := range candidates {
		ok, err := a.interestingFile(c)
		if err != nil {
			return "", fmt.Errorf("unable to investigate %s: %w", c, err)
		}
		if ok {
			artifacts = append(artifacts, c)
		}
	}

	if len(artifacts) != 1 {
		sort.Strings(artifacts)
		return "", fmt.Errorf("unable to find built artifact (executable JAR or WAR) in %s, candidates: %s", pattern, candidates)
	}

	return artifacts[0], nil
}

func (a Application) interestingEntry(f *zip.File) (bool, error) {
	if f.Name == "WEB-INF/" && f.FileInfo().IsDir() {
		return true, nil
	}

	if f.Name == "META-INF/MANIFEST.MF" {
		m, err := f.Open()
		if err != nil {
			return false, fmt.Errorf("unable to open %s: %w", f.Name, err)
		}
		defer m.Close()

		b, err := ioutil.ReadAll(m)
		if err != nil {
			return false, fmt.Errorf("unable to read %s: %w", f.Name, err)
		}

		p, err := properties.Load(b, properties.UTF8)
		if err != nil {
			return false, fmt.Errorf("unable to parse properties in %s: %w", f.Name, err)
		}

		if _, ok := p.Get("Main-Class"); ok {
			return true, nil
		}
	}

	return false, nil
}

func (a Application) interestingFile(path string) (bool, error) {
	z, err := zip.OpenReader(path)
	if err != nil {
		return false, fmt.Errorf("unable to open %s: %w", path, err)
	}
	defer z.Close()

	for _, f := range z.File {
		if i, err := a.interestingEntry(f); err != nil {
			return false, fmt.Errorf("unable to investigate entry %s/%s: %w", path, f.Name, err)
		} else if i {
			return true, nil
		}
	}

	return false, nil
}
