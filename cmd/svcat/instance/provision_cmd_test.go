/*
Copyright 2018 The Kubernetes Authors.

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

package instance_test

import (
	"github.com/kubernetes-incubator/service-catalog/cmd/svcat/command"
	. "github.com/kubernetes-incubator/service-catalog/cmd/svcat/instance"
	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat/service-catalog/service-catalogfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Provision Command", func() {
	Describe("NewProvisionCmd", func() {
		It("Builds and returns a cobra command with the correct flags", func() {
			cxt := &command.Context{}
			cmd := NewProvisionCmd(cxt)

			Expect(*cmd).NotTo(BeNil())
			Expect(cmd.Use).To(Equal("provision NAME --plan PLAN --class CLASS"))
			Expect(cmd.Short).To(ContainSubstring("Create a new instance of a service"))
			Expect(cmd.Example).To(ContainSubstring("svcat provision wordpress-mysql-instance --class mysqldb --plan free"))
			Expect(len(cmd.Aliases)).To(Equal(0))

			flag := cmd.Flags().Lookup("plan")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Usage).To(ContainSubstring("The plan name (Required)"))

			flag = cmd.Flags().Lookup("class")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Usage).To(ContainSubstring("The class name (Required)"))

			flag = cmd.Flags().Lookup("external-id")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Usage).To(ContainSubstring("The ID of the instance for use with the OSB SB API (Optional)"))

			flag = cmd.Flags().Lookup("param")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Usage).To(ContainSubstring("Additional parameter to use when provisioning the service, format: NAME=VALUE. Cannot be combined with --params-json, Sensitive information should be placed in a secret and specified with --secret"))

			flag = cmd.Flags().Lookup("secret")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Usage).To(ContainSubstring("Additional parameter, whose value is stored in a secret, to use when provisioning the service, format: SECRET[KEY]"))

			flag = cmd.Flags().Lookup("wait")
			Expect(flag).NotTo(BeNil())
			flag = cmd.Flags().Lookup("namespace")
			Expect(flag).NotTo(BeNil())
		})
	})
	FDescribe("Validate", func() {
		It("succeeds if an instance name is provided", func() {
			cmd := ProvisionCmd{}
			err := cmd.Validate([]string{"bananainstance"})
			Expect(err).NotTo(HaveOccurred())
		})
		It("errors if no instance name is provided", func() {
			cmd := ProvisionCmd{}
			err := cmd.Validate([]string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("an instance name is required"))
		})
		It("errors if both json params and raw params are provided", func() {
			cmd := ProvisionCmd{
				JsonParams: "{\"foo\":\"bar\"}",
				RawParams:  []string{"a=b"},
			}
			err := cmd.Validate([]string{"bananainstance"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("--params-json cannot be used with --param"))
		})
		It("succeeds only if the provided json params are parseable json", func() {
			cmd := ProvisionCmd{
				JsonParams: "{\"foo\":\"bar\"}",
			}
			err := cmd.Validate([]string{"bananainstance"})
			Expect(err).NotTo(HaveOccurred())
			p := make(map[string]interface{})
			p["foo"] = "bar"
			Expect(cmd.Params).To(Equal(p))
		})
		It("successfully parses raw params into the params map", func() {
			cmd := ProvisionCmd{
				RawParams: []string{"a=b"},
			}
			err := cmd.Validate([]string{"bananainstance"})
			Expect(err).NotTo(HaveOccurred())
			p := make(map[string]interface{})
			p["a"] = "b"
			Expect(cmd.Params).To(Equal(p))
		})
		It("errors if the provided json params are not parseable", func() {
			cmd := ProvisionCmd{
				JsonParams: "foo=bar",
			}
			err := cmd.Validate([]string{"bananainstance"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid --params-json value (invalid parameters (foo=bar))"))
		})
		It("parses secrets into the secrets map", func() {
			cmd := ProvisionCmd{
				RawSecrets: []string{"foo[bar]"},
			}
			err := cmd.Validate([]string{"bananainstance"})
			Expect(err).NotTo(HaveOccurred())
		})
		It("errors if secrets aren't parseable", func() {
			cmd := ProvisionCmd{
				RawSecrets: []string{"foo=bar"},
			}
			err := cmd.Validate([]string{"bananainstance"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid --secret value (invalid parameter (foo=bar), must be in MAP[KEY] format)"))
		})
	})
	Describe("Run", func() {
		var (
			instanceKubeName string
			instanceToReturn *v1beta1.ServiceInstance
			namespace        string
		)
		BeforeEach(func() {
			instanceKubeName = "mysql1234"
			namespace = "foobarnamespace"

			instanceToReturn = &v1beta1.ServiceInstance{
				ObjectMeta: v1.ObjectMeta{
					Name: instanceKubeName,
				},
			}
		})

		FIt("Calls the pkg/svcat libs Provision method with the passed in variables and prints output to the user", func() {
			//outputBuffer := &bytes.Buffer{}

			fakeApp, _ := svcat.NewApp(nil, nil, namespace)
			fakeSDK := new(servicecatalogfakes.FakeSvcatClient)
			fakeSDK.ProvisionReturns(instanceToReturn, nil)
			fakeApp.SvcatClient = fakeSDK
			cmd := ProvisionCmd{}
			cmd.Namespaced.ApplyNamespaceFlags(&pflag.FlagSet{})
			cmd.Waitable.ApplyWaitFlags()

			err := cmd.Run()

			Expect(err).NotTo(HaveOccurred())
			Expect(fakeSDK.RegisterCallCount()).To(Equal(1))
		})
	})
})
