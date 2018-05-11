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
	oauth2            = "oauth2"
	jsonApplication   = "application/json;charset=utf-8"
	oAuthConnectionTimeout = 15
	webhookConnectionTimeout = 60
	responseMaxSize   = 16 << 20
)

// ProcessWebhook processes webhook requests
func ProcessWebhook(webh Webhook, proxy string) error {
	var postErr error

	log.Debugf("Webhook authType is: %s\n", webh.Auth.AuthType)

	// Check authentication type and run the appropriate post
	switch strings.ToLower(webh.Auth.AuthType) {
	case oauth2:
		postErr = postOAuth2Webhook(webh, proxy)
	default:
		postErr = postWebhook(webh, proxy)
	}

	return postErr
}

// postOAuth2Webhook post to webhook using oauth 2 authentication
func postOAuth2Webhook(webhook Webhook, proxy string) error {
	// Metrics
	metrics.GetOrRegisterGauge(`CloudConnector.postOAuthWebhook.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`CloudConnector.postOAuthWebhook.Success`, nil)
	mAuthenticateError := metrics.GetOrRegisterGauge("CloudConnector.postOAuthWebhook.Auth-Error", nil)
	mResponseStatusError := metrics.GetOrRegisterGauge("CloudConnector.postOAuthWebhook.Status-Error", nil)
	mDecoderError := metrics.GetOrRegisterGauge("CloudConnector.postOAuthWebhook.Decoder-Error", nil)
	mMarshalError := metrics.GetOrRegisterGauge("CloudConnector.postOAuthWebhook.Marshal-Error", nil)
	mWebhookPostError := metrics.GetOrRegisterGauge("CloudConnector.postOAuthWebhook.Webhook-Error", nil)
	mAuthenticateLatency := metrics.GetOrRegisterTimer(`CloudConnector.postOAuthWebhook.Authenticate-Latency`, nil)
	mWebhookPostLatency := metrics.GetOrRegisterTimer(`CloudConnector.postOAuthWebhook.WebhookPost-Latency`, nil)

	log.Debug("Posting with oauth 2 Authentication")

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

	var tempResults map[string]interface{}

	log.Debugf("Posting to Auth endpoint %s with auth data", webhook.Auth.Endpoint)

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
		bodySize, errBoolResponseBody := checkBodySize(response)
		if !errBoolResponseBody {
			body := make([]byte, bodySize)
			_, err = io.ReadFull(response.Body, body)
			if err == nil {
				log.Errorf("Posting to Auth endpoint returned error status of: %d", response.StatusCode)

				return errors.Wrapf(errors.New("webhook authentication error: "+webhook.Auth.Endpoint), "StatusCode %d , Response %s",
					response.StatusCode, string(body))
			}
		}

		return errors.Wrapf(errors.New("webhook authentication error: "+webhook.Auth.Endpoint), "StatusCode %d", response.StatusCode)

	}

	if decErr := json.NewDecoder(response.Body).Decode(&tempResults); decErr != nil {
		mDecoderError.Update(1)
		return decErr
	}

	// Do post with access token
	mData, err := json.Marshal(webhook.Payload)
	if err != nil {
		mMarshalError.Update(1)
		log.Error("unable to marshal Auth response data")

		return errors.Wrapf(err, "unable to marshal payload")
	}
	postTimer := time.Now()
	if err := postWebhookAccessToken(mData, webhook.URL, tempResults["token_type"].(string), tempResults["access_token"].(string), client); err != nil {
		mWebhookPostError.Update(1)
		return err
	}
	mWebhookPostLatency.Update(time.Since(postTimer))

	mSuccess.Update(1)
	return nil
}

func postWebhookAccessToken(data []byte, URL string, tokenType string, accessToken string, client *http.Client) error {
	mPostResponseStatusError := metrics.GetOrRegisterGauge("CloudConnector.postWebhookAccessToken.Status-Error", nil)

	log.Debugf("Posting to endpoint %s\nwith auth", URL)

	postRequest, _ := http.NewRequest("POST", URL, bytes.NewBuffer(data))
	postRequest.Header.Set("content-type", jsonApplication)
	postRequest.Header.Set("Authorization", tokenType+" "+accessToken)
	postResponse, err := client.Do(postRequest)
	if err != nil {
		return errors.Wrapf(err, "unable to post notification: %s", URL)
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

	return nil
}

// postWebhook post to webhook
func postWebhook(webh Webhook, proxy string) error {
	metrics.GetOrRegisterGauge(`CloudConnector.postWebhook.Attempt`, nil).Update(1)
	mSuccess := metrics.GetOrRegisterGauge(`CloudConnector.postWebhook.Success`, nil)
	mMarshalError := metrics.GetOrRegisterGauge("CloudConnector.postWebhook.Marshal-Error", nil)
	mWebhookPostError := metrics.GetOrRegisterGauge("CloudConnector.postWebhook.Webhook-Error", nil)
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
	if err != nil || response.StatusCode != http.StatusOK {
		mWebhookPostError.Update(int64(response.StatusCode))
		return err
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
