/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package e2e_node

import (
	"time"

	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	imageRetryTimeout         = time.Minute * 10 // Image pulls can take a long time and shouldn't cause flakes
	imagePullInterval         = time.Second * 15
	imageConsistentlyTimeout  = time.Second * 30
	imageConsistentlyInterval = time.Second * 5
)

var _ = Describe("Image Container Conformance Test", func() {
	var cl *client.Client

	BeforeEach(func() {
		// Setup the apiserver client
		cl = client.NewOrDie(&restclient.Config{Host: *apiServerAddress})
	})

	Describe("image conformance blackbox test", func() {
		Context("when testing images that exist", func() {
			var conformImages []ConformanceImage
			BeforeEach(func() {
				existImageTags := []string{
					NoPullImagRegistry[pullTestExecHealthz],
					NoPullImagRegistry[pullTestAlpineWithBash],
				}
				for _, existImageTag := range existImageTags {
					conformImage, _ := NewConformanceImage("docker", existImageTag)
					// Pulling images from gcr.io is flaky, so retry failures
					Eventually(func() error {
						return conformImage.Pull()
					}, imageRetryTimeout, imagePullInterval).ShouldNot(HaveOccurred())
					conformImages = append(conformImages, conformImage)
				}
			})

			It("It should present successfully [Conformance]", func() {
				for _, conformImage := range conformImages {
					present, err := conformImage.Present()
					Expect(err).ShouldNot(HaveOccurred())
					Expect(present).To(BeTrue())
				}
			})

			It("should list pulled images [Conformance]", func() {
				image, _ := NewConformanceImage("docker", "")
				tags, err := image.List()
				Expect(err).ShouldNot(HaveOccurred())
				for _, conformImage := range conformImages {
					Expect(tags).To(ContainElement(conformImage.GetTag()))
				}
			})

			AfterEach(func() {
				for _, conformImage := range conformImages {
					conformImage.Remove()
				}
				conformImages = []ConformanceImage{}
			})

		})
		Context("when testing image that does not exist", func() {
			var conformImages []ConformanceImage
			invalidImageTags := []string{
				// nonexistent image registry
				"foo.com/foo/fooimage",
				// nonexistent image
				"gcr.io/google_containers/not_exist",
				// TODO(random-liu): Add test for image pulling credential
			}
			It("should ignore pull failures", func() {
				for _, invalidImageTag := range invalidImageTags {
					conformImage, _ := NewConformanceImage("docker", invalidImageTag)
					// Pulling images from gcr.io is flaky, so retry to make sure failure is not caused by flaky.
					Expect(conformImage.Pull()).Should(HaveOccurred())
					conformImages = append(conformImages, conformImage)
				}

				By("not presenting images [Conformance]", func() {
					for _, conformImage := range conformImages {
						present, err := conformImage.Present()
						Expect(err).ShouldNot(HaveOccurred())
						Expect(present).To(BeFalse())
					}
				})

				By("not listing pulled images [Conformance]", func() {
					image, _ := NewConformanceImage("docker", "")
					tags, err := image.List()
					Expect(err).ShouldNot(HaveOccurred())
					for _, conformImage := range conformImages {
						Expect(tags).NotTo(ContainElement(conformImage.GetTag()))
					}
				})

				By("not removing non-exist images [Conformance]", func() {
					for _, conformImage := range conformImages {
						err := conformImage.Remove()
						Expect(err).Should(HaveOccurred())
					}
				})
			})
		})
	})
})
