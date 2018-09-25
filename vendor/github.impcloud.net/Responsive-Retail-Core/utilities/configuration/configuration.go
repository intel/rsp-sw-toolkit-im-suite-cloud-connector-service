package configuration

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"encoding/json"
	consulApi "github.com/hashicorp/consul/api"
)

type Configuration struct {
	isNestedConfig       bool
	parsedJson           map[string]interface{}
	sectionName          string
	configChangeCallback func([]ChangeDetails)
}

type ChangeType uint

const (
	Invalid ChangeType = iota
	Added
	Updated
	Deleted

	TimeStampFilePermissions = 666
)

type ChangeDetails struct {
	Name      string
	Value     interface{}
	Operation ChangeType
}

func NewSectionedConfiguration(sectionName string) (*Configuration, error) {
	config := Configuration{}
	config.sectionName = sectionName

	err := config.loadConfiguration()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func NewConfiguration() (*Configuration, error) {
	config := Configuration{}

	_, executablePath, _, ok := runtime.Caller(2)
	if ok {
		config.sectionName = path.Base(path.Dir(executablePath))
	}

	err := config.loadConfiguration()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (config *Configuration) SetConfigChangeCallback(callback func([]ChangeDetails)) {
	config.configChangeCallback = callback
}

func (config *Configuration) Load(path string) error {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(file, &config.parsedJson)
	return err
}

func (config *Configuration) GetParsedJson() map[string]interface{} {
	return config.parsedJson
}

func (config *Configuration) GetNestedJSON(path string) (map[string]interface{}, error) {
	config.isNestedConfig = true
	if !config.pathExistsInConfigFile(path) {
		return nil, fmt.Errorf("%s not found", path)
	}

	item := config.getValue(path)
	value, ok := item.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unable to convert value for '%s' to a map[string]interface: Value='%v'", path, item)
	}
	config.isNestedConfig = false
	return value, nil

}

func (config *Configuration) GetNestedMapOfMapString(path string) (map[string]map[string]string, error) {
	mapOfInterface, err := config.GetNestedJSON(path)
	if err != nil {
		return nil, err
	}
	nestedKeyValue := make(map[string]map[string]string)
	for key, value := range mapOfInterface {
		switch value.(type) {
		case map[string]interface{}:
			nestedValueToString, err := interfaceToString(value.(map[string]interface{}))
			if err != nil {
				return nil, err
			}
			nestedKeyValue[key] = nestedValueToString
		default:
			return nil, fmt.Errorf("unexpected type found %s for value='%v' while conversion", reflect.TypeOf(value), value)
		}

	}
	return nestedKeyValue, nil
}

func interfaceToString(values map[string]interface{}) (map[string]string, error) {
	mapOfString := make(map[string]string)
	for key, value := range values {
		valType := reflect.TypeOf(value)
		fmt.Print(valType)
		switch value.(type) {
		case float64:
			mapOfString[key] = strconv.FormatFloat(value.(float64), 'f', -1, 64)
		case bool:
			mapOfString[key] = strconv.FormatBool(value.(bool))
		case string:
			mapOfString[key] = value.(string)
		default:
			return nil, fmt.Errorf("unexpected type found %s for value='%v' during conversion. currently accepts only float64,bool and string", reflect.TypeOf(value), value)
		}

	}
	return mapOfString, nil
}

func (config *Configuration) GetString(path string) (string, error) {
	if !config.pathExistsInConfigFile(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return "", fmt.Errorf("%s not found", path)
		}

		return value, nil
	}

	item := config.getValue(path)

	value, ok := item.(string)
	if !ok {
		return "", fmt.Errorf("unable to convert value for '%s' to a string: Value='%v'", path, item)
	}

	return value, nil
}

