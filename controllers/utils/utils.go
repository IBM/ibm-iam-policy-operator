//
// Copyright 2021 IBM Corporation
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

package utils

import (
	"reflect"

	"github.com/IBM/ibm-iam-policy-operator/controllers/constants"

	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("resources")

// GetPodNames returns the pod names of the array of pods passed in
func GetPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

// Helper functions to check and remove string from a slice of strings.
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

//IBMDEV
func AnnotationsForMetering() map[string]string {
	annotations := map[string]string{
		"productName":                        constants.ProductName,
		"productID":                          constants.ProductID,
		"productMetric":                      constants.ProductMetric,
		"clusterhealth.ibm.com/dependencies": "cert-manager, common-mongodb, icp-management-ingress",
		"openshift.io/scc":                   "restricted",
	}
	return annotations
}

func EqualLabels(found map[string]string, expected map[string]string) bool {
	logger := log.WithValues("func", "EqualLabels")
	if !reflect.DeepEqual(found, expected) {
		logger.Info("Labels not equal", "Found", found, "Expected", expected)
		return false
	}
	return true
}

func EqualAnnotations(found map[string]string, expected map[string]string) bool {
	logger := log.WithValues("func", "EqualAnnotations")
	if !reflect.DeepEqual(found, expected) {
		logger.Info("Annotations not equal", "Found", found, "Expected", expected)
		return false
	}
	return true
}
