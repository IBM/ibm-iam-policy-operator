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
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "github.com/IBM/ibm-iam-policy-operator/api/v1"
	"github.com/IBM/ibm-iam-policy-operator/controllers/constants"
	"github.com/IBM/ibm-iam-policy-operator/controllers/resources"
	"github.com/IBM/ibm-iam-policy-operator/controllers/utils"
	"github.com/IBM/ibm-iam-policy-operator/version"
)

// PolicyControllerReconciler reconciles a PolicyController object
type PolicyControllerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=operator.ibm.com,namespace=ibm-common-services,resources=policycontrollers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.ibm.com,namespace=ibm-common-services,resources=policycontrollers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,namespace=ibm-common-services,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *PolicyControllerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("policycontroller", req.NamespacedName)

	// Fetch the PolicyController instance
	instance := &operatorv1.PolicyController{}
	recErr := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if recErr != nil {
		if errors.IsNotFound(recErr) {
			// Request object not instance, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.Info("PolicyController resource instance not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		r.Log.Error(recErr, "Failed to get deploymentForPolicyController")
		return ctrl.Result{}, recErr
	}

	// Set a default Status value
	if len(instance.Status.Nodes) == 0 {
		instance.Status.Nodes = []string{"none"}
		instance.Status.Versions.Reconciled = version.Version
		recErr = r.Client.Status().Update(context.TODO(), instance)
		if recErr != nil {
			r.Log.Error(recErr, "Failed to set default status")
			return ctrl.Result{}, recErr
		}
	}

	// If the Deployment does not exist, create it
	recResult, recErr := r.handleDeployment(instance)
	if recErr != nil {
		return recResult, recErr
	}
	recResult, recErr = r.updateStatus(instance)
	if recErr != nil {
		return recResult, recErr
	}
	return ctrl.Result{}, nil
}

func (r *PolicyControllerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1.PolicyController{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *PolicyControllerReconciler) handleDeployment(instance *operatorv1.PolicyController) (ctrl.Result, error) {
	// Check if this Deployment already exists
	expected := resources.DeploymentForPolicyController(instance)
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: constants.IamPolicyControllerDepName, Namespace: instance.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Deployment
		if err := controllerutil.SetControllerReference(instance, expected, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		r.Log.Info("Creating a new Policy Controller Deployment", "Deployment.Namespace", expected.Namespace, "Deployment.Name", expected.Name)
		err = r.Client.Create(context.TODO(), expected)
		if err != nil && errors.IsAlreadyExists(err) {
			// Already exists from previous reconcile, requeue.
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			r.Log.Error(err, "Failed to create new Policy Controller Deployment", "Deployment.Namespace", expected.Namespace,
				"Deployment.Name", expected.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		r.Log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	} else if !resources.EqualDeployments(expected, found) {
		// If spec is incorrect, update it and requeue
		found.ObjectMeta.Labels = expected.ObjectMeta.Labels
		found.Spec = expected.Spec
		err = r.Client.Update(context.TODO(), found)
		if err != nil {
			r.Log.Error(err, "Failed to update Deployment", "Namespace", found.Namespace, "Name", found.Name)
			return ctrl.Result{}, err
		}
		r.Log.Info("Updating Policy Controller Deployment", "Deployment.Name", found.Name)
		// Spec updated - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}

func (r *PolicyControllerReconciler) updateStatus(instance *operatorv1.PolicyController) (ctrl.Result, error) {
	// Update the PolicyController status with the pod names
	// List the pods for this PolicyController deployment
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(map[string]string{"app": "iam-policy-controller"}),
	}
	if err := r.Client.List(context.TODO(), podList, listOpts...); err != nil {
		r.Log.Error(err, "Failed to list pods", "PolicyController", instance.Namespace, "PolicyController.Name", instance.Name)
		return ctrl.Result{}, err
	}
	podNames := utils.GetPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, instance.Status.Nodes) || version.Version != instance.Status.Versions.Reconciled {
		instance.Status.Nodes = podNames
		instance.Status.Versions.Reconciled = version.Version
		err := r.Client.Status().Update(context.TODO(), instance)
		if err != nil {
			r.Log.Error(err, "Failed to update PolicyController status")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
