/*
Copyright 2025.

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

package e2e

import (
	"fmt"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"sigs.k8s.io/rbgs/test/e2e/framework"
	"sigs.k8s.io/rbgs/test/e2e/testcase/rbg"
)

func TestE2E(t *testing.T) {
	fmt.Println("TestE2E")
	f := framework.NewFramework()
	ginkgo.BeforeSuite(func() {
		err := f.BeforeAll()
		gomega.Expect(err).To(gomega.BeNil())
	})

	ginkgo.AfterSuite(func() {
		f.AfterAll()
	})

	ginkgo.AfterEach(func() {
		f.AfterEach()
	})

	gomega.RegisterFailHandler(ginkgo.Fail)

	ginkgo.Describe("Run role based controller e2e tests", func() {
		rbg.RunRbgControllerTestCases(f)
		rbg.RunLwsRbgTestCases(f)
	})

	ginkgo.RunSpecs(t, "run rbg e2e test")
}
