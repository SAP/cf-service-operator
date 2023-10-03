/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	cfv1alpha1 "github.com/sap/cf-service-operator/api/v1alpha1"
	"github.com/sap/cf-service-operator/internal/facade"
)

const (
	serviceInstanceFinalizer = "cf.cs.sap.com/service-operator"
)

const (
	serviceInstanceReadyConditionReasonNew             = "FirstSeen"
	serviceInstanceReadyConditionReasonSpaceNotReady   = "SpaceNotReady"
	serviceInstanceReadyConditionReasonError           = "Error"
	serviceInstanceReadyConditionReasonDeletionBlocked = "DeletionBlocked"
	// Additionally, all of facade.InstanceState* may occur as Ready condition reason
)

// ServiceInstanceReconciler reconciles a ServiceInstance object
type ServiceInstanceReconciler struct {
	client.Client
	Scheme                   *runtime.Scheme
	ClusterResourceNamespace string
	ClientBuilder            facade.SpaceClientBuilder
}

// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=serviceinstances,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=serviceinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=serviceinstances/finalizers,verbs=update
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=clusterspaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=spaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=servicebindings,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

func (r *ServiceInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("running reconcile")

	// Retrieve target service instance
	serviceInstance := &cfv1alpha1.ServiceInstance{}
	if err := r.Get(ctx, req.NamespacedName, serviceInstance); err != nil {
		if err := client.IgnoreNotFound(err); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "unexpected get error")
		}
		log.Info("not found; ignoring")
		return ctrl.Result{}, nil
	}
	// Call the defaulting webhook logic also here (because defaulting through the webhook might be incomplete in case of generateName usage)
	serviceInstance.Default()

	spec := &serviceInstance.Spec
	status := &serviceInstance.Status
	status.ObservedGeneration = serviceInstance.Generation
	status.LastReconciledAt = &[]metav1.Time{metav1.Now()}[0]

	// Always attempt to update the status
	skipStatusUpdate := false
	defer func() {
		if skipStatusUpdate {
			return
		}
		if err != nil {
			serviceInstance.SetReadyCondition(cfv1alpha1.ConditionFalse, serviceInstanceReadyConditionReasonError, err.Error())
		}
		if updateErr := r.Status().Update(ctx, serviceInstance); updateErr != nil {
			err = utilerrors.NewAggregate([]error{err, updateErr})
			result = ctrl.Result{}
		}
	}()

	// Set a first status (and requeue, because the status update itself will not trigger another reconciliation because of the event filter set)
	if ready := serviceInstance.GetReadyCondition(); ready == nil {
		serviceInstance.SetReadyCondition(cfv1alpha1.ConditionUnknown, serviceInstanceReadyConditionReasonNew, "First seen")
		return ctrl.Result{Requeue: true}, nil
	}

	// Retrieve referenced space
	var space cfv1alpha1.GenericSpace
	var spaceSecretName types.NamespacedName

	if spec.SpaceName != "" {
		spaceName := types.NamespacedName{
			Namespace: serviceInstance.Namespace,
			Name:      spec.SpaceName,
		}
		space = &cfv1alpha1.Space{}
		if err := r.Get(ctx, spaceName, space); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed to get Space, name: %s", spec.SpaceName)
		}
		spaceSecretName = types.NamespacedName{
			Namespace: serviceInstance.Namespace,
			Name:      space.GetSpec().AuthSecretName,
		}
	} else if spec.ClusterSpaceName != "" {
		clusterSpaceName := types.NamespacedName{
			Name: spec.ClusterSpaceName,
		}
		space = &cfv1alpha1.ClusterSpace{}
		if err := r.Get(ctx, clusterSpaceName, space); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed to get ClusterSpace, name: %s", spec.ClusterSpaceName)
		}
		spaceSecretName = types.NamespacedName{
			Namespace: r.ClusterResourceNamespace,
			Name:      space.GetSpec().AuthSecretName,
		}
	}

	spaceGuid := space.GetSpec().Guid
	if spaceGuid == "" {
		spaceGuid = space.GetStatus().SpaceGuid
	}

	spaceSecret := &corev1.Secret{}
	if err := r.Get(ctx, spaceSecretName, spaceSecret); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to get Secret containing space credentials, secret name: %s", spaceSecretName)
	}

	// Find depending service bindings
	serviceBindingList := &cfv1alpha1.ServiceBindingList{}
	if err := client.NewNamespacedClient(r.Client, serviceInstance.Namespace).List(
		ctx,
		serviceBindingList,
		client.MatchingLabels{cfv1alpha1.LabelKeyServiceInstance: serviceInstance.Name},
	); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to list depending service bindings")
	}

	// Require readiness of space unless in deletion case
	if serviceInstance.DeletionTimestamp.IsZero() {
		if !space.IsReady() {
			serviceInstance.SetReadyCondition(cfv1alpha1.ConditionUnknown, serviceInstanceReadyConditionReasonSpaceNotReady,
				fmt.Sprintf("Referenced %s is not ready, name: %s", space.GetKind(), space.GetName()))
			// TODO: apply some increasing period, depending on the age of the last update
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		if spaceGuid == "" {
			return ctrl.Result{}, fmt.Errorf("unexpected error; unable to find guid on ready %s: name: %s", space.GetKind(), space.GetName())
		}
	}

	// Build cloud foundry client
	var client facade.SpaceClient
	if spaceGuid != "" {
		client, err = r.ClientBuilder(spaceGuid, string(spaceSecret.Data["url"]), string(spaceSecret.Data["username"]), string(spaceSecret.Data["password"]))
		if err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed to build the client from secret %s", spaceSecretName)
		}
	}

	// Retrieve cloud foundry instance
	var cfinstance *facade.Instance
	if client != nil {
		cfinstance, err = client.GetInstance(string(serviceInstance.UID))
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if serviceInstance.DeletionTimestamp.IsZero() {
		// Create/update case
		if !containsString(serviceInstance.Finalizers, serviceInstanceFinalizer) {
			controllerutil.AddFinalizer(serviceInstance, serviceInstanceFinalizer)
			if err := r.Update(ctx, serviceInstance); err != nil {
				return ctrl.Result{}, err
			}
		}

		servicePlanGuid := spec.ServicePlanGuid
		if servicePlanGuid == "" {
			servicePlanGuid, err = client.FindServicePlan(spec.ServiceOfferingName, spec.ServicePlanName, spaceGuid)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		var parameterObjects []map[string]interface{}
		if spec.Parameters != nil {
			obj, err := unmarshalObject(spec.Parameters.Raw)
			if err != nil {
				return ctrl.Result{}, errors.Wrap(err, "error decoding inline parameters")
			}
			parameterObjects = append(parameterObjects, obj)
		}
		for _, pf := range spec.ParametersFrom {
			secretName := types.NamespacedName{
				Namespace: serviceInstance.Namespace,
				Name:      pf.SecretKeyRef.Name,
			}
			secret := &corev1.Secret{}
			if err := r.Get(ctx, secretName, secret); err != nil {
				return ctrl.Result{}, errors.Wrapf(err, "failed to get Secret containing service instance parameters, secret name: %s", secretName)
			}
			if raw, ok := secret.Data[pf.SecretKeyRef.Key]; ok {
				obj, err := unmarshalObject(raw)
				if err != nil {
					return ctrl.Result{}, errors.Wrapf(err, "error decoding parameters from secret, secret name: %s, key: %s", secretName, pf.SecretKeyRef.Key)
				}
				parameterObjects = append(parameterObjects, obj)
			} else {
				return ctrl.Result{}, fmt.Errorf("secret key not found, secret name: %s, key: %s", secretName, pf.SecretKeyRef.Key)
			}
		}
		parameters, err := mergeObjects(parameterObjects...)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to unmarshal/merge parameters")
		}

		status.ServiceInstanceDigest = facade.ObjectHash(map[string]interface{}{"generation": serviceInstance.Generation, "parameters": parameters})

		recreateOnCreationFailure := serviceInstance.Annotations["service-operator.cf.cs.sap.com/recreate-on-creation-failure"] == "true"
		inRecreation := false

		if cfinstance == nil {
			log.V(1).Info("triggering creation")
			if err := client.CreateInstance(
				spec.Name,
				servicePlanGuid,
				parameters,
				spec.Tags,
				string(serviceInstance.UID),
				serviceInstance.Generation,
			); err != nil {
				return ctrl.Result{}, err
			}
			status.LastModifiedAt = &[]metav1.Time{metav1.Now()}[0]
		} else {
			if cfinstance.State == facade.InstanceStateDeleting {
				// This is the re-creation case; nothing to, we just wait until it is gone
			} else if recreateOnCreationFailure && cfinstance.State == facade.InstanceStateCreatedFailed {
				// Re-create instance
				log.V(1).Info("triggering re-creation")
				if err := client.DeleteInstance(cfinstance.Guid); err != nil {
					return ctrl.Result{}, err
				}
				status.LastModifiedAt = &[]metav1.Time{metav1.Now()}[0]
				inRecreation = true
				// Clear instance, so it will be re-read below
				cfinstance = nil
			} else if cfinstance.Generation < serviceInstance.Generation || cfinstance.ParameterHash != facade.ObjectHash(parameters) ||
				cfinstance.State == facade.InstanceStateCreatedFailed || cfinstance.State == facade.InstanceStateUpdateFailed {
				log.V(1).Info("triggering update")
				updateName := spec.Name
				if updateName == cfinstance.Name {
					updateName = ""
				}
				updateServicePlanGuid := servicePlanGuid
				if updateServicePlanGuid == cfinstance.ServicePlanGuid {
					updateServicePlanGuid = ""
				}
				updateParameters := parameters
				// note: actually it would be best to pass an empty map (instead of nil) to the update call,
				// in the case that no parameters have been defined in spec (neither spec.Parameters, nor spec.ParametersFrom);
				// because then parameters would be cleared in the cloud foundry instance, which would match the expected behavior;
				// but that does not work (e.g. uaa rejects an empty map if any non-empty parameters have been set in previous calls);
				// so we are forced to pass nil here, which means that no parameters at all will be passed to the update call;
				// which means that previous parameters remain as they are, and are not cleared;
				// which is counter-intuitive ...
				// if updateParameters == nil {
				// 	updateParameters = make(map[string]interface{})
				// }
				updateTags := spec.Tags
				if updateTags == nil {
					updateTags = make([]string, 0)
				}
				if err := client.UpdateInstance(
					cfinstance.Guid,
					updateName,
					updateServicePlanGuid,
					updateParameters,
					updateTags,
					serviceInstance.Generation,
				); err != nil {
					return ctrl.Result{}, err
				}
				status.LastModifiedAt = &[]metav1.Time{metav1.Now()}[0]
				// Clear instance, so it will be re-read below
				cfinstance = nil
			}
		}

		if cfinstance == nil {
			// Re-retrieve cloud foundry instance; this happens exactly if the instance was created or updated above
			cfinstance, err = client.GetInstance(string(serviceInstance.UID))
			if err != nil {
				return ctrl.Result{}, err
			}
			if cfinstance == nil {
				if inRecreation {
					// This is the re-create case, if the instance is already gone (maybe deleted synchronously)
					return ctrl.Result{Requeue: true}, nil
				}
				return ctrl.Result{}, fmt.Errorf("unexpected error; instance not found in cloud foundry although it should exist")
			}
		}

		// Update status
		status.SpaceGuid = spaceGuid
		status.ServicePlanGuid = servicePlanGuid
		status.ServiceInstanceGuid = cfinstance.Guid
		switch cfinstance.State {
		case facade.InstanceStateReady:
			serviceInstance.SetReadyCondition(cfv1alpha1.ConditionTrue, string(cfinstance.State), cfinstance.StateDescription)
			// TODO: apply some increasing period, depending on the age of the last update
			return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
		case facade.InstanceStateCreatedFailed, facade.InstanceStateUpdateFailed, facade.InstanceStateDeleteFailed:
			serviceInstance.SetReadyCondition(cfv1alpha1.ConditionFalse, string(cfinstance.State), cfinstance.StateDescription)
			// TODO: apply some increasing period, depending on the age of the last update
			return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
		default:
			serviceInstance.SetReadyCondition(cfv1alpha1.ConditionUnknown, string(cfinstance.State), cfinstance.StateDescription)
			// TODO: apply some increasing period, depending on the age of the last update
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	} else if len(serviceBindingList.Items) > 0 {
		serviceInstance.SetReadyCondition(cfv1alpha1.ConditionUnknown, serviceInstanceReadyConditionReasonDeletionBlocked, "Waiting for deletion of depending service bindings")
		// TODO: apply some increasing period, depending on the age of the last update
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	} else if len(removeString(serviceInstance.Finalizers, serviceInstanceFinalizer)) > 0 {
		serviceInstance.SetReadyCondition(cfv1alpha1.ConditionUnknown, serviceInstanceReadyConditionReasonDeletionBlocked, "Deletion blocked due to foreign finalizers")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		// TODO: apply some increasing period, depending on the age of the last update
	} else {
		// Deletion case
		if cfinstance == nil {
			if containsString(serviceInstance.Finalizers, serviceInstanceFinalizer) {
				controllerutil.RemoveFinalizer(serviceInstance, serviceInstanceFinalizer)
				if err := r.Update(ctx, serviceInstance); err != nil {
					return ctrl.Result{}, err
				}
			}
			// skip status update, since the instance will anyway deleted timely by the API server
			// this will suppress unnecessary ugly 409'ish error messages in the logs
			// (occurring in the case that API server would delete the resource in the course of the subsequent reconciliation)
			skipStatusUpdate = true
			return ctrl.Result{}, nil
		} else {
			if cfinstance.State != facade.InstanceStateDeleting {
				log.V(1).Info("triggering deletion")
				if err := client.DeleteInstance(cfinstance.Guid); err != nil {
					return ctrl.Result{}, err
				}
				status.LastModifiedAt = &[]metav1.Time{metav1.Now()}[0]
				cfinstance.State = facade.InstanceStateUnknown
				cfinstance.StateDescription = "Deletion triggered."
			}
			serviceInstance.SetReadyCondition(cfv1alpha1.ConditionUnknown, string(cfinstance.State), cfinstance.StateDescription)
			// TODO: apply some increasing period, depending on the age of the last update
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cfv1alpha1.ServiceInstance{}).
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{})).
		Complete(r)
}
