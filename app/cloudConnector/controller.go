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
		if webhook.Method == http.MethodGet {
			return getOAuth2Webhook(webhook, proxy)
		}
		return nil, postOAuth2Webhook(webhook, proxy)

	default:
		if webhook.Method == http.MethodGet {
			return getWebhook(webhook, proxy)
		}
		return nil, postWebhook(webhook, proxy)
	}
}

// getAccessToken posts to get access token.
func getAccessToken(webhook Webhook, client *http.Client) (map[string]interface{}, error) {
	// Metrics
	metrics.GetOrRegisterGauge(`CloudConnector.postOAuthWebhook.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`CloudConnector.postOAuthWebhook.Success`, nil)
	mAuthenticateError := metrics.GetOrRegisterGauge("CloudConnector.postOAuthWebhook.Auth-Error", nil)
	mResponseStatusError := metrics.GetOrRegisterGauge("CloudConnector.postOAuthWebhook.Status-Error", nil)
	mDecoderError := metrics.GetOrRegisterGauge("CloudConnector.postOAuthWebhook.Decoder-Error", nil)
	mAuthenticateLatency := metrics.GetOrRegisterTimer(`CloudConnector.postOAuthWebhook.Authenticate-Latency`, nil)

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

func postOAuth2Webhook(webhook Webhook, proxy string) error {
	metrics.GetOrRegisterGauge(`CloudConnector.postOAuth2Webhook.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`CloudConnector.postOAuth2Webhook.Success`, nil)
	mAuthenticateError := metrics.GetOrRegisterGauge("CloudConnector.postOAuth2Webhook.Auth-Error", nil)
	mAuthenticateLatency := metrics.GetOrRegisterTimer(`CloudConnector.postOAuth2Webhook.Authenticate-Latency`, nil)
	mPostResponseStatusError := metrics.GetOrRegisterGauge("CloudConnector.postOAuth2Webhook.Status-Error", nil)

	log.Debugf("Posting to endpoint %s\nwith auth", webhook.Auth.Endpoint)

	timeout := time.Duration(oAuthConnectionTimeout) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}
	if proxy != "" {
		proxyURL, parseErr := url.Parse(proxy)
		if parseErr != nil {
			log.Println(parseErr)
		}
		transport := http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		client.Transport = &transport
	}

	authenticateTimer := time.Now()
	accessTokenMap, accessTokenErr := getAccessToken(webhook, client)
	if accessTokenErr != nil {
		mAuthenticateError.Update(1)
		return accessTokenErr
	}
	mAuthenticateLatency.Update(time.Since(authenticateTimer))

	mData, err := json.Marshal(webhook.Payload)
	if err != nil {
		return errors.Wrapf(err, "unable to marshal payload")
	}

	postRequest, _ := http.NewRequest(webhook.Method, webhook.URL, bytes.NewBuffer(mData))
	postRequest.Header.Set("content-type", jsonApplication)
	postRequest.Header.Set("Authorization", accessTokenMap["token_type"].(string)+" "+accessTokenMap["access_token"].(string))
	postResponse, err := client.Do(postRequest)
	if err != nil {
		return errors.Wrapf(err, "unable to post notification: %s", webhook.Auth.Endpoint)
	}
	defer func() {
		if closeErr := postResponse.Body.Close(); closeErr != nil {
			log.WithFields(log.Fields{
				"Method": "postOAuth2Webhook",
				"Action": "process the oath webhook request",
			}).Fatal(err.Error())
		}
	}()

	if postResponse.StatusCode != http.StatusOK && postResponse.StatusCode != http.StatusNoContent {
		mPostResponseStatusError.Update(int64(postResponse.StatusCode))
		bodySize, errBoolResponseBody := checkBodySize(postResponse)
		if !errBoolResponseBody {
			body := make([]byte, bodySize)
			_, err = io.ReadFull(postResponse.Body, body)
			if err == nil {
				return errors.Wrapf(errors.New("request error"), "StatusCode %d , Response %s",
					postResponse.StatusCode, string(body))
			}
		}
		return errors.Wrapf(errors.New("request error"), "StatusCode %d ", postResponse.StatusCode)
	}

	mSuccess.Update(1)
	return nil
}

