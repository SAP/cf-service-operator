/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
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
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	cfv1alpha1 "github.com/sap/cf-service-operator/api/v1alpha1"
	"github.com/sap/cf-service-operator/internal/binding"
	"github.com/sap/cf-service-operator/internal/config"
	"github.com/sap/cf-service-operator/internal/facade"
)

const (
	serviceBindingFinalizer = "cf.cs.sap.com/service-operator"
)

const (
	serviceBindingReadyConditionReasonNew                     = "FirstSeen"
	serviceBindingReadyConditionReasonSpaceNotReady           = "SpaceNotReady"
	serviceBindingReadyConditionReasonServiceInstanceNotReady = "ServiceInstanceNotReady"
	serviceBindingReadyConditionReasonError                   = "Error"
	serviceBindingReadyConditionReasonDeletionBlocked         = "DeletionBlocked"
	// Additionally, all of facade.BindingState* may occur as Ready condition reason
)

// ServiceBindingReconciler reconciles a ServiceBinding object
type ServiceBindingReconciler struct {
	client.Client
	Scheme                   *runtime.Scheme
	ClusterResourceNamespace string
	EnableBindingMetadata    bool
	ClientBuilder            facade.SpaceClientBuilder
	Config                   *config.Config
}

// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=servicebindings,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=servicebindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=servicebindings/finalizers,verbs=update
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=serviceinstances,verbs=get;list;watch
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=clusterspaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=spaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch

