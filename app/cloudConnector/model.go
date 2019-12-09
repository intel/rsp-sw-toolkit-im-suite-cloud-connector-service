/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package cloudConnector

import (
	"net/http"
)

// AwsConnectionData contains headers, and payload
type AwsConnectionData struct {
	AccessKeyID     string      `json:"accesskeyid" valid:"required"`
	SecretAccessKey string      `json:"secretaccesskey" valid:"required"`
	Region          string      `json:"region" valid:"required"`
	Bucket          string      `json:"bucket" valid:"required"`
	Payload         interface{} `json:"payload" valid:"optional"`
}

type WebhookResponse struct {
	StatusCode int         `json:"statuscode"`
	Header     http.Header `json:"header"`
	Body       []byte      `json:"body" valid:"optional"`
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
