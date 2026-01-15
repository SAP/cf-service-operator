/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package facade

import "context"

type Space struct {
	Guid       string
	Name       string
	Owner      string
	Generation int64
}

type Instance struct {
	Guid             string
	Name             string
	ServicePlanGuid  string
	Owner            string
	Generation       int64
	ParameterHash    string
	State            InstanceState
	StateDescription string
}

type InstanceState string

const (
	InstanceStateUnknown       InstanceState = "Unknown"
	InstanceStateReady         InstanceState = "Ready"
	InstanceStateCreating      InstanceState = "Creating"
	InstanceStateCreatedFailed InstanceState = "CreateFailed"
	InstanceStateUpdating      InstanceState = "Updating"
	InstanceStateUpdateFailed  InstanceState = "UpdateFailed"
	InstanceStateDeleting      InstanceState = "Deleting"
	InstanceStateDeleteFailed  InstanceState = "DeleteFailed"
	InstanceStateDeleted       InstanceState = "Deleted"
)

type Binding struct {
	Guid             string
	Name             string
	Owner            string
	Generation       int64
	ParameterHash    string
	State            BindingState
	StateDescription string
	Credentials      map[string]interface{}
}

type BindingState string

const (
	BindingStateUnknown       BindingState = "Unknown"
	BindingStateReady         BindingState = "Ready"
	BindingStateCreating      BindingState = "Creating"
	BindingStateCreatedFailed BindingState = "CreateFailed"
	BindingStateDeleting      BindingState = "Deleting"
	BindingStateDeleteFailed  BindingState = "DeleteFailed"
	BindingStateDeleted       BindingState = "Deleted"
)

//counterfeiter:generate . OrganizationClient
type OrganizationClient interface {
	GetSpace(ctx context.Context, owner string) (*Space, error)
	CreateSpace(ctx context.Context, name string, owner string, generation int64) error
	UpdateSpace(ctx context.Context, guid string, name string, generation int64) error
	DeleteSpace(ctx context.Context, guid string) error
	AddAuditor(ctx context.Context, guid string, username string) error
	AddDeveloper(ctx context.Context, guid string, username string) error
	AddManager(ctx context.Context, guid string, username string) error
}

type OrganizationClientBuilder func(string, string, string, string) (OrganizationClient, error)

//counterfeiter:generate . SpaceClient
type SpaceClient interface {
	GetInstance(ctx context.Context, instanceOpts map[string]string) (*Instance, error)
	CreateInstance(ctx context.Context, name string, servicePlanGuid string, parameters map[string]interface{}, tags []string, owner string, generation int64) error
	UpdateInstance(ctx context.Context, guid string, name string, servicePlanGuid string, parameters map[string]interface{}, tags []string, generation int64) error
	DeleteInstance(ctx context.Context, guid string) error

	GetBinding(ctx context.Context, bindingOpts map[string]string) (*Binding, error)
	CreateBinding(ctx context.Context, name string, serviceInstanceGuid string, parameters map[string]interface{}, owner string, generation int64) error
	UpdateBinding(ctx context.Context, guid string, generation int64, parameters map[string]interface{}) error
	DeleteBinding(ctx context.Context, guid string) error

	FindServicePlan(ctx context.Context, serviceOfferingName string, servicePlanName string, spaceGuid string) (string, error)
}

type SpaceClientBuilder func(string, string, string, string) (SpaceClient, error)
