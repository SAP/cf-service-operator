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
	"github.com/sap/cf-service-operator/internal/config"
	"github.com/sap/cf-service-operator/internal/facade"
)

const (
	spaceFinalizer = "cf.cs.sap.com/service-operator"
)

const (
	spaceReadyConditionReasonNew             = "FirstSeen"
	spaceReadyConditionReasonError           = "Error"
	spaceReadyConditionReasonSuccess         = "Success"
	spaceReadyConditionReasonDeletionBlocked = "DeletionBlocked"
	spaceReadyConditionDeleting              = "Deleting"
)

// SpaceReconciler reconciles a (Cluster)Space object
type SpaceReconciler struct {
	Kind string
	client.Client
	Scheme                   *runtime.Scheme
	ClusterResourceNamespace string
	ClientBuilder            facade.OrganizationClientBuilder
	HealthCheckerBuilder     facade.SpaceHealthCheckerBuilder
	Config                   *config.Config
}

// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=clusterspaces,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=clusterspaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=clusterspaces/finalizers,verbs=update
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=spaces,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=spaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=spaces/finalizers,verbs=update
// +kubebuilder:rbac:groups=cf.cs.sap.com,resources=serviceinstances,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;update

func (r *SpaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(2).Info("Running reconcile")

	// Retrieve target (cluster) space
	space, err := r.newSpace()
	if err != nil {
		log.Error(err, "unrecognised space type; not retrying")
		return ctrl.Result{}, nil
	}
	if err := r.Get(ctx, req.NamespacedName, space); err != nil {
		if err := client.IgnoreNotFound(err); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "unexpected get error")
		}
		log.V(1).Info("Not found; ignoring")
		return ctrl.Result{}, nil
	}

	// Call defaulting webhook logic also here, because defaulting through webhook might be
	// incomplete in case of generateName usage
	space.Default()

	spec := space.GetSpec()
	status := space.GetStatus()
	status.ObservedGeneration = space.GetGeneration()
	status.LastReconciledAt = &[]metav1.Time{metav1.Now()}[0]

	// Always attempt to update the status at the end of this reconciliation
	skipStatusUpdate := false
	defer func() {
		if skipStatusUpdate {
			return
		}

		if err != nil {
			space.SetReadyCondition(cfv1alpha1.ConditionFalse, spaceReadyConditionReasonError, err.Error())
		}

		// update status of space CR
		if updateErr := r.Status().Update(ctx, space); updateErr != nil {
			err = utilerrors.NewAggregate([]error{err, updateErr})
			result = ctrl.Result{}
		}
	}()

	// TODO: check comment
	// Set first status (no need to requeue, because status update causes another reconciliation)
	if ready := space.GetReadyCondition(); ready == nil {
		space.SetReadyCondition(cfv1alpha1.ConditionUnknown, spaceReadyConditionReasonNew, "First seen")
		return ctrl.Result{Requeue: true}, nil
	}

	// Retrieve referenced space secret containing credentials for accessing CF
	secretName := types.NamespacedName{Name: spec.AuthSecretName}
	if space.IsNamespaced() {
		secretName.Namespace = space.GetNamespace()
	} else {
		secretName.Namespace = r.ClusterResourceNamespace
	}
	secret := &corev1.Secret{}
	if err := r.Get(ctx, secretName, secret); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to get Secret containing space credentials, secret name: %s", secretName)
	}

	// Retrieve space from CF
	var cfspace *facade.Space
	var orgClient facade.OrganizationClient
	if spec.Guid == "" {
		// Build client for accessing CF
		url := string(secret.Data["url"])
		username := string(secret.Data["org_username"])
		password := string(secret.Data["org_password"])
		if username == "" || password == "" {
			username = string(secret.Data["username"])
			password = string(secret.Data["password"])
		}

		orgClient, err = r.ClientBuilder(spec.OrganizationName, url, username, password, r.Config)
		if err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed to build the client from secret %s", secretName)
		}

		// Retrieve space from CF
		log.V(1).Info("Retrieving space")
		cfspace, err = orgClient.GetSpace(ctx, string(space.GetUID()))
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if space.GetDeletionTimestamp().IsZero() {
		return r.handleCreationOrUpdate(ctx, space, cfspace, secret, orgClient)
	} else {
		return r.handleDeletion(ctx, space, cfspace, secret, orgClient, &skipStatusUpdate)
	}
}