func getOAuth2Webhook(webhook Webhook, proxy string) (interface{}, error) {
	metrics.GetOrRegisterGauge(`CloudConnector.getOAuth2Webhook.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`CloudConnector.getOAuth2Webhook.Success`, nil)
	mAuthenticateError := metrics.GetOrRegisterGauge("CloudConnector.getOAuth2Webhook.Auth-Error", nil)
	mAuthenticateLatency := metrics.GetOrRegisterTimer(`CloudConnector.getOAuth2Webhook.Authenticate-Latency`, nil)
	mGetResponseStatusError := metrics.GetOrRegisterGauge("CloudConnector.getOAuth2Webhook.Status-Error", nil)

	log.Debugf("Posting to endpoint %s\nwith auth", webhook.Auth.Endpoint)

	timeout := time.Duration(oAuthConnectionTimeout) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}
	if proxy != "" {
		proxyURL, parseErr := url.Parse(proxy)
		if parseErr != nil {
			log.Println(parseErr)
		}
		transport := http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		client.Transport = &transport
	}

	authenticateTimer := time.Now()
	accessTokenMap, accessTokenErr := getAccessToken(webhook, client)
	if accessTokenErr != nil {
		mAuthenticateError.Update(1)
		return nil, accessTokenErr
	}
	mAuthenticateLatency.Update(time.Since(authenticateTimer))

	getRequest, _ := http.NewRequest(webhook.Method, webhook.URL, nil)
	getRequest.Header.Set("content-type", jsonApplication)
	getRequest.Header.Set("Authorization", accessTokenMap["token_type"].(string)+" "+accessTokenMap["access_token"].(string))
	getResponse, err := client.Do(getRequest)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to post notification: %s", webhook.Auth.Endpoint)
	}
	defer func() {
		if closeErr := getResponse.Body.Close(); closeErr != nil {
			log.WithFields(log.Fields{
				"Method": "postOAuth2Webhook",
				"Action": "process the oath webhook request",
			}).Fatal(err.Error())
		}
	}()

	if getResponse.StatusCode != http.StatusOK && getResponse.StatusCode != http.StatusNoContent {
		mGetResponseStatusError.Update(int64(getResponse.StatusCode))
		bodySize, errBoolResponseBody := checkBodySize(getResponse)
		if !errBoolResponseBody {
			body := make([]byte, bodySize)
			_, err = io.ReadFull(getResponse.Body, body)
			if err == nil {
				return nil, errors.Wrapf(errors.New("request error"), "StatusCode %d , Response %s",
					getResponse.StatusCode, string(body))
			}
		}
		return nil, errors.Wrapf(errors.New("request error"), "StatusCode %d ", getResponse.StatusCode)
	}

	mSuccess.Update(1)
	return getResponse.Body, nil
}

func getWebhook(webhook Webhook, proxy string) (interface{}, error) {
	metrics.GetOrRegisterGauge(`CloudConnector.getWebhook.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`CloudConnector.getWebhook.Success`, nil)
	mGetResponseStatusError := metrics.GetOrRegisterGauge("CloudConnector.getWebhook.Status-Error", nil)
	mWebhookGetLatency := metrics.GetOrRegisterTimer(`CloudConnector.getWebhook.mWebhookPost-Latency`, nil)

	log.Debugf("Posting to endpoint %s with auth", webhook.URL)

	//Adding or modifying neccessary parameters to http client fo proxy and timeout
	timeout := time.Duration(oAuthConnectionTimeout) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}
	if proxy != "" {
		proxyURL, parseErr := url.Parse(proxy)
		if parseErr != nil {
			log.Println(parseErr)
		}
		transport := http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		client.Transport = &transport
	}

	//Request creation and adding headers
	getRequest, _ := http.NewRequest(webhook.Method, webhook.URL, nil)
	getRequest.Header = webhook.Header

	getTimer := time.Now()
	getResponse, err := client.Do(getRequest)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to post notification: %s", webhook.Auth.Endpoint)
	}
	defer func() {
		if closeErr := getResponse.Body.Close(); closeErr != nil {
			log.WithFields(log.Fields{
				"Method": "postOAuth2Webhook",
				"Action": "process the oath webhook request",
			}).Fatal(err.Error())
		}
	}()

	if getResponse.StatusCode != http.StatusOK && getResponse.StatusCode != http.StatusNoContent {
		mGetResponseStatusError.Update(int64(getResponse.StatusCode))
		bodySize, errBoolResponseBody := checkBodySize(getResponse)
		if !errBoolResponseBody {
			body := make([]byte, bodySize)
			_, err = io.ReadFull(getResponse.Body, body)
			if err == nil {
				return nil, errors.Wrapf(errors.New("request error"), "StatusCode %d , Response %s",
					getResponse.StatusCode, string(body))
			}
		}
		return nil, errors.Wrapf(errors.New("request error"), "StatusCode %d ", getResponse.StatusCode)
	}
	mWebhookGetLatency.Update(time.Since(getTimer))

	mSuccess.Update(1)
	return getResponse.Body, nil
}

// postWebhook post to webhook
func postWebhook(webh Webhook, proxy string) error {
	metrics.GetOrRegisterGauge(`CloudConnector.postWebhook.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`CloudConnector.postWebhook.Success`, nil)
	mMarshalError := metrics.GetOrRegisterGauge("CloudConnector.postWebhook.Marshal-Error", nil)
	mWebhookPostError := metrics.GetOrRegisterGauge("CloudConnector.postWebhook.Webhook-Error", nil)
	mWebhookPostResponseStatusError := metrics.GetOrRegisterGauge("CloudConnector.postWebhook.Webhook-Status-Error", nil)
	mWebhookPostLatency := metrics.GetOrRegisterTimer(`CloudConnector.postWebhook.mWebhookPost-Latency`, nil)

	if webh.Auth.AuthType != "" {
		log.Debugf("Posting with %s Authentication\n", webh.Auth.AuthType)
	} else {
		log.Debug("Posting without Authentication\n")
	}

	timeout := time.Duration(webhookConnectionTimeout) * time.Second
	client := &http.Client{
		Timeout: timeout,
	}
	if proxy != "" {
		proxyURL, parseErr := url.Parse(proxy)
		if parseErr != nil {
			log.Println(parseErr)
		}
		transport := http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		client.Transport = &transport
	}

	mData, err := json.Marshal(webh.Payload)
	if err != nil {
		mMarshalError.Update(1)
		return errors.Wrapf(err, "unable to marshal payload")
	}
	request, _ := http.NewRequest(webh.Method, webh.URL, bytes.NewBuffer(mData))
	request.Header = webh.Header

	postTimer := time.Now()
	response, err := client.Do(request)
	if err != nil {
		mWebhookPostError.Update(1)
		return errors.Errorf("Error posting to Webhook: %s", err)
	}

	if response.StatusCode != http.StatusOK {
		mWebhookPostResponseStatusError.Update(int64(response.StatusCode))
		return errors.Errorf("Error posting to Webhook, response status returned is %d", response.StatusCode)
	}

	mWebhookPostLatency.Update(time.Since(postTimer))
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			log.WithFields(log.Fields{
				"Method": "postWebhook",
				"Action": "process the webhook request",
			}).Fatal(err.Error())
		}
	}()

	mSuccess.Update(1)
	return nil
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
