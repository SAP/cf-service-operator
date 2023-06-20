/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package cfclientext

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

// decodeBody is used to JSON decode a body
func decodeBody(resp *http.Response, out interface{}) error {
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}

// encodeBody is used to encode a request body
func encodeBody(obj interface{}) (io.Reader, error) {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(obj); err != nil {
		return nil, err
	}
	return buf, nil
}

// extract path (plus query parameters if present) from URL
func extractPathFromURL(requestURL string) (string, error) {
	url, err := url.Parse(requestURL)
	if err != nil {
		return "", err
	}
	result := url.Path
	if q := url.Query().Encode(); q != "" {
		result = result + "?" + q
	}
	return result, nil
}