func (r *SpaceReconciler) handleCreationOrUpdate(ctx context.Context,
	space cfv1alpha1.GenericSpace, cfspace *facade.Space, secret *corev1.Secret,
	orgClient facade.OrganizationClient) (result ctrl.Result, err error) {

	// Add finalizers on space and secret
	if !containsString(space.GetFinalizers(), spaceFinalizer) {
		controllerutil.AddFinalizer(space, spaceFinalizer)
		if err := r.Update(ctx, space); err != nil {
			return ctrl.Result{}, err
		}
	}
	if !containsString(secret.GetFinalizers(), spaceFinalizer) {
		controllerutil.AddFinalizer(secret, spaceFinalizer)
		if err := r.Update(ctx, secret); err != nil {
			return ctrl.Result{}, err
		}
	}

	log := ctrl.LoggerFrom(ctx)

	spec := space.GetSpec()
	status := space.GetStatus()
	if spec.Guid != "" {
		status.SpaceGuid = spec.Guid
	} else {
		if cfspace == nil {
			// creation case
			log.V(1).Info("Creating space")
			if err := orgClient.CreateSpace(
				ctx,
				spec.Name,
				string(space.GetUID()),
				space.GetGeneration(),
			); err != nil {
				return ctrl.Result{}, err
			}
			status.LastModifiedAt = &[]metav1.Time{metav1.Now()}[0]
			// cfspace is still nil here intentionally
		} else if cfspace.Generation < space.GetGeneration() {
			// update case
			log.V(1).Info("Updating space")
			updateName := spec.Name
			if updateName == cfspace.Name {
				updateName = ""
			}
			if err := orgClient.UpdateSpace(
				ctx,
				cfspace.Guid,
				updateName,
				cfspace.Owner,
				space.GetGeneration(),
			); err != nil {
				return ctrl.Result{}, err
			}
			status.LastModifiedAt = &[]metav1.Time{metav1.Now()}[0]
			// Clear instance, so it will be re-read below
			cfspace = nil
		}

		// Re-retrieve space from CF (happens only if it was created or updated above)
		if cfspace == nil {
			log.V(1).Info("Retrieving space")
			cfspace, err = orgClient.GetSpace(ctx, string(space.GetUID()))
			if err != nil {
				return ctrl.Result{}, err
			}
			if cfspace == nil {
				return ctrl.Result{}, fmt.Errorf("unexpected error; space not found in cloud foundry although it should exist")
			}
		}

		// TODO: the following is not very clean; if the user referenced by the secret changes, we leave the previous one orphaned;
		// maybe we should clean it up somehow (but how ... what if that previous user has already been taken over by another manager, such as CAM?)
		log.V(1).Info("Adding developer")
		if err := orgClient.AddDeveloper(ctx, cfspace.Guid, string(secret.Data["username"])); err != nil {
			return ctrl.Result{}, err
		}
		status.LastModifiedAt = &[]metav1.Time{metav1.Now()}[0]
		status.SpaceGuid = cfspace.Guid
	}

	// Setup health checker
	url := string(secret.Data["url"])
	username := string(secret.Data["username"])
	password := string(secret.Data["password"])
	checker, err := r.HealthCheckerBuilder(status.SpaceGuid, url, username, password, r.Config)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to build the healthchecker from secret %s", secret.Name)
	}

	// Check space health
	log.V(1).Info("Checking space")
	if err := checker.Check(ctx, cfspace.Owner); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "healthcheck failed")
	}
	log.V(1).Info("Healthcheck successful")

	// Space is ready
	space.SetReadyCondition(cfv1alpha1.ConditionTrue, spaceReadyConditionReasonSuccess, "Success")
	return getPollingInterval(space.GetAnnotations(), "60s", cfv1alpha1.AnnotationPollingIntervalReady), nil
}

