/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"encoding/json"
	"fmt"
)

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func unmarshalObject(rawObj []byte) (map[string]interface{}, error) {
	if rawObj == nil {
		return nil, nil
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(rawObj, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func mergeObjects(objs ...map[string]interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	for _, obj := range objs {
		if result == nil {
			result = obj
		} else {
			for key, value := range obj {
				if _, ok := result[key]; ok {
					return nil, fmt.Errorf("root level parameter key exists in more than one object, key: %s", key)
				}
				result[key] = value
			}
		}
	}
	return result, nil
}
