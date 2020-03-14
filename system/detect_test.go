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
	"fmt"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/build-system/system"
	"github.com/paketo-buildpacks/build-system/system/mocks"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx    libcnb.DetectContext
		detect system.Detect
		system *mocks.System
	)

	it.Before(func() {
		system = &mocks.System{}
		detect.Systems = append(detect.Systems, system)
	})

	it("returns unmodified result", func() {
		system.Mock.On("Detect", mock.Anything, mock.Anything).Return(nil)

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{}))
	})

	it("returns modified result", func() {
		system.Mock.On("Detect", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			result := args.Get(1).(*libcnb.DetectResult)
			result.Pass = true
			result.Plans = []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "test-provide-name"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "test-require-name"},
					},
				},
			}
		}).Return(nil)
		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "test-provide-name"},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: "test-require-name"},
					},
				},
			},
		}))
	})

	it("returns error", func() {
		system.Mock.On("Detect", mock.Anything, mock.Anything).Return(fmt.Errorf("test-error"))

		_, err := detect.Detect(ctx)
		Expect(err).To(MatchError("unable to detect\ntest-error"))
	})
}