func (config *Configuration) GetInt(path string) (int, error) {
	if !config.pathExistsInConfigFile(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return 0, fmt.Errorf("%s not found", path)
		}

		intValue, err := strconv.Atoi(value)
		if err != nil {
			return 0, fmt.Errorf("unable to convert value for '%s' to an int: Value='%v'", path, intValue)
		}

		return intValue, nil
	}

	item := config.getValue(path)

	value, ok := item.(float64)
	if !ok {
		return 0, fmt.Errorf("unable to convert value for '%s' to an int: Value='%v'", path, item)
	}

	return int(value), nil
}

func (config *Configuration) GetFloat(path string) (float64, error) {
	if !config.pathExistsInConfigFile(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return 0, fmt.Errorf("%s not found", path)
		}

		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, fmt.Errorf("unable to convert value for '%s' to an int: Value='%v'", path, value)
		}

		return floatValue, nil
	}

	item := config.getValue(path)

	value, ok := item.(float64)
	if !ok {
		return 0, fmt.Errorf("unable to convert value for '%s' to an int: Value='%v'", path, item)
	}

	return value, nil
}

func (config *Configuration) GetBool(path string) (bool, error) {
	if !config.pathExistsInConfigFile(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return false, fmt.Errorf("%s not found", path)
		}

		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return false, fmt.Errorf("unable to convert value for '%s' to a bool: Value='%v'", path, boolValue)
		}

		return boolValue, nil
	}

	item := config.getValue(path)

	value, ok := item.(bool)
	if !ok {
		return false, fmt.Errorf("unable to convert value for '%s' to a bool: Value='%v'", path, item)
	}

	return value, nil
}

func (config *Configuration) GetStringSlice(path string) ([]string, error) {
	if !config.pathExistsInConfigFile(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return nil, fmt.Errorf("%s not found", path)
		}

		value = strings.Replace(value, "[", "", 1)
		value = strings.Replace(value, "]", "", 1)

		slice := strings.Split(value, ",")
		var resultSlice []string
		for _, item := range slice {
			resultSlice = append(resultSlice, strings.Trim(item, " "))
		}

		return resultSlice, nil
	}

	item := config.getValue(path)
	slice := item.([]interface{})

	var stringSlice []string
	for _, sliceItem := range slice {
		value, ok := sliceItem.(string)
		if !ok {
			return nil, fmt.Errorf("unable to convert a value for '%s' to a string: Value='%v'", path, sliceItem)

		}
		stringSlice = append(stringSlice, value)
	}

	return stringSlice, nil
}

func (config *Configuration) getValue(path string) interface{} {
	if config.parsedJson == nil {
		return nil
	}

	if config.sectionName != "" {
		sectionedPath := fmt.Sprintf("%s.%s", config.sectionName, path)
		value := config.getValueFromJson(sectionedPath)
		if value != nil {
			return value
		}
	}

	value := config.getValueFromJson(path)
	return value
}

func (config *Configuration) getValueFromJson(path string) interface{} {
	pathNodes := strings.Split(path, ".")
	if len(pathNodes) == 0 {
		return nil
	}

	var ok bool
	var value interface{}
	jsonNodes := config.parsedJson
	for _, node := range pathNodes {
		if jsonNodes[node] == nil {
			return nil
		}

		item := jsonNodes[node]
		jsonNodes, ok = item.(map[string]interface{})
		if ok && !config.isNestedConfig {
			continue
		}

		if config.sectionName == node {
			continue
		}

		value = item
		break
	}

	return value
}

func (config *Configuration) pathExistsInConfigFile(path string) bool {
	if config.sectionName != "" {
		sectionPath := fmt.Sprintf("%s.%s", config.sectionName, path)
		if config.getValue(sectionPath) != nil {
			return true
		}
	}

	if config.getValue(path) != nil {
		return true
	}

	return false
}

