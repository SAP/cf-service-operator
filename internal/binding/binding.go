/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package binding

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/sap/cf-service-operator/api/v1alpha1"
)

type Binding struct {
	metadata    map[string]interface{}
	credentials map[string]interface{}
}

type BindingMetadata struct {
	MetaDataProperties   []BindingProperty `json:"metaDataProperties,omitempty"`
	CredentialProperties []BindingProperty `json:"credentialProperties,omitempty"`
}

type BindingProperty struct {
	Name       string `json:"name,omitempty"`
	SourceName string `json:"sourceName,omitempty"`
	Format     string `json:"format,omitempty"`
	Container  bool   `json:"container,omitempty"`
}

func NewBinding(serviceInstance *v1alpha1.ServiceInstance, serviceBinding *v1alpha1.ServiceBinding, credentials map[string]interface{}) *Binding {
	return &Binding{
		metadata: map[string]interface{}{
			"type":          serviceInstance.Spec.ServiceOfferingName,
			"label":         serviceInstance.Spec.ServiceOfferingName,
			"plan":          serviceInstance.Spec.ServicePlanName,
			"tags":          append([]string{serviceInstance.Spec.ServiceOfferingName}, serviceInstance.Spec.Tags...),
			"instance_name": serviceInstance.Spec.Name,
			"instance_guid": serviceInstance.Status.ServiceInstanceGuid,
		},
		credentials: credentials,
	}
}

func (binding *Binding) SecretData(secretKey string, withMetadata bool) (map[string][]byte, error) {
	metadata := BindingMetadata{}
	secretData := make(map[string][]byte)

	if withMetadata {
		secretData[".metadata"] = nil
		for k, v := range binding.metadata {
			w, format, err := encode(v)
			if err != nil {
				return nil, errors.Wrapf(err, "error encoding binding metadata key: %s", k)
			}
			secretData[k] = w
			metadata.MetaDataProperties = append(metadata.MetaDataProperties, BindingProperty{Name: k, Format: format})
		}
	}

	if secretKey == "" {
		for k, v := range binding.credentials {
			if _, ok := secretData[k]; ok {
				return nil, fmt.Errorf("conflicting binding credentials key: %s", k)
			}
			w, format, err := encode(v)
			if err != nil {
				return nil, errors.Wrapf(err, "error encoding binding credentials key: %s", k)
			}
			secretData[k] = w
			metadata.CredentialProperties = append(metadata.CredentialProperties, BindingProperty{Name: k, Format: format})
		}
	} else {
		if _, ok := secretData[secretKey]; ok {
			return nil, fmt.Errorf("conflicting binding credentials key: %s", secretKey)
		}
		w, format, err := encode(binding.credentials)
		if err != nil {
			return nil, errors.Wrap(err, "error encoding binding credentials")
		}
		secretData[secretKey] = w
		metadata.CredentialProperties = append(metadata.CredentialProperties, BindingProperty{Name: secretKey, Format: format, Container: true})
	}

	if withMetadata {
		w, _, err := encode(metadata)
		if err != nil {
			return nil, errors.Wrap(err, "error encoding binding metadata")
		}
		secretData[".metadata"] = w
	}

	return secretData, nil
}

func encode(v interface{}) ([]byte, string, error) {
	if s, ok := v.(string); ok {
		return []byte(s), "text", nil
	} else {
		w, err := json.Marshal(v)
		if err != nil {
			return nil, "", err
		}
		return w, "json", nil
	}
}
