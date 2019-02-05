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

package routes

import (
	"github.com/gorilla/mux"

	"github.impcloud.net/Responsive-Retail-Core/cloud-connector-service/app/routes/handlers"
	"github.impcloud.net/Responsive-Retail-Core/cloud-connector-service/pkg/middlewares"
	"github.impcloud.net/Responsive-Retail-Core/cloud-connector-service/pkg/web"
)

// Route struct holds attributes to declare routes
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc web.Handler
}

// NewRouter creates the routes for GET and POST
func NewRouter() *mux.Router {

	cloudConnector := handlers.CloudConnector{}

	var routes = []Route{
		// swagger:operation GET / default Healthcheck
		//
		// Healthcheck Endpoint
		//
		// Endpoint that is used to determine if the application is ready to take web requests
		//
		// ---
		// consumes:
		// - application/json
		//
		// produces:
		// - application/json
		//
		// schemes:
		// - http
		//
		// responses:
		//   '200':
		//     description: OK
		//
		{
			"Index",
			"GET",
			"/",
			cloudConnector.Index,
		},
		// swagger:operation POST /callwebhook webhooks callwebhook
		//
		// Send Notification
		//
		// This API call is used to request Responsive Retail to notify the enterprise system when specific events occur in the store. The notifications take place by a web callback, typically referred to as a web hook. A notification request must include the following information:
		//
		//     URL - (required) The call back URL. Responsive Retail must be able to post data to this URL.
		//
		//	   Method - (required) The http method to be ran on the webhook(Allowed methods: GET or POST)
		//
		//	   Header - (optional) The header for the webhook
		//
		//	   IsAsync - (required) Whether the cloud call should be made sync or async. To be notified of errors connecting to the cloud use IsAsync:true.GET HTTP verb ignores IsAsync flag.
		//
		//     Auth - (optional) Authentication settings used
		//       - AuthType - The Authentication method defined by the webhook (ex. OAuth2)
		//       - Endpoint - The Authentication endpoint if it differs from the webhook server
		//       - Data - The Authentication data required by the authentication server
		//
		//     Payload - (optional) The payload intended for the destination webhook. This is typically a json object or map of values.
		//
		//     Expected formatting of JSON input (as an example):<br><br>
		//
		//```
		// {
		// 	"url": "string",
		//	"method": "string",
		// 	"auth": {
		// 	  "authtype": "string",
		// 		"endpoint": "string",
		// 		"data":     "string"
		// 	},
		// 	"isasync": 		boolean,
		// 	"payload": "interface"
		//  }
		//  ```
		// ---
		// consumes:
		// - application/json
		//
		// produces:
		// - application/json
		//
		// schemes:
		// - http
		//
		// responses:
		//   '201':
		//      description: OK
		//   '400':
		//      description: ErrReport error
		//      schema:
		//        type: array
		//        items:
		//         "$ref": "#/definitions/ErrReport"
		//   '404':
		//      description: Not Found
		//   '500':
		//      description: Internal server error
		//
		{
			"CallWebhook",
			"POST",
			"/callwebhook",
			cloudConnector.CallWebhook,
		},
		// swagger:operation POST /aws-cloud/data awsclouddata AwsCloud
		//
		// Upload to AWS cloud
		//
		// This API call is used to upload data to an S3 bucket by passing the access key id, secret access key, region, and bucket name in the request along with the payload.
		//
		//     AccessKeyID - (required) AWS access key ID
		//
		//     SecretAccessKey - (required) AWS secret access key
		//
		//     Region - (required) AWS Region
		//
		//	   Bucket - (required) The bucket path/name
		//
		//     Payload - (optional) The payload intended for the destination. This is typically a json object or map of values.
		//
		//     Expected formatting of JSON input (as an example):<br><br>
		//
		//```
		//{
		//	"accesskeyid": "<ACCESS KEY ID>",
		//	"secretaccesskey": "<SECRET ACCESS KEY>",
		//	"bucket": "<BUCKET>",
		//	"region" : "<REGION>",
		//	"payload" : "data"
		//}
		//  ```
		// ---
		// consumes:
		// - application/json
		//
		// produces:
		// - application/json
		//
		// schemes:
		// - http
		//
		// responses:
		//   '200':
		//      description: OK
		//   '400':
		//      description: ErrReport error
		//      schema:
		//        type: array
		//        items:
		//         "$ref": "#/definitions/ErrReport"
		//   '500':
		//      description: Internal server error
		//
		{
			"AwsCloud",
			"POST",
			"/aws-cloud/data",
			cloudConnector.AwsCloud,
		},
	}

	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {

		handler := route.HandlerFunc
		handler = middlewares.Recover(handler)
		handler = middlewares.Logger(handler)
		handler = middlewares.Bodylimiter(handler)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}
