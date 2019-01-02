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
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metrics "github.impcloud.net/Responsive-Retail-Core/utilities/go-metrics"
)

const (
	oauth2                   = "oauth2"
	jsonApplication          = "application/json;charset=utf-8"
	oAuthConnectionTimeout   = 15
	webhookConnectionTimeout = 60
	responseMaxSize          = 16 << 20
)

// ProcessWebhook processes webhook requests
func ProcessWebhook(webhook Webhook, proxy string) (interface{}, error) {

	log.Debugf("Webhook authType is: %s\n", webhook.Auth.AuthType)

	// Check authentication type and run the appropriate POST or GET request.
	switch strings.ToLower(webhook.Auth.AuthType) {
	case oauth2:
		return getOrPostOAuth2Webhook(webhook, proxy)

	default:
		return getOrPostWebhook(webhook, proxy)
	}
}

// getAccessToken posts to get access token.
func getAccessToken(webhook Webhook, client *http.Client) (map[string]interface{}, error) {
	// Metrics
	metrics.GetOrRegisterGauge(`CloudConnector.getAccessToken.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`CloudConnector.getAccessToken.Success`, nil)
	mAuthenticateError := metrics.GetOrRegisterGauge("CloudConnector.getAccessToken.Auth-Error", nil)
	mResponseStatusError := metrics.GetOrRegisterGauge("CloudConnector.getAccessToken.Status-Error", nil)
	mDecoderError := metrics.GetOrRegisterGauge("CloudConnector.getAccessToken.Decoder-Error", nil)
	mAuthenticateLatency := metrics.GetOrRegisterTimer(`CloudConnector.getAccessToken.Authenticate-Latency`, nil)

	log.Debug("Getting access token")

	var tempResults map[string]interface{}

	log.Debugf("Posting to Auth endpoint %s with auth data", webhook.Auth.Endpoint)

	// Make the POST to authenticate
	request, _ := http.NewRequest("POST", webhook.Auth.Endpoint, nil)
	request.Header.Set("Authorization", webhook.Auth.Data)

	authenticateTimer := time.Now()
	response, err := client.Do(request)
	if err != nil {
		mAuthenticateError.Update(1)
		return nil, errors.Wrapf(err, "unable post auth webhook: %s", webhook.URL)
	}
	mAuthenticateLatency.Update(time.Since(authenticateTimer))
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			log.WithFields(log.Fields{
				"Method": "postOAuth2Webhook",
				"Action": "process the oath webhook request",
			}).Fatal(err.Error())
		}
	}()

	if response.StatusCode != http.StatusOK {
		mResponseStatusError.Update(int64(response.StatusCode))
		bodySize, errBoolResponseBody := checkBodySize(response)
		if !errBoolResponseBody {
			body := make([]byte, bodySize)
			_, err = io.ReadFull(response.Body, body)
			if err == nil {
				log.Errorf("Posting to Auth endpoint returned error status of: %d", response.StatusCode)

				return nil, errors.Wrapf(errors.New("webhook authentication error: "+webhook.Auth.Endpoint), "StatusCode %d , Response %s",
					response.StatusCode, string(body))
			}
		}

		return nil, errors.Wrapf(errors.New("webhook authentication error: "+webhook.Auth.Endpoint), "StatusCode %d", response.StatusCode)

	}

	if decErr := json.NewDecoder(response.Body).Decode(&tempResults); decErr != nil {
		mDecoderError.Update(1)
		return nil, decErr
	}

	//Return access token
	mSuccess.Update(1)
	return tempResults, nil
}

func getOrPostOAuth2Webhook(webhook Webhook, proxy string) (interface{}, error) {
	var mSuccess, mAuthenticateError, mResponseStatusError metrics.Gauge
	var mAuthenticateLatency metrics.Timer

	//Registering metrics based on HTTP method type.
	if webhook.Method == http.MethodPost {
		metrics.GetOrRegisterGauge(`CloudConnector.postOAuth2Webhook.Attempt`, nil).Update(1)
		mSuccess = metrics.GetOrRegisterGauge(`CloudConnector.postOAuth2Webhook.Success`, nil)
		mAuthenticateError = metrics.GetOrRegisterGauge("CloudConnector.postOAuth2Webhook.Auth-Error", nil)
		mAuthenticateLatency = metrics.GetOrRegisterTimer(`CloudConnector.postOAuth2Webhook.Authenticate-Latency`, nil)
		mResponseStatusError = metrics.GetOrRegisterGauge("CloudConnector.postOAuth2Webhook.Status-Error", nil)
	} else {
		metrics.GetOrRegisterGauge(`CloudConnector.getOAuth2Webhook.Attempt`, nil).Update(1)
		mSuccess = metrics.GetOrRegisterGauge(`CloudConnector.getOAuth2Webhook.Success`, nil)
		mAuthenticateError = metrics.GetOrRegisterGauge("CloudConnector.getOAuth2Webhook.Auth-Error", nil)
		mAuthenticateLatency = metrics.GetOrRegisterTimer(`CloudConnector.getOAuth2Webhook.Authenticate-Latency`, nil)
		mResponseStatusError = metrics.GetOrRegisterGauge("CloudConnector.getOAuth2Webhook.Status-Error", nil)

	}

	log.Debugf("Posting to endpoint %s\n with auth", webhook.Auth.Endpoint)

	//Set timeout and proxy for http client if present/needed
	timeout := time.Duration(oAuthConnectionTimeout) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}
	if proxy != "" {
		proxyURL, parseErr := url.Parse(proxy)
		if parseErr != nil {
			return nil, errors.Wrapf(parseErr, "unable to post notification due to error in parsing proxy URL: %s", proxy)
		}
		transport := http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		client.Transport = &transport
	}

	//Get Access token for the endpoint
	authenticateTimer := time.Now()
	accessTokenMap, accessTokenErr := getAccessToken(webhook, client)
	if accessTokenErr != nil {
		mAuthenticateError.Update(1)
		return nil, accessTokenErr
	}
	mAuthenticateLatency.Update(time.Since(authenticateTimer))

	//Based on HTTP method type, set body and content type.
	var request *http.Request
	if webhook.Method == http.MethodPost {
		mData, err := json.Marshal(webhook.Payload)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to marshal payload")
		}
		request, _ = http.NewRequest(webhook.Method, webhook.URL, bytes.NewBuffer(mData))
		request.Header.Set("content-type", jsonApplication)
	} else {
		request, _ = http.NewRequest(webhook.Method, webhook.URL, nil)

	}
	request.Header.Set("Authorization", accessTokenMap["token_type"].(string)+" "+accessTokenMap["access_token"].(string))
	if webhook.Header != nil {
		request.Header = webhook.Header
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to post notification: %s", webhook.Auth.Endpoint)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			log.WithFields(log.Fields{
				"Method": "getOrPostOAuth2Webhook",
				"Action": "process the OAuth webhook request",
			}).Fatal(err.Error())
		}
	}()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		mResponseStatusError.Update(int64(response.StatusCode))
		bodySize, errBoolResponseBody := checkBodySize(response)
		if !errBoolResponseBody {
			body := make([]byte, bodySize)
			_, err = io.ReadFull(response.Body, body)
			if err == nil {
				return nil, errors.Wrapf(errors.New("request error"), "StatusCode %d , Response %s",
					response.StatusCode, string(body))
			}
		}
		return nil, errors.Wrapf(errors.New("request error"), "StatusCode %d ", response.StatusCode)
	}

	mSuccess.Update(1)
	return response.Body, nil
}

func getOrPostWebhook(webhook Webhook, proxy string) (interface{}, error) {

	var mSuccess, mWebhookResponseStatusError, mMarshalError metrics.Gauge
	var mWebhookLatency metrics.Timer

	//Registering metrics based on HTTP method type.
	if webhook.Method == http.MethodPost {
		metrics.GetOrRegisterGauge(`CloudConnector.getWebhook.Attempt`, nil).Update(1)
		mSuccess = metrics.GetOrRegisterGauge(`CloudConnector.getWebhook.Success`, nil)
		mWebhookResponseStatusError = metrics.GetOrRegisterGauge("CloudConnector.getWebhook.Status-Error", nil)
		mWebhookLatency = metrics.GetOrRegisterTimer(`CloudConnector.getWebhook.mWebhookPost-Latency`, nil)
	} else {
		metrics.GetOrRegisterGauge(`CloudConnector.postWebhook.Attempt`, nil).Update(1)
		mSuccess = metrics.GetOrRegisterGauge(`CloudConnector.postWebhook.Success`, nil)
		mMarshalError = metrics.GetOrRegisterGauge("CloudConnector.postWebhook.Marshal-Error", nil)
		mWebhookResponseStatusError = metrics.GetOrRegisterGauge("CloudConnector.postWebhook.Webhook-Status-Error", nil)
		mWebhookLatency = metrics.GetOrRegisterTimer(`CloudConnector.postWebhook.mWebhookPost-Latency`, nil)

	}

	log.Debugf("Getting from endpoint %s without auth", webhook.URL)

	//Adding or modifying neccessary parameters to http client such as proxy and timeout
	timeout := time.Duration(oAuthConnectionTimeout) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}
	if proxy != "" {
		proxyURL, parseErr := url.Parse(proxy)
		if parseErr != nil {
			return nil, errors.Wrapf(parseErr, "unable to post notification due to error in parsing proxy URL: %s", proxy)
		}
		transport := http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		client.Transport = &transport
	}

	//Request creation based on HTTTP mehtod type and adding headers
	var request *http.Request
	if webhook.Method == http.MethodPost {
		mData, err := json.Marshal(webhook.Payload)
		if err != nil {
			mMarshalError.Update(1)
			return nil, errors.Wrapf(err, "unable to marshal payload")
		}
		request, _ = http.NewRequest(webhook.Method, webhook.URL, bytes.NewBuffer(mData))
		request.Header.Set("content-type", jsonApplication)
	} else {
		request, _ = http.NewRequest(webhook.Method, webhook.URL, nil)
	}

	if webhook.Header != nil {
		request.Header = webhook.Header
	}

	getTimer := time.Now()
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get notification: %s", webhook.Auth.Endpoint)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			log.WithFields(log.Fields{
				"Method": "getOAuth2Webhook",
				"Action": "process the oath webhook request",
			}).Fatal(err.Error())
		}
	}()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		mWebhookResponseStatusError.Update(int64(response.StatusCode))
		bodySize, errBoolResponseBody := checkBodySize(response)
		if !errBoolResponseBody {
			body := make([]byte, bodySize)
			_, err = io.ReadFull(response.Body, body)
			if err == nil {
				return nil, errors.Wrapf(errors.New("request error"), "StatusCode %d , Response %s",
					response.StatusCode, string(body))
			}
		}
		return nil, errors.Wrapf(errors.New("request error"), "StatusCode %d ", response.StatusCode)
	}
	mWebhookLatency.Update(time.Since(getTimer))

	mSuccess.Update(1)
	return response.Body, nil
}

func checkBodySize(response *http.Response) (int64, bool) {
	var writer http.ResponseWriter
	resBody := http.MaxBytesReader(writer, response.Body, responseMaxSize)
	bodySize, err := io.Copy(ioutil.Discard, resBody)
	if err != nil {
		return 0, false
	}
	return bodySize, true
}
