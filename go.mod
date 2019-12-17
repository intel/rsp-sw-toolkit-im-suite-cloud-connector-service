module github.com/intel/rsp-sw-toolkit-im-suite-cloud-connector-service

go 1.12

require (
	github.com/aws/aws-sdk-go v1.19.27
	github.com/gorilla/mux v1.7.1
	github.com/intel/rsp-sw-toolkit-im-suite-gojsonschema v1.0.0
	github.com/intel/rsp-sw-toolkit-im-suite-utilities v0.1.0
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.1
)

replace github.com/intel/rsp-sw-toolkit-im-suite-utilities => ../../utilities
