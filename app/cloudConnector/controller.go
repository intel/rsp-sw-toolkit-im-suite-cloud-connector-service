/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package cloudConnector

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	metrics "github.com/intel/rsp-sw-toolkit-im-suite-utilities/go-metrics"
	"github.com/intel/rsp-sw-toolkit-im-suite-utilities/helper"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	oauth2                   = "oauth2"
	jsonApplication          = "application/json;charset=utf-8"
	oAuthConnectionTimeout   = 15
	webhookConnectionTimeout = 60
	responseMaxSize          = 16 << 20
)

var accessTokens sync.Map

// ProcessWebhook processes webhook requests
func ProcessWebhook(webhook Webhook, proxy string) (*WebhookResponse, error) {

	log.Debugf("Webhook authType is: %s\n", webhook.Auth.AuthType)

	// Check authentication type and run the appropriate POST or GET request.
	switch strings.ToLower(webhook.Auth.AuthType) {
	case oauth2:
		// Call endpoint using authentication stored in the accessTokens map or using a newly retrieved token
		response, err := getOrPostOAuth2Webhook(webhook, proxy)
		// If the call fails and the status code returned is auth related then we need to try again
		// It's possible the cached token timed out so we should attempt to get a new token before failing
		if response != nil && err != nil && (response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden) {
			return getOrPostOAuth2Webhook(webhook, proxy)
		}
		return response, err

	default:
		return getOrPostWebhook(webhook, proxy)
	}
}

