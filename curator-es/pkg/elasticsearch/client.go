// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package elasticsearch

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
)

const (
	httpPrefix  = "http://"
	httpsPrefix = "https://"
)

// NewClient creates new Elasticsearch client from given <url> and <http_auth>.
// <http_auth> is in format <username>:<password> or "" if basic auth is not required.
func NewClient(url, httpAuth string) *Client {
	return &Client{
		URL:      url,
		HTTPAuth: httpAuth,
	}
}

// CatNodes queries the cat nodes API of Elasticsearch.
func (c *Client) CatNodes() ([]CatNode, error) {
	url := path.Join(c.URL, "/_cat/nodes?h=id,name&format=json&full_id=true")

	statusCode, body, err := doGet(url, c.HTTPAuth)
	if err != nil {
		return nil, err
	} else if statusCode != http.StatusOK {
		return nil, unexpectedError(url, statusCode)
	}

	response := []CatNode{}
	err = json.Unmarshal(body, &response)

	return response, err
}

// GetNodeStats queries the node stats API of Elasticsearch.
func (c *Client) GetNodeStats(name string) (*NodeStats, error) {
	url := path.Join(c.URL, fmt.Sprintf("/_nodes/%s/stats/fs", name))

	statusCode, body, err := doGet(url, c.HTTPAuth)
	if err != nil {
		return nil, err
	} else if statusCode != http.StatusOK {
		return nil, unexpectedError(url, statusCode)
	}

	response := NodeStats{}
	err = json.Unmarshal(body, &response)

	return &response, err
}

// GetIndices queries the get all indices API of Elasticsearch.
func (c *Client) GetIndices(name string) (map[string]Index, error) {
	url := path.Join(c.URL, fmt.Sprintf("/%s/_settings", name))

	statusCode, body, err := doGet(url, c.HTTPAuth)
	if err != nil {
		return nil, err
	} else if statusCode != http.StatusOK {
		return nil, unexpectedError(url, statusCode)
	}

	response := make(map[string]Index)
	err = json.Unmarshal(body, &response)

	return response, err
}

// DeleteIndex queries the delete index API of Elasticsearch.
func (c *Client) DeleteIndex(name string) error {
	url := path.Join(c.URL, "/"+name)

	statusCode, _, err := doDelete(url, c.HTTPAuth)
	if err != nil {
		return err
	} else if statusCode != http.StatusOK {
		return unexpectedError(url, statusCode)
	}

	return nil
}

func doGet(url, httpAuth string) (int, []byte, error) {
	return makeHTTPRequest("GET", url, nil, getHeaders(httpAuth), nil)
}

func doDelete(url, httpAuth string) (int, []byte, error) {
	return makeHTTPRequest("DELETE", url, nil, getHeaders(httpAuth), nil)
}

func makeHTTPRequest(method, url string, body []byte, headers map[string][]string, form map[string]string) (int, []byte, error) {
	url = getURLWithSchemaString(url)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))

	if err != nil {
		return 0, nil, err
	}
	setHeaders(req, headers)
	setForm(req, form)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, responseBody, nil
}

func getHeaders(httpAuth string) map[string][]string {
	var headers map[string][]string
	if httpAuth != "" {
		headers = make(map[string][]string)
		headers["Authorization"] = []string{"Basic " + basicAuth(httpAuth)}
	}
	return headers
}

func basicAuth(auth string) string {
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func setHeaders(request *http.Request, headers map[string][]string) {
	for header, values := range headers {
		for _, value := range values {
			request.Header.Set(header, value)
		}
	}
}

func setForm(request *http.Request, form map[string]string) {
	for key, value := range form {
		request.Form.Add(key, value)
	}
}

func getURLWithSchemaString(url string) string {
	if !strings.HasPrefix(url, httpPrefix) {
		if strings.HasPrefix(url, httpsPrefix) {
			url = strings.Replace(url, httpsPrefix, httpPrefix, 1)
		} else {
			url = httpPrefix + url
		}
	}
	return url
}

func unexpectedError(url string, statusCode int) error {
	return fmt.Errorf("Response status code from url %q is %d", url, statusCode)
}
