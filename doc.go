// Cloud Connector Service.
//
// __Cloud Connector Service Description__
//
// The purpose of this service is to provide a way for applications to notify a given/registered webhook of various events that occur in the system.
//
//	__Configuration Values__
// <blockquote>Cloud Connector service configuration is is split between values set in a configuration file and those set as environment variables in the compose file. The configuration file is expected to be contained in a docker secret for production deployments, but can be on a docker volume for validation and development.
//    <blockquote><b>Configuration file values</b>
//       <blockquote>•<b> serviceName</b> - Runtime name of the service.</blockquote>
//       <blockquote>•<b> loggingLevel</b> - Logging level to use: "info" (default) or "debug" (verbose).</blockquote>
//       <blockquote>•<b> telemetryEndpoint</b> - URL of the telemetry service receiving the metrics from the service.</blockquote>
//       <blockquote>•<b> telemetryDataStoreName</b> - Name of the data store in the telemetry service to store the metrics.</blockquote>
//       <blockquote>•<b> port</b> - Port to run the service's HTTP Server on.</blockquote>
//       <blockquote>•<b> httpsProxyURL</b> - URL of the proxy server  </blockquote>
//    </blockquote>
//    <blockquote><b>Compose file environment variable values</b>
//       <blockquote>•<b> runtimeConfigPath</b> - Path to the configuration file to use at runtime.</blockquote>
//    </blockquote>
//
//    <pre><b>Example configuration file json
//    &#9{
//    &#9&#9"serviceName": "RRP - Cloud Connector",
//    &#9&#9"loggingLevel": "debug",
//    &#9&#9"telemetryEndpoint": "http://166.130.9.122:8000",
//    &#9&#9"telemetryDataStoreName" : "Store105",
//    &#9&#9"port": "8080",
//    &#9&#9"httpsProxyURL" : http://proxy-us.intel.com:912
//    &#9}
//    </b></pre>
//    <pre><b>Example environment variables in compose file
//    &#9runtimeConfigPath: "/data/configs/cloudConnector.json"
//    </b></pre>
//
// __Known services that depend upon this service:__
// ○ Rules
// ○ RFID-Alert
//
//
//     Schemes: https
//     Host: cloudConnector:8080
//     BasePath: /
//     Version: 1.0.0
//     Contact: Intel RRP Support<rrp@intel.com>
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
// swagger:meta
package main

import (
	"github.impcloud.net/Responsive-Retail-Core/cloud-connector-service/app/cloudConnector"
)

// swagger:parameters callwebhook
type WebhookWrapper struct {
	//in: body

	WebhookObj cloudConnector.Webhook `json:"webhook"`
}
