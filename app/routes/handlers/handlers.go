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

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/Responsive-Retail-Core/cloud-connector-service/app/cloudConnector"
	"github.impcloud.net/Responsive-Retail-Core/cloud-connector-service/app/config"
	"github.impcloud.net/Responsive-Retail-Core/cloud-connector-service/pkg/web"
	metrics "github.impcloud.net/Responsive-Retail-Core/utilities/go-metrics"
	"github.impcloud.net/Responsive-Retail-Core/utilities/gojsonschema"
	"github.impcloud.net/Responsive-Retail-Core/utilities/helper"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/s3"
)

// CloudConnector represents the User API method handler set.
type CloudConnector struct {
}

// Response wraps results, inlinecount, and extra fields in a json object
// swagger:model resultsResponse
type Response struct {
	Results interface{} `json:"results"`
	Count   int         `json:"count,omitempty"`
}

// ErrorList provides a collection of errors for processing
// swagger:response ErrReport
type ErrorList struct {
	// The error list
	// in: body
	Errors []ErrReport `json:"errors"`
}

//ErrReport is used to wrap schema validation errors int json object
type ErrReport struct {
	Field       string      `json:"field"`
	ErrorType   string      `json:"errortype"`
	Value       interface{} `json:"value"`
	Description string      `json:"description"`
}

// Index is used for Docker Healthcheck commands to indicate
// whether the http server is up and running to take requests
// nolint: unparam
func (connector *CloudConnector) Index(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
	web.Respond(ctx, writer, config.AppConfig.ServiceName, http.StatusOK)
	return nil
}

