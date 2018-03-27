/*
 * INTEL CONFIDENTIAL
 * Copyright (2017) Intel Corporation.
 *
 * The source code contained or described herein and all documents related to the source code ("Material")
 * are owned by Intel Corporation or its suppliers or licensors. Title to the Material remains with
 * Intel Corporation or its suppliers and licensors. The Material may contain trade secrets and proprietary
 * and confidential information of Intel Corporation and its suppliers and licensors, and is protected by
 * worldwide copyright and trade secret laws and treaty provisions. No part of the Material may be used,
 * copied, reproduced, modified, published, uploaded, posted, transmitted, distributed, or disclosed in
 * any way without Intel/'s prior express written permission.
 * No license under any patent, copyright, trade secret or other intellectual property right is granted
 * to or conferred upon you by disclosure or delivery of the Materials, either expressly, by implication,
 * inducement, estoppel or otherwise. Any license under such intellectual property rights must be express
 * and approved by Intel in writing.
 * Unless otherwise agreed by Intel in writing, you may not remove or alter this notice or any other
 * notice embedded in Materials by Intel or Intel's suppliers or licensors in any way.
 */

package cloudConnector

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.impcloud.net/Responsive-Retail-Inventory/cloud-connector-service/app/config"
)

func GenerateWebhook(testServerURL string, auth bool) Webhook {
	n := Webhook{
		URL:     testServerURL + "/callwebhook",
		Payload: []byte{},
	}
	if auth {
		n.Auth = Auth{AuthType: "oauth2", Endpoint: testServerURL + "/oauth", Data: "testname:testpassword"}
	}

	return n
}

func TestMain(m *testing.M) {

	_ = config.InitConfig()

	os.Exit(m.Run())

}

// nolint: dupl
func TestOAuth2PostWebhookOk(t *testing.T) {
	testJdaMockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != "POST" {
			t.Errorf("Expected 'POST' request, received '%s", request.Method)
		}

		escapedPath := request.URL.EscapedPath()
		if escapedPath == "/oauth" {
			data := make(map[string]interface{})
			data["access_token"] = "eyJhbGci0iJSUzI1NiJ9.eyJeHAi0jE0NjUzMzU.eju3894"
			data["token_type"] = "bearer"
			data["expires_in"] = 3599
			data["scope"] = "access"
			data["jti"] = "aceable12-1709-4aae-a289-df8b88c84c95"
			data["x-tenant-id"] = "c54d9ccb-6ddd-4416-85d4-e01f565b1266"

			jsonData, _ := json.Marshal(data)
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(jsonData)
		} else if escapedPath == "/callwebhook" {
			data := make(map[string]interface{})
			data["timestamp"] = 14903136768
			data["skus"] = `["MS122-38"]`
			data["ruleId"] = "SomeRuleId-1234"
			data["notificationId"] = "Out-of-stock-ShoesId123"
			data["stockCount"] = 0.0
			data["sellCount"] = 0.0

			jsonData, _ := json.Marshal(data)
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(jsonData)
		} else {
			t.Errorf("Expected request to '/oauth' or 'notification', received %s", escapedPath)
		}
	}))

	defer testJdaMockServer.Close()

	webHook := GenerateWebhook(testJdaMockServer.URL, true)
	data := []byte(`{ }`)

	webHook.URL = testJdaMockServer.URL + "/callwebhook"
	webHook.Auth.AuthType = "OAuth2"
	webHook.Auth.Endpoint = testJdaMockServer.URL + "/oauth"
	webHook.Auth.Data = "this is a test"
	webHook.Payload = data

	err := ProcessWebhook(webHook, "")
	if err != nil {
		t.Error(err)
	}
}

func TestOAuth2PostWebhookForbidden(t *testing.T) {
	testJdaMockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != "POST" {
			t.Errorf("Expected 'POST' request, received '%s", request.Method)
		}

		escapedPath := request.URL.EscapedPath()
		if escapedPath == "/oauth" {
			// authentication failed
			writer.WriteHeader(http.StatusUnauthorized)
		} else if escapedPath == "/callwebhook" {
			writer.WriteHeader(http.StatusForbidden)
		} else {
			t.Errorf("Expected request to '/oauth' or 'notification', received %s", escapedPath)
		}
	}))

	defer testJdaMockServer.Close()

	webHook := GenerateWebhook(testJdaMockServer.URL, true)
	data := []byte(`{ }`)

	webHook.URL = testJdaMockServer.URL + "/callwebhook"
	webHook.Auth.AuthType = "OAuth2"
	webHook.Auth.Endpoint = testJdaMockServer.URL + "/oauth"
	webHook.Auth.Data = "this is a test"
	webHook.Payload = data

	// expecting unauthorized
	err := ProcessWebhook(webHook, "")
	if err == nil {
		t.Error("Expected authentication error, not 200 status")
	}
}

