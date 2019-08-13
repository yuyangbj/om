package commands_test

import (
	"github.com/jessevdk/go-flags"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/om/interpolate"
	"testing"
)

func TestCommands(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "commands")
}

func executeCommand(command interface{}, args []string) error {
	_, err := interpolate.FromConfigFile(command, args)
	if err != nil {
		return err
	}

	return command.(flags.Commander).Execute([]string{})
}
