/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package config

import (
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/configuration"
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
func InitConfig(configChangedCallback func([]configuration.ChangeDetails)) error {
	AppConfig = variables{}

	var err error

	config, err := configuration.NewConfiguration()
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables")
	}
	config.SetConfigChangeCallback(configChangedCallback)

	AppConfig.ServiceName, err = config.GetString("serviceName")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables")
	}

	AppConfig.Port, err = config.GetString("port")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables")
	}
	AppConfig.TelemetryEndpoint, err = config.GetString("telemetryEndpoint")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables: %s", err.Error())
	}

	AppConfig.TelemetryDataStoreName, err = config.GetString("telemetryDataStoreName")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables")
	}

	AppConfig.HttpsProxyURL, err = config.GetString("httpsProxyURL")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables")
	}

	// Set "debug" for development purposes. Nil for Production.
	AppConfig.LoggingLevel, err = config.GetString("loggingLevel")
	if err != nil {
		return errors.Wrapf(err, "Unable to load config variables")
	}

	return nil
}
