package commands_test

import (
	"github.com/jessevdk/go-flags"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/om/interpolate"
	"os"
	"testing"
)

func TestCommands(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "commands")
}

func executeCommand(command interface{}, args []string) error {
	parser := flags.NewParser(command, flags.HelpFlag|flags.PassDoubleDash)
	_, parseErr := parser.ParseArgs(args)
	if ok, configErr := interpolate.FromConfigFile(command, os.Environ); ok {
		if configErr != nil {
			return configErr
		}
		return command.(flags.Commander).Execute([]string{})
	}

	if parseErr != nil {
		return parseErr
	}

	return command.(flags.Commander).Execute([]string{})
}
