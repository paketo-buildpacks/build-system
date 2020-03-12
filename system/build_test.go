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
	lMocks "github.com/buildpacks/libcnb/mocks"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/build-system/system"
	sMocks "github.com/paketo-buildpacks/build-system/system/mocks"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		build        system.Build
		ctx          libcnb.BuildContext
		distribution *lMocks.LayerContributor
		system       *sMocks.System
	)

	it.Before(func() {
		var err error

		ctx.Application.Path, err = ioutil.TempDir("", "build")
		Expect(err).NotTo(HaveOccurred())

		distribution = &lMocks.LayerContributor{}
		distribution.On("Name").Return("distribution")

		system = &sMocks.System{}
		build.Systems = append(build.Systems, system)
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
	})

	it("does not contribute with no participating system", func() {
		system.On("Participate", mock.Anything).Return(false, nil)

		Expect(build.Build(ctx)).To(BeZero())
	})

	it("contributes system with wrapper", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "test-wrapper"), []byte(""), 0644))

		system.On("Participate", mock.Anything).Return(true, nil)
		system.On("Wrapper").Return("test-wrapper")
		system.On("CachePath").Return("test-cache-path", nil)
		system.On("DefaultArguments").Return([]string{"test-argument"})
		system.On("DefaultTarget").Return("test-target")

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(2))
		Expect(result.Layers[0].Name()).To(Equal("cache"))
		Expect(result.Layers[1].Name()).To(Equal("application"))
	})

	it("contributes system with distribution", func() {
		system.On("Participate", mock.Anything).Return(true, nil)
		system.On("Wrapper").Return("test-wrapper")
		system.On("Distribution", mock.Anything).Return("test-distribution")
		system.On("DistributionLayer", mock.Anything, mock.Anything, mock.Anything).Return(distribution, nil)
		system.On("CachePath").Return("test-cache-path", nil)
		system.On("DefaultArguments").Return([]string{"test-argument"})
		system.On("DefaultTarget").Return("test-target")

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(3))
		Expect(result.Layers[0].Name()).To(Equal("distribution"))
		Expect(result.Layers[1].Name()).To(Equal("cache"))
		Expect(result.Layers[2].Name()).To(Equal("application"))
	})

}
