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

import "net/http"

// AwsConnectionData contains headers, and payload
type AwsConnectionData struct {
	AccessKeyID     string      `json:"accesskeyid" valid:"required"`
	SecretAccessKey string      `json:"secretaccesskey" valid:"required"`
	Region          string      `json:"region" valid:"required"`
	Bucket          string      `json:"bucket" valid:"required"`
	Payload         interface{} `json:"payload" valid:"optional"`
}

type WebhookResponse struct {
	Body       []byte
	StatusCode int
	Header     http.Header
}

// Webhook contains webhook address, headers, method, authentication method, and payload
type Webhook struct {
	Header  http.Header `json:"header" valid:"optional"`
	Method  string      `json:"method" valid:"required"`
	URL     string      `json:"url" valid:"required,url"`
	Auth    Auth        `json:"auth" valid:"optional"`
	Payload interface{} `json:"payload" valid:"optional"`
	IsAsync bool        `json:"isasync" valid:"required"`
}

// Auth contains the type and the endpoint of authentication
type Auth struct {
	AuthType string `json:"authtype" valid:"length(0|1024)"`
	Endpoint string `json:"endpoint" valid:"url"`
	Data     string `json:"data" valid:"length(0|1024)"`
}

// WebhookSchema defines Webhook schema for input validation
const WebhookSchema = `
{
	"$ref": "#/definitions/Webhook",
	"definitions": {
			"Auth": {
					"properties": {
							"authtype": {
									"type": "string",
									"minLength": 0,
									"maxLength": 1024
							},
							"data": {
									"type": "string",
									"minLength": 0,
									"maxLength": 1024
							},
							"endpoint": {
									"type": "string",
									"minLength": 0,
									"maxLength": 1024
							}
					},
					"additionalProperties": false,
					"type": "object"
			},
			"Header": {
				"type": "object",
				"additionalProperties": {"$ref": "#/definitions/StringSlice"}
			},
			"StringSlice": {
				"data": {
					"type": "array",
					"minItems": 0,
					"maxItems": 100,
					"items": {
					  "type": "string"
					}
				  },
				"additionalProperties": false
			},
			"Webhook": {
					"required": [
							"url"
					],
					"properties": {
							"auth": {
									"$ref": "#/definitions/Auth"
							},
							"payload": {},
							"url": {
									"type": "string",
									"format": "uri"
							},
							"header": {
								"oneOf": [
									{"type": "null"},
									{"$ref": "#/definitions/Header"}
								]
							},
							"method": {
								"type": ["string", "null"],
								"enum": ["POST", "GET"]
							},
							"isasync": {
								"type": "boolean"
							}
					},
					"additionalProperties": false,
					"type": "object"
			}
	}
}
`

// AwsConnectionDataSchema defines schema for input validation
const AwsConnectionDataSchema = `
{
	"$ref": "#/definitions/AwsConnectionData",
	"definitions": {
			"AwsConnectionData" : {
				"required": [
					"accesskeyid",
					"secretaccesskey",
					"bucket"
				],
				"properties": {
					"accesskeyid": {
						"type": "string",
						"minLength": 1,
						"maxLength": 1024
					},
					"secretaccesskey": {
						"type": "string",
						"minLength": 1,
						"maxLength": 1024
					},
					"bucket": {
						"type": "string",
						"minLength": 1,
						"maxLength": 1024
					},
					"region": {
						"type": "string",
						"minLength": 1,
						"maxLength": 1024
					},
					"payload": {}
				},
				"additionalProperties": false,
				"type": "object"
			}
	}
}
`