// nolint: dupl
func TestOAuth2PostWebhookFailNotification(t *testing.T) {
	testJdaMockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != "POST" {
			t.Errorf("Expected 'POST' request, received '%s", request.Method)
		}

		escapedPath := request.URL.EscapedPath()
		if escapedPath == "/oauth" {
			data := make(map[string]interface{})
			data["access_token"] = "eyJhbGci0iJSUzI1NiJ9.eyJeHAi0jE0NjUzMzU.eju3894"
			data["token_type"] = "bearer"
			data["expires_in"] = 3599
			data["scope"] = "access"
			data["jti"] = "aceable12-1709-4aae-a289-df8b88c84c95"
			data["x-tenant-id"] = "c54d9ccb-6ddd-4416-85d4-e01f565b1266"

			jsonData, _ := json.Marshal(data)
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(jsonData)
		} else if escapedPath == "/callwebhook" {
			writer.WriteHeader(http.StatusForbidden)
		} else {
			t.Errorf("Expected request to '/oauth' or 'notification', received %s", escapedPath)
		}
	}))

	defer testJdaMockServer.Close()

	webHook := GenerateWebhook(testJdaMockServer.URL, true)
	data := []byte(`{ }`)

	webHook.URL = testJdaMockServer.URL + "/callwebhook"
	webHook.Auth.AuthType = "OAuth2"
	webHook.Auth.Endpoint = testJdaMockServer.URL + "/oauth"
	webHook.Auth.Data = "this is a test"
	webHook.Payload = data

	// expecting unauthorized
	err := ProcessWebhook(webHook, "")
	if err == nil {
		t.Error("Expected POST notification error, not 200 status")
	}
}

// nolint: dupl
func TestOAuth2PostWebhookNoAuthenticationOK(t *testing.T) {
	testMockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != "POST" {
			t.Errorf("Expected 'POST' request, received '%s", request.Method)
		}

		escapedPath := request.URL.EscapedPath()

		expectedHeaderItem := "application/x-www-form-urlencoded"

		if request.Header["Content-Type"][0] != expectedHeaderItem {
			t.Errorf("Expected request header content to be %s, received %s", expectedHeaderItem, request.Header["Content-Type"][0])
		}

		if escapedPath == "/callwebhook" {
			data := make(map[string]interface{})
			jsonData, _ := json.Marshal(data)
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(jsonData)
		} else {
			t.Errorf("Expected request to '/oauth' or 'notification', received %s", escapedPath)
		}
	}))

	defer testMockServer.Close()

	webHook := GenerateWebhook(testMockServer.URL, false)
	data := []byte(`{ }`)

	webHook.Method = "POST"
	webHook.Header = http.Header{}
	webHook.Header["Content-Type"] = []string{"application/x-www-form-urlencoded"}
	webHook.URL = testMockServer.URL + "/callwebhook"
	webHook.Payload = data

	err := ProcessWebhook(webHook, "")
	if err != nil {
		t.Error(err)
	}
}

func TestOAuth2PostWebhookNoAuthenticationForbidden(t *testing.T) {
	testJdaMockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != "POST" {
			t.Errorf("Expected 'POST' request, received '%s", request.Method)
		}

		escapedPath := request.URL.EscapedPath()
		if escapedPath == "/callwebhook" {
			writer.WriteHeader(http.StatusForbidden)
		} else {
			t.Errorf("Expected request to '/oauth' or 'notification', received %s", escapedPath)
		}
	}))

	defer testJdaMockServer.Close()

	webHook := GenerateWebhook(testJdaMockServer.URL, false)
	data := []byte(`{ }`)
	webHook.Method = "POST"
	webHook.URL = testJdaMockServer.URL + "/callwebhook"
	webHook.Payload = data

	err := ProcessWebhook(webHook, "")
	if err != nil {
		t.Error(err)
	}
}

func TestPostWebhookNoAuthenticationProxy(t *testing.T) {
	testURL := "testURL.com"
	webHook := GenerateWebhook(testURL, false)
	data := []byte(`{ }`)

	webHook.URL = testURL + "/callwebhook"
	webHook.Payload = data

	err := ProcessWebhook(webHook, "test.proxy")
	if err == nil {
		t.Error(err)
	}
}

func TestPostOAuth2WebhookProxy(t *testing.T) {
	testURL := "testURL.com"
	webHook := GenerateWebhook(testURL, false)
	data := []byte(`{ }`)

	webHook.URL = testURL + "/callwebhook"
	webHook.Auth.AuthType = "OAuth2"
	webHook.Auth.Endpoint = testURL + "/oauth"
	webHook.Auth.Data = "this is a test"
	webHook.Payload = data

	err := ProcessWebhook(webHook, "test.proxy")
	if err == nil {
		t.Error(err)
	}
}
