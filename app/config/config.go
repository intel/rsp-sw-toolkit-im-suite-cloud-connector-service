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

package config

import (
	"github.com/pkg/errors"
	"github.impcloud.net/Responsive-Retail-Inventory/utilities/configuration"
)

type (
	variables struct {
		ServiceName            string
		LoggingLevel           string
		HttpsProxyURL          string
		Port                   string
		TelemetryEndpoint      string
		TelemetryDataStoreName string
	}
)

// AppConfig exports all config variables
var AppConfig variables

// InitConfig loads application variables
func InitConfig() error {
	AppConfig = variables{}

	var err error

	config, err := configuration.NewConfiguration()
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.ServiceName, err = config.GetString("serviceName")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.Port, err = config.GetString("port")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}
	AppConfig.TelemetryEndpoint, err = config.GetString("telemetryEndpoint")
	if err != nil {
		// since telemetryEndpoint is optional, it's ok to get an error on it
		err = nil
	}

	AppConfig.TelemetryDataStoreName, err = config.GetString("telemetryDataStoreName")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.HttpsProxyURL, err = config.GetString("httpsProxyURL")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	// Set "debug" for development purposes. Nil for Production.
	AppConfig.LoggingLevel, err = config.GetString("loggingLevel")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	return nil
}
