/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.impcloud.net/RSP-Inventory-Suite/cloud-connector-service/app/cloudConnector"
	"github.impcloud.net/RSP-Inventory-Suite/cloud-connector-service/app/config"
	"github.impcloud.net/RSP-Inventory-Suite/cloud-connector-service/pkg/web"
)

type inputTest struct {
	input []byte
	code  int
}

func TestMain(m *testing.M) {

	_ = config.InitConfig(nil)

	os.Exit(m.Run())

}

func TestGetIndex(t *testing.T) {
	request, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("Unable to create new HTTP request %s", err.Error())
	}
	recorder := httptest.NewRecorder()
	cloudConnector := CloudConnector{}
	handler := web.Handler(cloudConnector.Index)
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected 200 response")
	}

	if recorder.Body.String()[1:len(recorder.Body.String())-1] != config.AppConfig.ServiceName {
		t.Errorf("Expected body to equal CloudConnector Service")
	}
}

// nolint: dupl
func TestCallWebhook(t *testing.T) {

	data := cloudConnector.Webhook{
		URL:    "http://localhost/test",
		Method: "POST",
		Auth: cloudConnector.Auth{
			AuthType: "oauth2",
			Endpoint: "http://localhost/testServerURL/oauth",
			Data:     "testname:testpassword"},
		IsAsync: true,
		Payload: []byte{}}
	mData, marshalErr := json.Marshal(data)
	if marshalErr != nil {
		t.Errorf("Unable to marshal data: %s", marshalErr.Error())
	}
	request, err := http.NewRequest("POST", "/callwebhook'", bytes.NewBuffer(mData))
	if err != nil {
		t.Errorf("Unable to create new HTTP Request: %s", err.Error())
	}

	recorder := httptest.NewRecorder()

	cloudConnector := CloudConnector{}

	handler := web.Handler(cloudConnector.CallWebhook)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected pass with 200 but returned: %d", recorder.Code)
	}
}

func TestCallWebhookwithGetRequest(t *testing.T) {
	// Define mockresponse for webhhook
	var actualResponse cloudConnector.WebhookResponse
	mockResponse := "success"
	testMockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != "GET" {
			t.Errorf("Expected 'GET' request, received '%s", request.Method)
		}
		escapedPath := request.URL.EscapedPath()
		if escapedPath == "/callwebhook" {
			jsonData, _ := json.Marshal(mockResponse)
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(jsonData)
		} else {
			t.Errorf("Expected request to '/oauth' or 'notification', received %s", escapedPath)
		}
	}))

	defer testMockServer.Close()

	data := cloudConnector.Webhook{
		URL:     testMockServer.URL + "/callwebhook",
		Method:  "GET",
		IsAsync: false,
		Payload: []byte{}}
	mData, marshalErr := json.Marshal(data)
	if marshalErr != nil {
		t.Errorf("Unable to marshal data: %s", marshalErr.Error())
	}
	request, err := http.NewRequest("GET", "/callwebhook'", bytes.NewBuffer(mData))
	if err != nil {
		t.Errorf("Unable to create new HTTP Request: %s", err.Error())
	}

	recorder := httptest.NewRecorder()

	cloudConnector := CloudConnector{}

	handler := web.Handler(cloudConnector.CallWebhook)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected pass with 200 but returned: %d", recorder.Code)
	}
	response := recorder.Result()
	body, _ := ioutil.ReadAll(response.Body)
	if len(body) < 0 {
		t.Fatal("Get request is expected to have some response back")
	}

	if err = json.Unmarshal(body, &actualResponse); err != nil {
		t.Fatalf("Error in unmarshalling response body %s", err.Error())
	}

	responseString := string(actualResponse.Body)
	//removing extra quotes in string
	responseString = responseString[1 : len(responseString)-1]
	if strings.Compare(mockResponse, responseString) != 0 {
		t.Fatalf("Actual response and expected response differs %s %s", responseString, mockResponse)
	}

}

func TestCallWebhookNotAsync(t *testing.T) {

	data := cloudConnector.Webhook{
		URL:    "http://localhost/test",
		Method: "POST",
		Auth: cloudConnector.Auth{
			AuthType: "oauth2",
			Endpoint: "http://localhost/testServerURL/oauth",
			Data:     "testname:testpassword"},
		IsAsync: false,
		Payload: []byte{}}
	mData, marshalErr := json.Marshal(data)
	if marshalErr != nil {
		t.Errorf("Unable to marshal data: %s", marshalErr.Error())
	}
	request, err := http.NewRequest("POST", "/callwebhook'", bytes.NewBuffer(mData))
	if err != nil {
		t.Errorf("Unable to create new HTTP Request: %s", err.Error())
	}

	recorder := httptest.NewRecorder()

	cloudConnector := CloudConnector{}

	handler := web.Handler(cloudConnector.CallWebhook)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Errorf("Expected to fail with 404 but returned: %d", recorder.Code)
	}
}

func TestCallWebhookWithForbiddenHTTPMethods(t *testing.T) {

	data := cloudConnector.Webhook{
		URL:    "http://localhost/test",
		Method: "PUT",
		Auth: cloudConnector.Auth{
			AuthType: "oauth2",
			Endpoint: "http://localhost/testServerURL/oauth",
			Data:     "testname:testpassword"},
		IsAsync: false,
		Payload: []byte{}}
	mData, marshalErr := json.Marshal(data)
	if marshalErr != nil {
		t.Errorf("Unable to marshal data: %s", marshalErr.Error())
	}
	request, err := http.NewRequest("POST", "/callwebhook'", bytes.NewBuffer(mData))
	if err != nil {
		t.Errorf("Unable to create new HTTP Request: %s", err.Error())
	}

	recorder := httptest.NewRecorder()

	cloudConnector := CloudConnector{}

	handler := web.Handler(cloudConnector.CallWebhook)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected to fail with 400	 but returned: %d", recorder.Code)
	}
}