// getAccessToken posts to get access token.
func getAccessToken(webhook Webhook, proxy string) error {
	// Metrics
	metrics.GetOrRegisterGauge(`CloudConnector.getAccessToken.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`CloudConnector.getAccessToken.Success`, nil)
	mAuthenticateError := metrics.GetOrRegisterGauge("CloudConnector.getAccessToken.Auth-Error", nil)
	mResponseStatusError := metrics.GetOrRegisterGauge("CloudConnector.getAccessToken.Status-Error", nil)
	mDecoderError := metrics.GetOrRegisterGauge("CloudConnector.getAccessToken.Decoder-Error", nil)
	mAuthenticateLatency := metrics.GetOrRegisterTimer(`CloudConnector.getAccessToken.Authenticate-Latency`, nil)

	log.Debugf("POST to endpoint %s\n with auth to get access token", webhook.Auth.Endpoint)

	var accessTokenMap map[string]interface{}
	endPointCache, ok := accessTokens.Load(webhook.Auth.Endpoint)
	if ok && endPointCache != nil {
		accessTokenMap = endPointCache.(map[string]interface{})
	}

	// Check for an existing token and if you find a valid token that isn't expired use that and don't call the endpoint
	if accessTokenMap == nil || accessTokenMap["token_type"] == nil || accessTokenMap["token_type"] == "" ||
		accessTokenMap["access_token"] == nil || accessTokenMap["access_token"] == "" ||
		accessTokenMap["expires_in"] == nil || accessTokenMap["expires_in"] == "" ||
		accessTokenMap["expirationDate"] == nil ||
		(accessTokenMap["expirationDate"] != nil && accessTokenMap["expirationDate"].(int64) < helper.UnixMilliNow()) {
		log.Debug("Getting access token")

		client, httpClientErr := getHTTPClient(oAuthConnectionTimeout, proxy)
		if httpClientErr != nil {
			return errors.Wrapf(httpClientErr, "unable to %s webhook due to error in parsing proxy URL: %s", webhook.Method, proxy)
		}

		// Make the POST to authenticate
		request, _ := http.NewRequest("POST", webhook.Auth.Endpoint, nil)
		request.Header.Set("Authorization", webhook.Auth.Data)

		authenticateTimer := time.Now()
		response, err := client.Do(request)
		if err != nil {
			mAuthenticateError.Update(1)
			return errors.Wrapf(err, "unable post auth webhook: %s", webhook.URL)
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
			webhookResponse, err := getWebhookResponse(response)
			if err != nil {
				log.Errorf("Posting to Auth endpoint returned error status of: %d", response.StatusCode)
				return err
			}
			return errors.Wrapf(errors.New("webhook authentication error"+webhook.Auth.Endpoint), "StatusCode %d with following response %s",
				webhookResponse.StatusCode, string(webhookResponse.Body))
		}

		if decErr := json.NewDecoder(response.Body).Decode(&accessTokenMap); decErr != nil {
			mDecoderError.Update(1)
			return decErr
		}

		if accessTokenMap["expires_in"] != nil {
			accessTokenMap["expirationDate"] = helper.UnixMilliNow() + int64(accessTokenMap["expires_in"].(float64)*1000)
		}
		accessTokens.Store(webhook.Auth.Endpoint, accessTokenMap)

		//Return access token
		mSuccess.Update(1)
	}
	//Return access token
	mSuccess.Update(1)
	return nil
}

func getOrPostOAuth2Webhook(webhook Webhook, proxy string) (*WebhookResponse, error) {
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

	log.Debugf("%s to endpoint %s with auth", webhook.Method, webhook.URL)

	//Set timeout and proxy for http client if present/needed
	client, httpClientErr := getHTTPClient(webhookConnectionTimeout, proxy)
	if httpClientErr != nil {
		return nil, errors.Wrapf(httpClientErr, "unable to %s webhook due to error in parsing proxy URL: %s", webhook.Method, proxy)
	}

	//Get Access token for the endpoint
	authenticateTimer := time.Now()
	accessTokenErr := getAccessToken(webhook, proxy)
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

	endPointCache, ok := accessTokens.Load(webhook.Auth.Endpoint)
	endPointCacheMap := endPointCache.(map[string]interface{})
	if ok && endPointCacheMap != nil {
		if endPointCacheMap["token_type"].(string) != "" &&
			endPointCacheMap["access_token"].(string) != "" {
			request.Header.Set("Authorization", endPointCacheMap["token_type"].(string)+" "+
				endPointCacheMap["access_token"].(string))
		}
	}

	if webhook.Header != nil {
		request.Header = webhook.Header
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to %s endpoint: %s", webhook.Method, webhook.URL)
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
		if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			accessTokens.Delete(webhook.Auth.Endpoint)
		}

		mResponseStatusError.Update(int64(response.StatusCode))

		webhookResponse, responseErr := getWebhookResponse(response)
		if responseErr != nil {
			return nil, responseErr
		}
		return nil, errors.Wrapf(errors.New("request error:"), "StatusCode %d with following response %s",
			webhookResponse.StatusCode, string(webhookResponse.Body))
	}

	webhookResponse, err := getWebhookResponse(response)
	if err != nil {
		return nil, err
	}
	mSuccess.Update(1)

	return webhookResponse, nil
}

func getOrPostWebhook(webhook Webhook, proxy string) (*WebhookResponse, error) {

	var mSuccess, mWebhookResponseStatusError, mMarshalError metrics.Gauge
	var mWebhookLatency metrics.Timer

	//Registering metrics based on HTTP method type.
	if webhook.Method == http.MethodPost {
		metrics.GetOrRegisterGauge(`CloudConnector.postWebhook.Attempt`, nil).Update(1)
		mSuccess = metrics.GetOrRegisterGauge(`CloudConnector.postWebhook.Success`, nil)
		mMarshalError = metrics.GetOrRegisterGauge("CloudConnector.postWebhook.Marshal-Error", nil)
		mWebhookResponseStatusError = metrics.GetOrRegisterGauge("CloudConnector.postWebhook.Webhook-Status-Error", nil)
		mWebhookLatency = metrics.GetOrRegisterTimer(`CloudConnector.postWebhook.mWebhookPost-Latency`, nil)
	} else {
		metrics.GetOrRegisterGauge(`CloudConnector.getWebhook.Attempt`, nil).Update(1)
		mSuccess = metrics.GetOrRegisterGauge(`CloudConnector.getWebhook.Success`, nil)
		mMarshalError = metrics.GetOrRegisterGauge("CloudConnector.getWebhook.Marshal-Error", nil)
		mWebhookResponseStatusError = metrics.GetOrRegisterGauge("CloudConnector.getWebhook.Status-Error", nil)
		mWebhookLatency = metrics.GetOrRegisterTimer(`CloudConnector.getWebhook.mWebhookPost-Latency`, nil)

	}

	log.Debugf("%s to endpoint %s without auth", webhook.Method, webhook.URL)

	//Set timeout and proxy for http client if present/needed
	client, httpClientErr := getHTTPClient(webhookConnectionTimeout, proxy)
	if httpClientErr != nil {
		return nil, errors.Wrapf(httpClientErr, "unable to %s webhook due to error in parsing proxy URL: %s", webhook.Method, proxy)
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
		return nil, errors.Wrapf(err, "unable to %s endpoint: %s", webhook.Method, webhook.URL)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			log.WithFields(log.Fields{
				"Method": "getOrPostWebhook",
				"Action": "process the webhook request",
			}).Fatal(err.Error())
		}
	}()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		mWebhookResponseStatusError.Update(int64(response.StatusCode))
		webhookResponse, responseErr := getWebhookResponse(response)
		if responseErr != nil {
			return nil, responseErr
		}
		return nil, errors.Wrapf(errors.New("request error:"), "StatusCode %d with following response %s",
			webhookResponse.StatusCode, string(webhookResponse.Body))
	}
	mWebhookLatency.Update(time.Since(getTimer))

	webhookResponse, err := getWebhookResponse(response)
	if err != nil {
		return nil, err
	}

	mSuccess.Update(1)

	return webhookResponse, nil
}

func getWebhookResponse(response *http.Response) (*WebhookResponse, error) {
	var webhookResponse WebhookResponse
	response.Body = http.MaxBytesReader(nil, response.Body, responseMaxSize)
	body, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		return nil, errors.Wrapf(errors.New("error in reading webhook response"), readErr.Error())
	}
	webhookResponse.Body = body
	webhookResponse.StatusCode = response.StatusCode
	webhookResponse.Header = response.Header
	return &webhookResponse, nil
}

func getHTTPClient(timeout time.Duration, proxy string) (*http.Client, error) {
	timeOutSec := timeout * time.Second
	client := &http.Client{
		Timeout: timeOutSec,
	}
	if proxy != "" {
		proxyURL, parseErr := url.Parse(proxy)
		if parseErr != nil {
			return nil, parseErr
		}
		transport := http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		client.Transport = &transport
	}
	return client, nil
}
