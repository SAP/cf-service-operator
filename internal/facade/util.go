/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package facade

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func ObjectHash(obj map[string]interface{}) string {
	raw, err := json.Marshal(obj)
	if err != nil {
		// TODO: should this be handled ?
		panic(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