//CallWebhook
// 200 OK, 400 Bad Request, 404 endpoint not found, 500 Internal Error
func (connector *CloudConnector) CallWebhook(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {

	traceID := ctx.Value(web.KeyValues).(*web.ContextValues).TraceID

	// Metrics
	metrics.GetOrRegisterGauge("CloudConnector.callwebhook.Attempt", nil).Update(1)

	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("CloudConnector.callwebhook.Latency", nil).Update(time.Since(startTime))
	var webHookObj cloudConnector.Webhook

	validationErrors, marshalError := unmarshalRequestBody(writer, request, &webHookObj, cloudConnector.WebhookSchema)

	if marshalError != nil {
		if marshalError.Error() == "http: request body too large" {
			log.WithFields(log.Fields{
				"Method": "CallWebhook",
				"Action": "post notification to webhooks",
				"Code":   http.StatusRequestEntityTooLarge,
			}).Error("Request Body too large")
			web.RespondError(ctx, writer, marshalError, http.StatusRequestEntityTooLarge)
			return nil
		}
		return marshalError
	}

	log.WithFields(log.Fields{
		"Method":      "CallWebhook",
		"Webhook Obj": webHookObj.Payload,
		"TraceID":     traceID,
	}).Debug()

	if len(validationErrors) > 0 {
		log.WithFields(log.Fields{
			"Method": "CallWebhook",
			"Action": "post notification to webhooks",
			"Code":   http.StatusBadRequest,
		}).Error("Validation errors")
		web.Respond(ctx, writer, validationErrors, http.StatusBadRequest)
		return nil
	}

	//Get call always has an response object, so isAsync flag will be ignored even if set
	if webHookObj.IsAsync && webHookObj.Method == http.MethodPost {
		go cloudCall(ctx, writer, webHookObj)
		web.Respond(ctx, writer, nil, http.StatusOK)

	} else {
		//In case if GET calls Isasync option is set to true by mistake, we reset it back to false.
		webHookObj.IsAsync = false
		cloudCall(ctx, writer, webHookObj)
	}

	return nil
}

func cloudCall(ctx context.Context, writer http.ResponseWriter, webHookObj cloudConnector.Webhook) {

	traceID := ctx.Value(web.KeyValues).(*web.ContextValues).TraceID
	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("CloudConnector.syncCloudCall.Latency", nil).Update(time.Since(startTime))
	mSuccess := metrics.GetOrRegisterGauge("CloudConnector.syncCloudCall.Success", nil)
	mError := metrics.GetOrRegisterGauge("CloudConnector.syncCloudCall.Error", nil)

	if response, err := cloudConnector.ProcessWebhook(webHookObj, config.AppConfig.HttpsProxyURL); err != nil {
		log.WithFields(log.Fields{
			"Method":      "CallWebhook",
			"Action":      "process the webhook request",
			"Webhook URL": webHookObj.URL,
			"TraceID":     traceID,
		}).Error(err.Error())

		if !webHookObj.IsAsync {
			web.Respond(ctx, writer, err, http.StatusNotFound)
		}
		mError.Update(1)
	} else {
		log.WithFields(log.Fields{
			"Method":     "ProcessWebhook",
			"TraceID":    traceID,
			"webhookURL": webHookObj.URL,
		}).Debug("Successful!")

		if !webHookObj.IsAsync {
			if response != nil {
				web.Respond(ctx, writer, response, http.StatusOK)
			}
			web.Respond(ctx, writer, nil, http.StatusOK)
		}
		mSuccess.Update(1)
	}
}

// AwsCloud triggers a set of rules based on the user input
// 200 OK, 400 Bad Request, 500 Internal Error
func (connector *CloudConnector) AwsCloud(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {

	traceID := ctx.Value(web.KeyValues).(*web.ContextValues).TraceID
	// Metrics
	metrics.GetOrRegisterMeter("CloudConnector.AwsCloud.Attempt", nil).Mark(1)

	startTime := time.Now()
	defer metrics.GetOrRegisterTimer("CloudConnector.AwsCloud.Latency", nil).Update(time.Since(startTime))
	mSuccess := metrics.GetOrRegisterMeter("CloudConnector.AwsCloud.Success", nil)

	statusCode := http.StatusOK
	var logLevel aws.LogLevelType = 1
	var awsConnectionData cloudConnector.AwsConnectionData

	validationErrors, marshalError := unmarshalRequestBody(writer, request, &awsConnectionData, cloudConnector.AwsConnectionDataSchema)
	if marshalError != nil {
		if marshalError.Error() == "http: request body too large" {
			log.WithFields(log.Fields{
				"Method": "AwsCloud",
				"Action": "post to aws",
				"Code":   http.StatusRequestEntityTooLarge,
			}).Error("Request Body too large")
			web.RespondError(ctx, writer, marshalError, http.StatusBadRequest)
			return nil
		}
		return marshalError
	}

	log.WithFields(log.Fields{
		"Method":  "AwsCloud",
		"TraceID": traceID,
	}).Debug()

	if len(validationErrors) > 0 {
		log.WithFields(log.Fields{
			"Method": "AwsCloud",
			"Action": "post to aws",
			"Code":   http.StatusBadRequest,
		}).Error("Validation errors")
		web.Respond(ctx, writer, validationErrors, http.StatusBadRequest)
		return nil
	}

	data, err := json.Marshal(awsConnectionData.Payload)
	if err != nil {
		log.WithFields(log.Fields{
			"Method": "AwsCloud",
			"Action": "post to aws",
			"Code":   http.StatusBadRequest,
		}).Error("Failed unmarshalling payload")
		web.Respond(ctx, writer, nil, http.StatusBadRequest)
		return nil
	}

	awsConfig := aws.Config{
		Region:      &awsConnectionData.Region,
		Credentials: credentials.NewStaticCredentials(awsConnectionData.AccessKeyID, awsConnectionData.SecretAccessKey, ""),
		LogLevel:    &logLevel,
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: awsConfig,
	})

	if err != nil {
		log.WithFields(log.Fields{
			"Method": "AwsCloud",
			"Action": "post to aws",
			"Code":   http.StatusBadRequest,
		}).Error("Failed creating AWS session")
		web.Respond(ctx, writer, nil, http.StatusBadRequest)
		return nil
	}

	s3Client := s3.New(sess, &awsConfig)

	if err := s3AddDataToBucket(s3Client, awsConnectionData.Bucket, data); err != nil {
		log.WithFields(log.Fields{
			"Method": "AwsCloud",
			"Action": "post to aws",
			"Code":   http.StatusBadRequest,
		}).Error("Failed creating AWS client")
		web.Respond(ctx, writer, err, http.StatusBadRequest)
		return err
	}

	mSuccess.Mark(1)
	web.Respond(ctx, writer, nil, statusCode)
	return nil
}

func s3AddDataToBucket(s3Client *s3.S3, bucketName string, data []byte) error {

	objectName := fmt.Sprintf("awsfile_%v", helper.UnixMilliNow())

	bucketExists := s3BucketExists(s3Client, bucketName)

	if bucketExists && !s3FileExists(s3Client, bucketName, objectName) {
		_, err := s3Client.PutObject(&s3.PutObjectInput{
			Bucket:             aws.String(bucketName),
			Key:                aws.String(objectName),
			ACL:                aws.String("private"),
			Body:               bytes.NewReader(data),
			ContentDisposition: aws.String("attachment"),
		})
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("bucket %s does not exist or file %s already exists", bucketName, objectName)
	}

	return nil
}

func s3FileExists(s3Client *s3.S3, bucketName string, objectName string) bool {

	_, err := s3Client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectName),
	})

	if err != nil {
		log.WithFields(log.Fields{
			"Method": "s3FileExists",
			"Action": "checking if aws file already exists",
			"Error":  err,
		}).Error("Failed creating AWS client")
	}

	return err == nil
}

