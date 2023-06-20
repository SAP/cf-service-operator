/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package facade

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

type OrganizationClient interface {
	GetSpace(owner string) (*Space, error)
	CreateSpace(name string, owner string, generation int64) error
	UpdateSpace(guid string, name string, generation int64) error
	DeleteSpace(guid string) error
	AddAuditor(guid string, username string) error
	AddDeveloper(guid string, username string) error
	AddManager(guid string, username string) error
}

type OrganizationClientBuilder func(string, string, string, string) (OrganizationClient, error)

type SpaceClient interface {
	GetInstance(owner string) (*Instance, error)
	CreateInstance(name string, servicePlanGuid string, parameters map[string]interface{}, tags []string, owner string, generation int64) error
	UpdateInstance(guid string, name string, servicePlanGuid string, parameters map[string]interface{}, tags []string, generation int64) error
	DeleteInstance(guid string) error

	GetBinding(owner string) (*Binding, error)
	CreateBinding(name string, serviceInstanceGuid string, parameters map[string]interface{}, owner string, generation int64) error
	UpdateBinding(guid string, generation int64) error
	DeleteBinding(guid string) error

	FindServicePlan(serviceOfferingName string, servicePlanName string, spaceGuid string) (string, error)
}

type SpaceClientBuilder func(string, string, string, string) (SpaceClient, error)