func (r *ServiceBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(2).Info("Running reconcile")

	// Retrieve target service binding
	serviceBinding := &cfv1alpha1.ServiceBinding{}
	if err := r.Get(ctx, req.NamespacedName, serviceBinding); err != nil {
		if err := client.IgnoreNotFound(err); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "unexpected get error")
		}
		log.V(1).Info("Not found; ignoring")
		return ctrl.Result{}, nil
	}

	// Call defaulting webhook logic also here, because defaulting through webhook might be
	// incomplete in case of generateName usage
	serviceBinding.Default()

	spec := &serviceBinding.Spec
	status := &serviceBinding.Status
	status.ObservedGeneration = serviceBinding.Generation
	status.LastReconciledAt = &[]metav1.Time{metav1.Now()}[0]

	// Always attempt to update status at end of this reconciliation
	skipStatusUpdate := false
	defer func() {
		if skipStatusUpdate {
			return
		}

		if err != nil {
			serviceBinding.SetReadyCondition(cfv1alpha1.ConditionFalse, serviceBindingReadyConditionReasonError, err.Error())
		}

		// update status of service binding CR
		if updateErr := r.Status().Update(ctx, serviceBinding); updateErr != nil {
			err = utilerrors.NewAggregate([]error{err, updateErr})
			result = ctrl.Result{}
		}
	}()

	// Set first status (and requeue, because status update itself does not peform another
	// reconciliation because of set event filter
	if ready := serviceBinding.GetReadyCondition(); ready == nil {
		serviceBinding.SetReadyCondition(cfv1alpha1.ConditionUnknown, serviceBindingReadyConditionReasonNew, "First seen")
		return ctrl.Result{Requeue: true}, nil
	}

	// Retrieve referenced service instance
	serviceInstanceName := types.NamespacedName{
		Namespace: serviceBinding.Namespace,
		Name:      spec.ServiceInstanceName,
	}
	serviceInstance := &cfv1alpha1.ServiceInstance{}
	if err := r.Get(ctx, serviceInstanceName, serviceInstance); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to get ServiceInstance, name: %s", spec.ServiceInstanceName)
	}
	// Call defaulting webhook logic also here, because defaulting through webhook might be
	// incomplete in case of generateName usage
	serviceInstance.Default()

	// Retrieve referenced space
	var space cfv1alpha1.GenericSpace
	var spaceSecretName types.NamespacedName
	if serviceInstance.Spec.SpaceName != "" {
		// Namespaced Space
		spaceName := types.NamespacedName{
			Namespace: serviceInstance.Namespace,
			Name:      serviceInstance.Spec.SpaceName,
		}
		space = &cfv1alpha1.Space{}
		if err := r.Get(ctx, spaceName, space); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed to get Space, name: %s", serviceInstance.Spec.SpaceName)
		}
		spaceSecretName = types.NamespacedName{
			Namespace: serviceInstance.Namespace,
			Name:      space.GetSpec().AuthSecretName,
		}
	} else if serviceInstance.Spec.ClusterSpaceName != "" {
		// Cluster Space
		clusterSpaceName := types.NamespacedName{
			Name: serviceInstance.Spec.ClusterSpaceName,
		}
		space = &cfv1alpha1.ClusterSpace{}
		if err := r.Get(ctx, clusterSpaceName, space); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed to get ClusterSpace, name: %s", serviceInstance.Spec.ClusterSpaceName)
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

	// Retrieve referenced space secret containing credentials for accessing CF
	spaceSecret := &corev1.Secret{}
	if err := r.Get(ctx, spaceSecretName, spaceSecret); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to get Secret containing space credentials, secret name: %s", spaceSecretName)
	}

	// Require readiness of space unless in deletion case
	if serviceBinding.DeletionTimestamp.IsZero() {
		if !space.IsReady() {
			serviceBinding.SetReadyCondition(cfv1alpha1.ConditionUnknown, serviceBindingReadyConditionReasonSpaceNotReady,
				fmt.Sprintf("Referenced %s is not ready, name: %s", space.GetKind(), space.GetName()))
			// TODO: apply some increasing period, depending on the age of the last update
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		if spaceGuid == "" {
			return ctrl.Result{}, fmt.Errorf("unexpected error; unable to find guid on ready %s: name: %s", space.GetKind(), space.GetName())
		}
	}

	// Build client for accessing CF
	var spaceClient facade.SpaceClient
	if spaceGuid != "" {
		spaceClient, err = r.ClientBuilder(spaceGuid, string(spaceSecret.Data["url"]), string(spaceSecret.Data["username"]), string(spaceSecret.Data["password"]), r.Config)
		if err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed to build the client from secret %s", spaceSecretName)
		}
	}

	// Retrieve service binding from CF
	var cfbinding *facade.Binding
	bindingOpts := map[string]string{"name": "", "owner": string(serviceBinding.UID)}
	if spaceClient != nil {
		log.V(1).Info("Retrieving binding by owner")
		cfbinding, err = spaceClient.GetBinding(ctx, bindingOpts)
		if err != nil {
			return ctrl.Result{}, err
		}

		orphan, exists := serviceBinding.Annotations[cfv1alpha1.AnnotationAdoptCFResources]
		if exists && cfbinding == nil && orphan == "adopt" {
			// find orphaned binding by name
			bindingOpts["name"] = serviceBinding.Name
			log.V(1).Info("Retrieving binding by name")
			cfbinding, err = spaceClient.GetBinding(ctx, bindingOpts)
			if err != nil {
				return ctrl.Result{}, err
			}

			if cfbinding != nil && cfbinding.State == facade.BindingStateReady {
				return r.adoptBinding(ctx, spaceClient, serviceBinding, cfbinding)
			} else if cfbinding != nil && cfbinding.State != facade.BindingStateReady {
				return ctrl.Result{}, fmt.Errorf("orphaned binding is not ready to be adopted")
			}
		}
	}

	if serviceBinding.DeletionTimestamp.IsZero() {
		return r.handleCreationOrUpdate(ctx, spaceClient, serviceBinding, cfbinding,
			bindingOpts, serviceInstance)
	} else {
		return r.handleDeletion(ctx, spaceClient, serviceBinding, cfbinding,
			&skipStatusUpdate)
	}
}

func (r *ServiceBindingReconciler) adoptBinding(ctx context.Context,
	spaceClient facade.SpaceClient, serviceBinding *cfv1alpha1.ServiceBinding,
	cfbinding *facade.Binding) (result ctrl.Result, err error) {

	// prepare parameters to adopt orphaned binding
	var parameterObjects []map[string]interface{}
	paramMap := make(map[string]interface{})
	paramMap["parameter-hash"] = cfbinding.ParameterHash
	paramMap["owner"] = cfbinding.Owner
	parameterObjects = append(parameterObjects, paramMap)
	parameters, err := mergeObjects(parameterObjects...)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to unmarshal/merge parameters")
	}

	log := ctrl.LoggerFrom(ctx)

	// update orphaned binding in CF
	log.V(1).Info("Updating binding")
	if err := spaceClient.UpdateBinding(
		ctx,
		cfbinding.Guid,
		cfbinding.Owner,
		serviceBinding.Generation,
		parameters,
	); err != nil {
		return ctrl.Result{}, err
	}
	serviceBinding.Status.LastModifiedAt = &[]metav1.Time{metav1.Now()}[0]
	serviceBinding.SetReadyCondition(cfv1alpha1.ConditionUnknown, string(cfbinding.State),
		cfbinding.StateDescription)

	return ctrl.Result{Requeue: true}, nil // requeue inmediatly after update
}