func s3BucketExists(s3Client *s3.S3, bucketName string) bool {

	s3Buckets, err := s3Client.ListBuckets(nil)
	if err != nil {
		return false
	}

	for _, bucket := range s3Buckets.Buckets {
		if bucketName == *bucket.Name {
			return true
		}
	}
	return false
}

// Remove this linter comment once the unmarshal is used in another function
// nolint :unparam
func unmarshalRequestBody(writer http.ResponseWriter, request *http.Request, obj interface{}, schema string) ([]ErrReport, error) {
	// metrics
	mReadFullErr := metrics.GetOrRegisterGauge("CloudConnector.unmarshalRequestbody.ReadFull-Error", nil)
	mSchemaValidationErr := metrics.GetOrRegisterGauge("CloudConnector.unmarshalRequestbody.SchemaValidation-Error", nil)
	mSchemaValidationLatency := metrics.GetOrRegisterTimer("CloudConnector.unmarshalRequestbody.SchemaValdation-Latency", nil)
	mUnmarshalLatency := metrics.GetOrRegisterTimer("CloudConnector.unmarshalRequestbody.Unmarshal-Latency", nil)
	mUnmarshalErr := metrics.GetOrRegisterGauge("CloudConnector.unmarshalRequestbody.Unmarshal-Error", nil)

	body := make([]byte, request.ContentLength)
	_, err := io.ReadFull(request.Body, body)
	if err != nil {
		mReadFullErr.Update(1)
		return nil, err
	}

	schemaLoader := gojsonschema.NewStringLoader(schema)
	dataLoader := gojsonschema.NewBytesLoader(body)

	// Validating data with defined schema
	schemaValidationTimer := time.Now()
	result, err := gojsonschema.Validate(schemaLoader, dataLoader)
	if err != nil {
		mSchemaValidationErr.Update(1)
		return nil, errors.Wrap(err, "Error in validation schema")
	}
	mSchemaValidationLatency.Update(time.Since(schemaValidationTimer))

	if !result.Valid() {
		var errorRep ErrReport
		var errorSlice []ErrReport
		for _, err := range result.Errors() {
			// ignore extraneous "number_one_of" error
			if err.Type() == "number_one_of" {
				continue
			}
			errorRep.Description = err.Description()
			errorRep.Field = err.Field()
			errorRep.ErrorType = err.Type()
			errorRep.Value = err.Value()
			errorSlice = append(errorSlice, errorRep)
		}
		return errorSlice, nil
	}
	unMarshalTimer := time.Now()
	// unmarshal request body
	if err := json.Unmarshal(body, obj); err != nil {
		mUnmarshalErr.Update(1)
		return nil, errors.Wrap(err, "Unmarshalling error")
	}
	mUnmarshalLatency.Update(time.Since(unMarshalTimer))

	return nil, nil
}
