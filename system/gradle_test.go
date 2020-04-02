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

package system_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/build-system/system"
	"github.com/paketo-buildpacks/libpak"
	"github.com/sclevine/spec"
)

func testGradle(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		gradle system.Gradle
	)

	context("Build", func() {
		var (
			ctx libcnb.BuildContext
		)

		it.Before(func() {
			var err error

			ctx.Application.Path, err = ioutil.TempDir("", "gradle-application")
			Expect(err).NotTo(HaveOccurred())

			ctx.Layers.Path, err = ioutil.TempDir("", "gradle-layers")
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
			Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
		})

		it("contributes Gradle distribution", func() {
			dr := libpak.DependencyResolver{
				Dependencies: []libpak.BuildpackDependency{
					{
						ID:      "gradle",
						Version: "1.1.1",
						URI:     "https://localhost/stub-gradle.zip",
						SHA256:  "5fa754fef54387acdf1ab3107e4ddcaf141e713cd5f946afad4edfbf9461928f",
						Stacks:  []string{"test-stack-id"},
					},
				},
				StackID: "test-stack-id",
			}

			dc := libpak.DependencyCache{CachePath: "testdata"}

			d, err := gradle.DistributionLayer(dr, dc, &libcnb.BuildpackPlan{})
			Expect(err).NotTo(HaveOccurred())

			layer, err := ctx.Layers.Layer("test-layer")
			Expect(err).NotTo(HaveOccurred())

			layer, err = d.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.Build).To(BeTrue())
			Expect(layer.Cache).To(BeTrue())
			Expect(filepath.Join(layer.Path, "fixture-marker")).To(BeARegularFile())
		})

		it("it participates", func() {
			pr := libpak.PlanEntryResolver{Plan: libcnb.BuildpackPlan{
				Entries: []libcnb.BuildpackPlanEntry{
					{Name: "gradle"},
				},
			}}

			Expect(gradle.Participate(pr)).To(BeTrue())
		})
	})

	context("Detect", func() {
		var (
			ctx    libcnb.DetectContext
			result libcnb.DetectResult
		)

		it.Before(func() {
			var err error

			ctx.Application.Path, err = ioutil.TempDir("", "gradle-application")
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		})

		it("does not modify if it does not detect", func() {
			Expect(gradle.Detect(ctx, &result)).To(Succeed())

			Expect(result.Pass).To(BeFalse())
			Expect(result.Plans).To(HaveLen(0))
		})

		it("modifies result if build.gradle exists", func() {
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "build.gradle"), []byte(""), 0644)).To(Succeed())

			Expect(gradle.Detect(ctx, &result)).To(Succeed())

			Expect(result.Pass).To(BeTrue())
			Expect(result.Plans).To(HaveLen(1))
			Expect(result.Plans[0]).To(Equal(libcnb.BuildPlan{
				Provides: []libcnb.BuildPlanProvide{
					{Name: "gradle"},
					{Name: "jvm-application"},
				},
				Requires: []libcnb.BuildPlanRequire{
					{Name: "gradle"},
					{Name: "jdk"},
				},
			}))
		})

		it("modifies result if build.gradle exists.kts", func() {
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "build.gradle.kts"), []byte(""), 0644)).To(Succeed())

			Expect(gradle.Detect(ctx, &result)).To(Succeed())

			Expect(result.Pass).To(BeTrue())
			Expect(result.Plans).To(HaveLen(1))
			Expect(result.Plans[0]).To(Equal(libcnb.BuildPlan{
				Provides: []libcnb.BuildPlanProvide{
					{Name: "gradle"},
					{Name: "jvm-application"},
				},
				Requires: []libcnb.BuildPlanRequire{
					{Name: "gradle"},
					{Name: "jdk"},
				},
			}))
		})
	})
}
