package interpolate

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v2"
	"reflect"
	"strconv"
)

// Load the config file, (optionally) load the vars file, vars env as well
// To use this function, `Config` field must be defined in the command struct being passed in.
// To load vars, VarsFile and/or VarsEnv must exist in the command struct being passed in.
// If VarsEnv is used, envFunc must be defined instead of nil
func FromConfigFile(config interface{}, envFunc func() []string, args []string) error {
	commandValue := reflect.ValueOf(config).Elem()
	configFileField := commandValue.FieldByName("ConfigFile")
	if !configFileField.IsValid() {
		commandValue = commandValue.FieldByName("Options")
		if !commandValue.IsValid() {
			return nil
		}

		configFileField = commandValue.FieldByName("ConfigFile")
		if !configFileField.IsValid() {
			return nil
		}
	}

	configFile := configFileField.String()
	if configFile == "" {
		return nil
	}

	varsFileField := commandValue.FieldByName("VarsFile")
	varsEnvField := commandValue.FieldByName("VarsEnv")
	cmdVarsField := commandValue.FieldByName("Vars")

	var (
		varsField []string
		varsEnv   []string
		cmdVars   []string
		ok        bool
		options   map[string]interface{}
		contents  []byte
	)

	if varsFileField.IsValid() {
		if varsField, ok = varsFileField.Interface().([]string); !ok {
			return fmt.Errorf("expect VarsFile field to be a `[]string`, found %s", varsEnvField.Type())
		}
	}

	if cmdVarsField.IsValid() {
		if cmdVars, ok = cmdVarsField.Interface().([]string); !ok {
			return fmt.Errorf("expect Vars field to be a `[]string`, found %s", cmdVarsField.Type())
		}
	}

	if varsEnvField.IsValid() {
		if varsEnv, ok = varsEnvField.Interface().([]string); !ok {
			return fmt.Errorf("expect VarsEnv field to be a `[]string`, found %s", varsEnvField.Type())
		}
	}

	contents, err := Execute(Options{
		TemplateFile:  configFile,
		VarsEnvs:      varsEnv,
		VarsFiles:     varsField,
		Vars:          cmdVars,
		EnvironFunc:   envFunc,
		OpsFiles:      nil,
		ExpectAllKeys: true,
	})
	if err != nil {
		return fmt.Errorf("could not load the config file: %s", err)
	}

	err = yaml.Unmarshal(contents, &options)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config file %s: %s", configFile, err)
	}

	var fileArgs []string
	for key, value := range options {
		switch convertedValue := value.(type) {
		case []interface{}:
			for _, v := range convertedValue {
				fileArgs = append(fileArgs, fmt.Sprintf("--%s=%s", key, v))
			}
		case bool:
			fileArgs = append(fileArgs, fmt.Sprintf("--%s=%s", key, strconv.FormatBool(convertedValue)))
		default:
			fileArgs = append(fileArgs, fmt.Sprintf("--%s=%s", key, value))
		}

	}
	if len(args) > 0 {
		fileArgs = append(fileArgs, args...)
	}
	_, err = flags.ParseArgs(config, fileArgs)
	return err
}