func (r *ServiceBindingReconciler) handleCreationOrUpdate(
	ctx context.Context, spaceClient facade.SpaceClient,
	serviceBinding *cfv1alpha1.ServiceBinding, cfbinding *facade.Binding,
	bindingOpts map[string]string,
	serviceInstance *cfv1alpha1.ServiceInstance) (result ctrl.Result, err error) {

	// Add finalizer
	if !containsString(serviceBinding.Finalizers, serviceBindingFinalizer) {
		controllerutil.AddFinalizer(serviceBinding, serviceBindingFinalizer)
		if err := r.Update(ctx, serviceBinding); err != nil {
			return ctrl.Result{}, err
		}
	}

	log := ctrl.LoggerFrom(ctx)

	spec := &serviceBinding.Spec
	status := &serviceBinding.Status

	if !serviceInstance.IsReady() {
		serviceBinding.SetReadyCondition(cfv1alpha1.ConditionUnknown, serviceBindingReadyConditionReasonServiceInstanceNotReady,
			fmt.Sprintf("Referenced ServiceInstance is not ready, name: %s", serviceInstance.Name))
		// TODO: apply some increasing period, depending on the age of the last update
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Process parameters of service binding
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
			Namespace: serviceBinding.Namespace,
			Name:      pf.SecretKeyRef.Name,
		}
		secret := &corev1.Secret{}
		if err := r.Get(ctx, secretName, secret); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed to get Secret containing service binding parameters, secret name: %s", secretName)
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

	status.ServiceBindingDigest = facade.ObjectHash(map[string]interface{}{"generation": serviceBinding.Generation, "parameters": parameters})

	recreateOnParameterChange := serviceBinding.Annotations["service-operator.cf.cs.sap.com/rotate-on-parameter-change"] == "true"
	recreateOnInstanceChange := serviceBinding.Annotations["service-operator.cf.cs.sap.com/rotate-on-instance-change"] == "true"
	inRecreation := false

	if cfbinding == nil {
		log.V(1).Info("Creating binding")
		if err := spaceClient.CreateBinding(
			ctx,
			spec.Name,
			serviceInstance.Status.ServiceInstanceGuid,
			parameters,
			string(serviceBinding.UID), // owner
			serviceBinding.Generation,
		); err != nil {
			return ctrl.Result{}, err
		}
		status.LastModifiedAt = &[]metav1.Time{metav1.Now()}[0]
	} else {
		if cfbinding.State == facade.BindingStateDeleting {
			// This is the re-creation case; nothing to, we just wait until it is gone
		} else if (recreateOnParameterChange && cfbinding.ParameterHash != facade.ObjectHash(parameters)) ||
			(recreateOnInstanceChange && status.ServiceInstanceDigest != serviceInstance.Status.ServiceInstanceDigest) ||
			cfbinding.State == facade.BindingStateCreatedFailed || cfbinding.State == facade.BindingStateDeleteFailed {
			// CF does not support binding updates except metadata => re-create instead
			log.V(1).Info("Deleting binding for later re-creation")
			if err := spaceClient.DeleteBinding(ctx, cfbinding.Guid, cfbinding.Owner); err != nil {
				return ctrl.Result{}, err
			}
			status.LastModifiedAt = &[]metav1.Time{metav1.Now()}[0]
			inRecreation = true
			// Clear binding, so it will be re-read below
			cfbinding = nil
		} else if cfbinding.Generation < serviceBinding.Generation {
			// metadata updates (such as updating the generation here) are possible with service bindings
			log.V(1).Info("Updating binding")
			if err := spaceClient.UpdateBinding(
				ctx,
				cfbinding.Guid,
				cfbinding.Owner,
				serviceBinding.Generation,
				nil,
			); err != nil {
				return ctrl.Result{}, err
			}
			status.LastModifiedAt = &[]metav1.Time{metav1.Now()}[0]
			// Clear binding, so it will be re-read below
			cfbinding = nil
		}
	}

	status.ServiceInstanceDigest = serviceInstance.Status.ServiceInstanceDigest

	if cfbinding == nil {
		// Re-read binding from CF by UID (happens in case it was created or updated above)
		log.V(1).Info("Retrieving binding")
		cfbinding, err = spaceClient.GetBinding(ctx, bindingOpts)
		if err != nil {
			return ctrl.Result{}, err
		}
		if cfbinding == nil {
			if inRecreation {
				// binding is already gone but this is the recreation case
				// just requeue immediately and check recreation in next reconcile
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{},
				fmt.Errorf("unexpected error; binding not found in CF although it should exist")
		}
	}

	// Update status
	status.SpaceGuid = serviceInstance.Status.SpaceGuid
	status.ServiceInstanceGuid = serviceInstance.Status.ServiceInstanceGuid
	status.ServiceBindingGuid = cfbinding.Guid
	switch cfbinding.State {
	case facade.BindingStateReady:
		serviceBinding.SetReadyCondition(cfv1alpha1.ConditionTrue, string(cfbinding.State), cfbinding.StateDescription)
		withMetadata := r.EnableBindingMetadata
		if serviceBinding.Annotations["service-operator.cf.cs.sap.com/with-sap-binding-metadata"] == "true" {
			withMetadata = true
		} else if serviceBinding.Annotations["service-operator.cf.cs.sap.com/with-sap-binding-metadata"] == "false" {
			withMetadata = false
		}

		err = spaceClient.FillBindingDetails(ctx, cfbinding)
		if err != nil {
			// TODO: implement error handling
			return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
		}

		err = r.storeBindingSecret(ctx, serviceInstance, serviceBinding, cfbinding.Credentials, spec.SecretName, spec.SecretKey, withMetadata)
		if err != nil {
			// TODO: implement error handling
			return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
		}
		// TODO: apply some increasing period, depending on the age of the last update
		return getPollingInterval(serviceBinding.GetAnnotations(), "10m", cfv1alpha1.AnnotationPollingIntervalReady), nil
	case facade.BindingStateCreatedFailed, facade.BindingStateDeleteFailed:
		serviceBinding.SetReadyCondition(cfv1alpha1.ConditionFalse, string(cfbinding.State), cfbinding.StateDescription)
		// TODO: apply some increasing period, depending on the age of the last update
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	default:
		// Processing case
		serviceBinding.SetReadyCondition(cfv1alpha1.ConditionUnknown, string(cfbinding.State), cfbinding.StateDescription)
		// TODO: apply some increasing period, depending on the age of the last update
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
}

func (r *ServiceBindingReconciler) handleDeletion(
	ctx context.Context, spaceClient facade.SpaceClient,
	serviceBinding *cfv1alpha1.ServiceBinding, cfbinding *facade.Binding,
	skipStatusUpdate *bool) (result ctrl.Result, err error) {

	// Still some foreign finalizers?
	if len(removeString(serviceBinding.Finalizers, serviceBindingFinalizer)) > 0 {
		serviceBinding.SetReadyCondition(cfv1alpha1.ConditionUnknown,
			serviceBindingReadyConditionReasonDeletionBlocked,
			"Deletion blocked due to foreign finalizers")
		// TODO: apply some increasing period, depending on the age of the last update
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Delete associated secrets for service binding
	secretName := serviceBinding.Spec.SecretName
	exists, deleting, err := r.existsCredentialsSecret(ctx,
		types.NamespacedName{Namespace: serviceBinding.Namespace, Name: secretName})
	if err != nil {
		return ctrl.Result{}, err
	}
	if exists {
		if !deleting {
			if err := r.deleteBindingSecret(ctx, serviceBinding.Namespace, secretName); err != nil {
				return ctrl.Result{}, err
			}
		}

		serviceBinding.SetReadyCondition(cfv1alpha1.ConditionUnknown,
			serviceBindingReadyConditionReasonDeletionBlocked,
			"Waiting for deletion of binding secret")

		// TODO: apply some increasing period, depending on the age of the last update
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// CF binding already gone?
	if cfbinding == nil {
		// Remove finalizers from service binding
		if containsString(serviceBinding.Finalizers, serviceBindingFinalizer) {
			controllerutil.RemoveFinalizer(serviceBinding, serviceBindingFinalizer)
			if err := r.Update(ctx, serviceBinding); err != nil {
				return ctrl.Result{}, err
			}
		}

		// skip status update, since custom resource will anyway be deleted timely by API server
		// this will suppress unnecessary ugly 409'ish error messages in the logs
		// (might occur when API server deletes resource during subsequent reconcile)
		*skipStatusUpdate = true

		return ctrl.Result{}, nil
	}

	if cfbinding.State != facade.BindingStateDeleting {
		// Actual deletion of service binding in CF
		log := ctrl.LoggerFrom(ctx)
		log.V(1).Info("Deleting binding")
		if err := spaceClient.DeleteBinding(ctx, cfbinding.Guid, cfbinding.Owner); err != nil {
			return ctrl.Result{}, err
		}
		serviceBinding.Status.LastModifiedAt = &[]metav1.Time{metav1.Now()}[0]
		cfbinding.State = facade.BindingStateUnknown
		cfbinding.StateDescription = "Deletion triggered."
	}

	serviceBinding.SetReadyCondition(cfv1alpha1.ConditionUnknown,
		string(cfbinding.State),
		cfbinding.StateDescription)

	// TODO: apply some increasing period, depending on the age of the last update
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *ServiceBindingReconciler) existsCredentialsSecret(ctx context.Context, secretName types.NamespacedName) (bool, bool, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, secretName, secret); err != nil {
		if err := client.IgnoreNotFound(err); err == nil {
			return false, false, nil
		}
		return false, false, errors.Wrap(err, "failed to read binding secret")
	}
	return true, !secret.DeletionTimestamp.IsZero(), nil
}

func (r *ServiceBindingReconciler) storeBindingSecret(ctx context.Context, serviceInstance *cfv1alpha1.ServiceInstance, serviceBinding *cfv1alpha1.ServiceBinding, credentials map[string]interface{}, secretName string, secretKey string, withMetadata bool) error {
	data, err := binding.NewBinding(serviceInstance, serviceBinding, credentials).SecretData(secretKey, withMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to build binding secret")
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: serviceBinding.Namespace, Name: secretName}, secret); err != nil {
		if err := client.IgnoreNotFound(err); err != nil {
			return errors.Wrap(err, "failed to read binding secret")
		}
		secret.Namespace = serviceBinding.Namespace
		secret.Name = secretName
		if err := controllerutil.SetControllerReference(serviceBinding, secret, r.Scheme); err != nil {
			return errors.Wrap(err, "failed to create binding secret")
		}
		secret.Labels = map[string]string{cfv1alpha1.LabelKeyServiceBinding: serviceBinding.Name}
		secret.Data = data
		if err := r.Create(ctx, secret); err != nil {
			return errors.Wrap(err, "failed to create binding secret")
		}
	} else {
		if err := controllerutil.SetControllerReference(serviceBinding, secret, r.Scheme); err != nil {
			return errors.Wrap(err, "failed to update binding secret")
		}
		secret.Labels = map[string]string{cfv1alpha1.LabelKeyServiceBinding: serviceBinding.Name}
		secret.Data = data
		// TODO: should we suppress idempotent secret updates ?
		if err := r.Update(ctx, secret); err != nil {
			return errors.Wrap(err, "failed to update binding secret")
		}
	}

	secretList := &corev1.SecretList{}
	if err := client.NewNamespacedClient(r.Client, serviceBinding.Namespace).List(ctx, secretList, client.MatchingLabels{cfv1alpha1.LabelKeyServiceBinding: serviceBinding.Name}); err != nil {
		return errors.Wrap(err, "failed to retrieve dependent secrets")
	}
	for _, secret := range secretList.Items {
		if secret.Name != secretName {
			if err := r.deleteBindingSecret(ctx, secret.Namespace, secret.Name); err != nil {
				return errors.Wrap(err, "failed to delete obsolete secret")
			}
		}
	}

	return nil
}

func (r *ServiceBindingReconciler) deleteBindingSecret(ctx context.Context, secretNamespace string, secretName string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: secretNamespace,
			Name:      secretName,
		},
	}
	return client.IgnoreNotFound(r.Delete(ctx, secret, &client.DeleteOptions{PropagationPolicy: &[]metav1.DeletionPropagation{metav1.DeletePropagationForeground}[0]}))
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cfv1alpha1.ServiceBinding{}).
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{})).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		Complete(r)
}
