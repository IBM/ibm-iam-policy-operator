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

package resources

import (
	"os"
	"reflect"
	gorun "runtime"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1 "github.com/IBM/ibm-iam-policy-operator/api/v1"

	"github.com/IBM/ibm-iam-policy-operator/controllers/constants"
	"github.com/IBM/ibm-iam-policy-operator/controllers/utils"
)

var log = logf.Log.WithName("resources")

var trueVar = true
var falseVar = false
var seconds60 int64 = 60
var cpu100 = resource.NewMilliQuantity(100, resource.DecimalSI)        // 100m
var cpu200 = resource.NewMilliQuantity(200, resource.DecimalSI)        // 200m
var memory384 = resource.NewQuantity(384*1024*1024, resource.BinarySI) // 384Mi
var memory128 = resource.NewQuantity(128*1024*1024, resource.BinarySI) // 128Mi
var serviceAccountName = "ibm-iam-policy-controller"

// DeploymentForPolicyController returns a IAM PolicyController Deployment object
func DeploymentForPolicyController(instance *operatorv1.PolicyController) *appsv1.Deployment {
	image := os.Getenv(constants.PolicyControllerImgEnvVar)
	replicas := instance.Spec.Replicas
	resources := instance.Spec.Resources

	if resources == nil {
		resources = &corev1.ResourceRequirements{
			Limits: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:    *cpu200,
				corev1.ResourceMemory: *memory384},
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:    *cpu100,
				corev1.ResourceMemory: *memory128},
		}
	}

	iamPolicyDep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.IamPolicyControllerDepName,
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app": "iam-policy-controller",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "iam-policy-controller",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                        "iam-policy-controller",
						"app.kubernetes.io/instance": "iam-policy-controller",
					},
					Annotations: utils.AnnotationsForMetering(),
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &seconds60,
					ServiceAccountName:            serviceAccountName,
					HostNetwork:                   false,
					HostIPC:                       false,
					HostPID:                       false,
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "beta.kubernetes.io/arch",
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{gorun.GOARCH},
											},
										},
									},
								},
							},
						},
						PodAntiAffinity: &corev1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
								{
									Weight: 100,
									PodAffinityTerm: corev1.PodAffinityTerm{
										TopologyKey: "kubernetes.io/hostname",
										LabelSelector: &metav1.LabelSelector{
											MatchExpressions: []metav1.LabelSelectorRequirement{
												{
													Key:      "app",
													Operator: metav1.LabelSelectorOpIn,
													Values:   []string{"iam-policy-controller"},
												},
											},
										},
									},
								},
							},
						},
					},
					Tolerations: []corev1.Toleration{
						{
							Key:      "dedicated",
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoSchedule,
						},
						{
							Key:      "CriticalAddonsOnly",
							Operator: corev1.TolerationOpExists,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "tmp",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            constants.IamPolicyControllerDepName,
							Image:           image,
							ImagePullPolicy: corev1.PullAlways,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "tmp",
									MountPath: "/tmp",
								},
							},
							Args: []string{"--v=0", "--update-frequency=60"},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{"sh", "-c", "pgrep iam-policy -l"},
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      5,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{"sh", "-c", "exec echo start iam-policy-controller"},
									},
								},
								InitialDelaySeconds: 10,
								TimeoutSeconds:      2,
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &falseVar,
								RunAsNonRoot:             &trueVar,
								ReadOnlyRootFilesystem:   &trueVar,
								AllowPrivilegeEscalation: &falseVar,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
							Resources: *resources,
						},
					},
				},
			},
		},
	}
	return iamPolicyDep
}

// EqualDeployments returns a Boolean
func EqualDeployments(expected *appsv1.Deployment, found *appsv1.Deployment) bool {
	logger := log.WithValues("func", "EqualDeployments")
	if !utils.EqualLabels(found.ObjectMeta.Labels, expected.ObjectMeta.Labels) {
		return false
	}
	if !reflect.DeepEqual(expected.Spec.Replicas, found.Spec.Replicas) {
		logger.Info("Replicas not equal", "Found", found.Spec.Replicas, "Expected", expected.Spec.Replicas)
		return false
	}
	if !EqualPods(expected.Spec.Template, found.Spec.Template) {
		return false
	}
	return true
}

// EqualPods returns a Boolean
func EqualPods(expected corev1.PodTemplateSpec, found corev1.PodTemplateSpec) bool {
	logger := log.WithValues("func", "EqualPods")
	if !utils.EqualLabels(found.ObjectMeta.Labels, expected.ObjectMeta.Labels) {
		return false
	}
	if !utils.EqualAnnotations(found.ObjectMeta.Annotations, expected.ObjectMeta.Annotations) {
		return false
	}
	if !reflect.DeepEqual(found.Spec.ServiceAccountName, expected.Spec.ServiceAccountName) {
		logger.Info("ServiceAccount not equal", "Found", found.Spec.ServiceAccountName, "Expected", expected.Spec.ServiceAccountName)
		return false
	}
	if len(found.Spec.Containers) != len(expected.Spec.Containers) {
		logger.Info("Number of containers not equal", "Found", len(found.Spec.Containers), "Expected", len(expected.Spec.Containers))
		return false
	}
	if !EqualContainers(expected.Spec.Containers[0], found.Spec.Containers[0]) {
		return false
	}
	return true
}

// EqualContainers returns a Boolean
func EqualContainers(expected corev1.Container, found corev1.Container) bool {
	logger := log.WithValues("func", "EqualContainers")
	if !reflect.DeepEqual(found.Name, expected.Name) {
		logger.Info("Container name not equal", "Found", found.Name, "Expected", expected.Name)
		return false
	}
	if !reflect.DeepEqual(found.Image, expected.Image) {
		logger.Info("Image not equal", "Found", found.Image, "Expected", expected.Image)
		return false
	}
	if !reflect.DeepEqual(found.ImagePullPolicy, expected.ImagePullPolicy) {
		logger.Info("ImagePullPolicy not equal", "Found", found.ImagePullPolicy, "Expected", expected.ImagePullPolicy)
		return false
	}
	if !reflect.DeepEqual(found.VolumeMounts, expected.VolumeMounts) {
		logger.Info("VolumeMounts not equal", "Found", found.VolumeMounts, "Expected", expected.VolumeMounts)
		return false
	}
	if !reflect.DeepEqual(found.SecurityContext, expected.SecurityContext) {
		logger.Info("SecurityContext not equal", "Found", found.SecurityContext, "Expected", expected.SecurityContext)
		return false
	}
	if !equalResources(found.Resources, expected.Resources) {
		logger.Info("Resources not equal", "Found", found.Resources, "Expected", expected.Resources)
		return false
	}
	return true
}

func equalResources(found, expected corev1.ResourceRequirements) bool {
	if !found.Limits.Cpu().Equal(*expected.Limits.Cpu()) {
		return false
	}
	if !found.Limits.Memory().Equal(*expected.Limits.Memory()) {
		return false
	}
	if !found.Requests.Cpu().Equal(*expected.Requests.Cpu()) {
		return false
	}
	if !found.Requests.Memory().Equal(*expected.Requests.Memory()) {
		return false
	}
	return true
}