func TestCallWebhookInvalidJson(t *testing.T) {

	var invalidJSONSample = []inputTest{
		{
			// Empty request body
			input: []byte(`{ }`),
			code:  400,
		},
		{
			// request has int in string map
			input: []byte(`{
				"url":"http://localhost/test",
				"method": "POST",
				"header": {
					1: ["innerThing"]
				},
				"auth" : {
					"authtype" : "oauth2",
					"endpoint" : "http://localhost/testServerURL/oauth",
					"data" :     "testname:testpassword"
				},
				"payload": "test string",
				"isasync":true
		}`),
			code: 500,
		},
		{
			// request int in string slice
			input: []byte(`{
				"url":"http://localhost/test",
				"method": "POST",
				"header": {
					"thing1": [1]
				},
				"auth" : {
					"authtype" : "oauth2",
					"endpoint" : "http://localhost/testServerURL/oauth",
					"data" :     "testname:testpassword"
				},
				"payload": "test string",
				"isasync":true
		}`),
			code: 500,
		},
		{
			// request int in string slice
			input: []byte(`{
				"url":"http://localhost/test",
				"method": "post",
				"auth" : {
					"authtype" : "oauth2",
					"endpoint" : "http://localhost/testServerURL/oauth",
					"data" :     "testname:testpassword"
				},
				"payload": "test string",
				"isasync":true
		}`),
			code: 400,
		},
	}

	cloudConnector := CloudConnector{}

	handler := web.Handler(cloudConnector.CallWebhook)

	testHandlerHelper(invalidJSONSample, handler, t)
}

//nolint: dupl
func TestCallWebhookSchemaFailed(t *testing.T) {

	var invalidJSONSample = []inputTest{
		{
			// invalid url
			input: []byte(`{
					"url": "localhost/test",
					"auth":
						{	"authtype":"oauth2",
							"endpoint":"localhost/test",
							"data":"testname:testpassword"
						},
						"isasync":true,
					"payload": ""
					}`),
			code: 400,
		},
		{
			// invalid input for data
			input: []byte(`{
				"url": "http://loocal/test",
				"auth":
					{	"authtype":"oauth2",
						"endpoint":"http://local/oauth",
						"data":123
					},
					"isasync":true,
				"payload": 123
				}`),
			code: 400,
		},
		{
			// Empty request body
			input: []byte(`{}`),
			code:  400,
		},
	}

	cloudConnector := CloudConnector{}

	handler := web.Handler(cloudConnector.CallWebhook)

	testHandlerHelper(invalidJSONSample, handler, t)
}

func TestAwsCloudCallInvalidJsonInput(t *testing.T) {

	var invalidJSONSample = []inputTest{
		{
			// Empty request body
			input: []byte(`{ }`),
			code:  400,
		},
		{
			// missing required params
			input: []byte(`{
				"payload": 123
				}`),
			code: 400,
		},
		{
			// extra characters in json
			input: []byte(`{
				"accesskeyidd": "keyid",
				"secretaccesskeyy": "key",
				"buckett": "bucket",
				"regionn" : "us-west-2",
				"payloadd" : "data"
			}`),
			code: 400,
		},
	}

	cloudConnector := CloudConnector{}

	handler := web.Handler(cloudConnector.AwsCloud)

	testHandlerHelper(invalidJSONSample, handler, t)
}

func testHandlerHelper(input []inputTest, handler web.Handler, t *testing.T) {

	for _, item := range input {

		request, err := http.NewRequest("POST", "/callwebhook", bytes.NewBuffer(item.input))
		if err != nil {
			t.Errorf("Unable to create new HTTP request %s", err.Error())
		}

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		fmt.Printf("recorder body %s", recorder.Body)

		if item.code != recorder.Code {
			t.Errorf("Status code didn't match, status code received: %d", recorder.Code)
		}
	}
}

func TestS3FileDoesntExist(t *testing.T) {
	region := "us-west-2"
	var logLevel aws.LogLevelType = 1

	awsConfig := aws.Config{
		Region:      &region,
		Credentials: credentials.NewStaticCredentials("AccessKeyID", "SecretAccessKey", ""),
		LogLevel:    &logLevel,
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: awsConfig,
	})

	if err != nil {
		t.Errorf("Failed to create the session %v", err)
	}

	newClient := s3.New(sess, &awsConfig)
	exists := s3FileExists(newClient, "bucket", "file")

	if exists {
		t.Error("File was found when it should not exist")
	}
}

func TestAwsCloudCallValidJsonInputWithFailure(t *testing.T) {
	var validJSONSample = []inputTest{
		{
			// extra characters in json
			input: []byte(`{
				"accesskeyid": "keyid",
				"secretaccesskey": "key",
				"bucket": "bucket",
				"region" : "us-west-2",
				"payload" : "data"
			}`),
			code: 400,
		},
	}
	cloudConnector := CloudConnector{}
	handler := web.Handler(cloudConnector.AwsCloud)
	testHandlerHelper(validJSONSample, handler, t)
}