func (config *Configuration) loadConfiguration() error {
	_, filename, _, ok := runtime.Caller(2)
	if !ok {
		log.Print("No caller information")
	}

	absolutePath := path.Join(path.Dir(filename), "configuration.json")

	// By default load local configuration file if it exists
	if _, err := os.Stat(absolutePath); err != nil {
		absolutePath, ok = os.LookupEnv("runtimeConfigPath")
		if !ok {
			absolutePath = "/run/secrets/configuration.json"
		}
		if _, err := os.Stat(absolutePath); err != nil {
			absolutePath = ""
		}
	}

	// Attempt to get configuration from Consul service. If can not then continue loading from file.
	if config.loadFromConsul(absolutePath) {
		return nil
	}

	if absolutePath != "" {
		err := config.Load(absolutePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (config *Configuration) loadFromConsul(configFilePath string) bool {
	consulConfigKey, ok := os.LookupEnv("consulConfigKey")
	if !ok {
		log.Print("consulConfigKey environment variable not set, using local configuration file")
		return false
	}

	consulUrl, ok := os.LookupEnv("consulUrl")
	if !ok {
		log.Print("consulUrl environment variable not set, using local configuration file")
		return false
	}

	consul, err := consulApi.NewClient(&consulApi.Config{Address: consulUrl})
	if err != nil {
		log.Printf("not able to communicate with Consul service: %s, using local configuration file", err.Error())
		return false
	}

	keyValuePair := checkAndUpdateFromLocal(consul, consulConfigKey, configFilePath)
	if keyValuePair == nil {
		return false
	}

	if err = json.Unmarshal(keyValuePair.Value, &config.parsedJson); err != nil {
		log.Printf("error marshaling JSON configuration received from/pushed to Consul Service: %s, using local configuration file", err.Error())
		return false
	}

	// Now that we know we are using Consul service, we need to create a watch on the configuration for changes.
	watcher, err := NewWatcher(consul, consulConfigKey)
	if err != nil {
		log.Printf("error creating watcher for chnages to value for %s: %s", consulConfigKey, err.Error())
		return true // true because configuration came from consul, but unable to watch for changes.
	}

	err = watcher.Start(config.processConfigurationChanged)

	if err != nil {
		log.Printf("error starting watcher for chnages to value for %s: %s", consulConfigKey, err.Error())
		return true // true because configuration came from consul, but unable to watch for changes.
	}

	return true
}

func (config *Configuration) applyConfigurationJson(jsonBytes []byte) error {

	// Clear parsed JSON so start fresh since old deleted fields don't get removed.
	config.parsedJson = map[string]interface{}{}

	return json.Unmarshal(jsonBytes, &config.parsedJson)
}

func (config *Configuration) processConfigurationChanged(configurationJson []byte) {

	// Have to get these before applying new configuration JSON for comparing later
	previousGlobalSection, previousTargetSection := config.getGlobalAndTargetSections()

	// This saves the new configuration
	if err := config.applyConfigurationJson(configurationJson); err != nil {
		log.Printf("error marshaling JSON configuration received from change Consul watcher: %s", err.Error())
	}

	// if callback not set there is no need to continue the processing looking if anything changed.
	if config.configChangeCallback == nil {
		return
	}

	var changedList []ChangeDetails
	newGlobalSection, newTargetSection := config.getGlobalAndTargetSections()

	changedList = config.getChanges(changedList, previousGlobalSection, newGlobalSection, false)
	changedList = config.getChanges(changedList, previousTargetSection, newTargetSection, true)

	if len(changedList) > 0 {
		config.configChangeCallback(changedList)
	}
}

func (config *Configuration) getGlobalAndTargetSections() (map[string]interface{}, map[string]interface{}) {

	globalSection := make(map[string]interface{})
	targetSection := make(map[string]interface{})

	for configItemName, configItemValue := range config.parsedJson {
		configValueDetail := reflect.ValueOf(configItemValue)
		kind := configValueDetail.Kind()

		if kind == reflect.Map {
			if configItemName == config.sectionName {
				for _, key := range configValueDetail.MapKeys() {
					valueFromMap := configValueDetail.MapIndex(key)
					name := key.Interface().(string)
					value := valueFromMap.Elem().Interface()

					targetSection[name] = value
				}
			}
		} else {
			globalSection[configItemName] = configItemValue
		}
	}

	return globalSection, targetSection
}

func (config *Configuration) getChanges(changedList []ChangeDetails, previousSection map[string]interface{}, newSection map[string]interface{}, isTargetSection bool) []ChangeDetails {
	for itemName, itemValue := range previousSection {
		name := itemName
		if isTargetSection {
			name = config.sectionName + "." + itemName
		}

		if newSection[itemName] == nil {
			details := ChangeDetails{
				Name:      name,
				Value:     nil,
				Operation: Deleted,
			}
			changedList = append(changedList, details)
			continue
		}

		if itemValue != newSection[itemName] {
			details := ChangeDetails{
				Name:      name,
				Value:     newSection[itemName],
				Operation: Updated,
			}
			changedList = append(changedList, details)
		}
	}

	for itemName, itemValue := range newSection {
		name := itemName
		if isTargetSection {
			name = config.sectionName + "." + itemName
		}

		if previousSection[itemName] == nil {
			details := ChangeDetails{
				Name:      name,
				Value:     itemValue,
				Operation: Added,
			}
			changedList = append(changedList, details)
		}
	}

	return changedList
}

func checkAndUpdateFromLocal(consul *consulApi.Client, consulConfigKey string, configFilePath string) *consulApi.KVPair {
	keyValuePair, _, err := consul.KV().Get(consulConfigKey, nil)
	if err != nil {
		log.Printf("error attempting to get '%s' value from Consul service: %s, using local configuration file", consulConfigKey, err.Error())
		return nil
	}

	timestampFile := configFilePath + ".timestamp"

	localFileChanged, localConfigFileStats := localConfigurationChanged(configFilePath, timestampFile)
	if localConfigFileStats == nil {
		return nil
	}

	if keyValuePair == nil || localFileChanged {
		log.Printf("%s not found in Consul Service or local file changed since last pushed. Attempting to push local default configuration to Consul Service", consulConfigKey)

		// Load the local default configuration file in order to push it to Consul.
		fileBytes, readErr := ioutil.ReadFile(configFilePath)
		if readErr != nil {
			log.Printf("error attempting to load default configuration inorder to push to Consul Service: %s", err.Error())
			return nil
		}

		keyValuePair = &consulApi.KVPair{
			Key:   consulConfigKey,
			Value: fileBytes,
		}

		// Using local configuration to Consul so need to save the timestamp for the file so we know next time if it has changed.
		timestamp := fmt.Sprintf("%d", uint64(localConfigFileStats.ModTime().UnixNano()))
		writeErr := ioutil.WriteFile(timestampFile, []byte(timestamp), TimeStampFilePermissions)
		if writeErr != nil {
			log.Printf("error saving timestamp of local configuration to file '%s': %s", timestampFile, err.Error())
			return nil
		}

		if _, putErr := consul.KV().Put(keyValuePair, nil); putErr != nil {
			log.Printf("error pushing default configuration to '%s' value in Consul Service: %s", consulConfigKey, err.Error())
			return nil
		}
	}

	return keyValuePair
}

func localConfigurationChanged(configFilePath string, timestampFile string) (bool, os.FileInfo) {

	fileStats, statErr := os.Stat(configFilePath)
	if statErr != nil {
		log.Printf("error getting timestamp of local configuration to file '%s': %s", timestampFile, statErr.Error())
		// can't get the timestamp of current file, so assume changed.
		return true, nil
	}

	_, existsErr := os.Stat(timestampFile)
	if os.IsNotExist(existsErr) {
		// file doesn't exist, so can't compare, so file must be new. i.e. changed
		return true, fileStats
	}

	timestampBytes, readErr := ioutil.ReadFile(timestampFile)
	if readErr != nil {
		log.Printf("error reading timestamp of last local configuration to file '%s': %s", timestampFile, readErr.Error())
		// can't get the timestamp of last file used, so assume changed.
		return true, fileStats
	}

	originalTimeStamp := string(timestampBytes)
	newTimeStamp := fmt.Sprintf("%d", fileStats.ModTime().UnixNano())
	return newTimeStamp != originalTimeStamp, fileStats
}