func (r *SpaceReconciler) handleDeletion(ctx context.Context, space cfv1alpha1.GenericSpace,
	cfspace *facade.Space, secret *corev1.Secret,
	orgClient facade.OrganizationClient, skipStatusUpdate *bool) (result ctrl.Result, err error) {

	// Find depending service instances (directly from K8s, i.e. bypassing cache)
	// This is not very efficient. Do this only in this deletion case when it is needed.
	serviceInstanceList := &cfv1alpha1.ServiceInstanceList{}
	if space.IsNamespaced() {
		// Space
		if err := client.NewNamespacedClient(r.Client, space.GetNamespace()).List(
			ctx,
			serviceInstanceList,
			client.MatchingLabels{cfv1alpha1.LabelKeySpace: space.GetName()},
		); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to list depending service instances")
		}
	} else {
		// ClusterSpace
		if err := r.Client.List(
			ctx,
			serviceInstanceList,
			client.MatchingLabels{cfv1alpha1.LabelKeyClusterSpace: space.GetName()},
		); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to list depending service instances")
		}
	}

	if len(serviceInstanceList.Items) > 0 {
		space.SetReadyCondition(cfv1alpha1.ConditionUnknown, spaceReadyConditionReasonDeletionBlocked, "Waiting for deletion of depending service instances")
		// TODO: apply some increasing period, depending on the age of the last update
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	} else if len(removeString(space.GetFinalizers(), spaceFinalizer)) > 0 {
		space.SetReadyCondition(cfv1alpha1.ConditionUnknown, spaceReadyConditionReasonDeletionBlocked, "Deletion blocked due to foreign finalizers")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		// TODO: apply some increasing period, depending on the age of the last update
	} else if cfspace == nil {
		// Remove finalizers from secret and space
		if containsString(secret.GetFinalizers(), spaceFinalizer) {
			controllerutil.RemoveFinalizer(secret, spaceFinalizer)
			if err := r.Update(ctx, secret); err != nil {
				return ctrl.Result{}, err
			}
		}
		if containsString(space.GetFinalizers(), spaceFinalizer) {
			controllerutil.RemoveFinalizer(space, spaceFinalizer)
			if err := r.Update(ctx, space); err != nil {
				return ctrl.Result{}, err
			}
		}

		// skip status update, since the instance will anyway deleted timely by the API server
		// this will suppress unnecessary ugly 409'ish error messages in the logs
		// (occurring in the case that API server would delete the resource in the course of the subsequent reconciliation)
		*skipStatusUpdate = true
		return ctrl.Result{}, nil
	} else {
		// Actual deletion of space
		log := ctrl.LoggerFrom(ctx)
		log.V(1).Info("Deleting space")
		if err := orgClient.DeleteSpace(ctx, cfspace.Guid, cfspace.Owner); err != nil {
			return ctrl.Result{}, err
		}
		status := space.GetStatus()
		status.LastModifiedAt = &[]metav1.Time{metav1.Now()}[0]
		space.SetReadyCondition(cfv1alpha1.ConditionUnknown, spaceReadyConditionDeleting, "Deletion triggered")
		// TODO: apply some increasing period, depending on the age of the last update
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
}

func (r *SpaceReconciler) newSpace() (cfv1alpha1.GenericSpace, error) {
	spaceGVK := cfv1alpha1.GroupVersion.WithKind(r.Kind)
	obj, err := r.Scheme.New(spaceGVK)
	if err != nil {
		return nil, err
	}
	space, ok := obj.(cfv1alpha1.GenericSpace)
	if !ok {
		return nil, fmt.Errorf("not a space object: %s", spaceGVK.String())
	}
	return space, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SpaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	spaceType, err := r.newSpace()
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(spaceType).
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{})).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		Complete(r)
}
