//
// Copyright 2020 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package controllers

import (
	"context"
	"crypto/rand"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1 "github.com/IBM/ibm-iam-policy-operator/api/v1"
	"github.com/IBM/ibm-iam-policy-operator/controllers/constants"
	"github.com/IBM/ibm-iam-policy-operator/version"

	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyController controller", func() {
	const requestName = "example-policycontroller"
	const namespace = "test"
	var (
		ctx              context.Context
		requestNamespace string
		pc               *operatorv1.PolicyController
		namespacedName   types.NamespacedName
	)

	BeforeEach(func() {
		ctx = context.Background()
		requestNamespace = createNSName(namespace)
		By("Creating the Namespace")
		Expect(k8sClient.Create(ctx, namespaceObj(requestNamespace))).Should(Succeed())

		pc = policyControllerObj(requestName, requestNamespace)
		namespacedName = types.NamespacedName{Name: requestName, Namespace: requestNamespace}
		By("Creating a new PolicyController")
		Expect(k8sClient.Create(ctx, pc)).Should(Succeed())
	})

	AfterEach(func() {
		By("Deleting the PolicyController")
		Expect(k8sClient.Delete(ctx, pc)).Should(Succeed())
		By("Deleting the Namespace")
		Expect(k8sClient.Delete(ctx, namespaceObj(requestNamespace))).Should(Succeed())
	})

	Context("When creating a PolicyController instance", func() {
		It("Should create Policy Controller deployment", func() {
			createdPC := &operatorv1.PolicyController{}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespacedName, createdPC)
			}, timeout, interval).Should(BeNil())

			By("Check status of PolicyController")
			Eventually(func() string {
				Expect(k8sClient.Get(ctx, namespacedName, createdPC)).Should(Succeed())
				return createdPC.Status.Versions.Reconciled
			}, timeout, interval).Should(Equal(version.Version))

			By("Check if Deployment was created")
			foundDeploy := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: constants.IamPolicyControllerDepName, Namespace: requestNamespace}, foundDeploy)
			}, timeout, interval).Should(Succeed())
		})
	})
})

func createNSName(prefix string) string {
	suffix := make([]byte, 20)
	rand.Read(suffix)
	return fmt.Sprintf("%s-%x", prefix, suffix)
}

func namespaceObj(name string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func policyControllerObj(name, namespace string) *operatorv1.PolicyController {
	return &operatorv1.PolicyController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: operatorv1.PolicyControllerSpec{},
	}
}
