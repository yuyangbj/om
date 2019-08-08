package interpolate_test

import (
	"github.com/pivotal-cf/om/interpolate"
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestInterpolate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Interpolate Suite")
}

var _ = Describe("Execute", func() {
	It("errors when the template file does not exist", func() {
		_, err := interpolate.Execute(interpolate.Options{
			TemplateFile: "unknown.txt",
		})
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("could not read file (unknown.txt): open unknown.txt: no such file or directory"))
	})

	When("path is specified", func() {
		It("returns that specific value", func() {
			contents, err := interpolate.Execute(interpolate.Options{
				TemplateFile: writeFile(`{name: Bob}`),
				Path:         "/name",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal("Bob\n"))
		})

		It("errors with an invalid path", func() {
			_, err := interpolate.Execute(interpolate.Options{
				TemplateFile: writeFile(`{name: Bob}`),
				Path:         "/age",
			})
			Expect(err).To(HaveOccurred())
		})
	})

	When("environment variables are used", func() {
		It("supports variables with a prefix", func() {
			contents, err := interpolate.Execute(interpolate.Options{
				TemplateFile: writeFile(`{name: ((name))}`),
				VarsEnvs:     []string{"PREFIX"},
				EnvironFunc: func() []string {
					return []string{"PREFIX_name=Bob"}
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(MatchYAML(`name: Bob`))
		})

		It("errors with an invalid environment variable definition", func() {
			_, err := interpolate.Execute(interpolate.Options{
				TemplateFile: writeFile(`{name: ((name))}`),
				VarsEnvs:     []string{"PREFIX"},
				EnvironFunc: func() []string {
					return []string{"PREFIX_name"}
				},
			})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("Expected environment variable to be key-value pair"))
		})

		It("errors when the environment variable is invalid YAML", func() {
			_, err := interpolate.Execute(interpolate.Options{
				TemplateFile: writeFile(`{name: ((name))}`),
				VarsEnvs:     []string{"PREFIX"},
				EnvironFunc: func() []string {
					return []string{"PREFIX_name={]"}
				},
			})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(`Could not deserialize YAML from environment variable "PREFIX_name"`))
		})

		It("modifies a number if it has been quoted", func() {
			contents, err := interpolate.Execute(interpolate.Options{
				TemplateFile: writeFile(`{age: ((age))}`),
				VarsEnvs:     []string{"PREFIX"},
				EnvironFunc: func() []string {
					return []string{`PREFIX_age="123"`}
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(MatchYAML(`age: "123"`))
		})

		It("handles multiline environment variables", func() {
			contents, err := interpolate.Execute(interpolate.Options{
				TemplateFile: writeFile(`{name: ((name))}`),
				VarsEnvs:     []string{"PREFIX"},
				EnvironFunc: func() []string {
					return []string{"PREFIX_name=some\nmulti\nline\nvalue"}
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(MatchYAML(`name: "some\nmulti\nline\nvalue"`))
		})

		It("handles environment variables that are objects", func() {
			contents, err := interpolate.Execute(interpolate.Options{
				TemplateFile: writeFile(`{name: ((name))}`),
				VarsEnvs:     []string{"PREFIX"},
				EnvironFunc: func() []string {
					return []string{"PREFIX_name={some: value}"}
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(MatchYAML(`name: {some: value}`))
		})

		It("allows vars files to override environment variables", func() {
			contents, err := interpolate.Execute(interpolate.Options{
				TemplateFile: writeFile(`{age: ((age))}`),
				VarsFiles:    []string{writeFile(`age: 456`)},
				VarsEnvs:     []string{"PREFIX"},
				EnvironFunc: func() []string {
					return []string{`PREFIX_age="123"`}
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(MatchYAML(`age: 456`))
		})
	})

	When("vars files are specified", func() {
		It("supports loading vars from files", func() {
			contents, err := interpolate.Execute(interpolate.Options{
				TemplateFile: writeFile(`{name: ((username))}`),
				VarsFiles:    []string{writeFile(`username: Bob`)},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(MatchYAML(`name: Bob`))
		})

		It("applies vars files on left to right precedence", func() {
			contents, err := interpolate.Execute(interpolate.Options{
				TemplateFile: writeFile(`{name: ((username))}`),
				VarsFiles: []string{
					writeFile(`username: Bob`),
					writeFile(`username: Susie`),
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(MatchYAML(`name: Susie`))
		})

		It("allows individual vars to override vars files", func() {
			contents, err := interpolate.Execute(interpolate.Options{
				TemplateFile: writeFile(`{name: ((username))}`),
				VarsFiles:    []string{writeFile(`username: Bob`)},
				Vars:         []string{"username=Susie"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(MatchYAML(`name: Susie`))
		})
	})

	When("ops files are specified", func() {
		It("supports ops file modifications", func() {
			contents, err := interpolate.Execute(interpolate.Options{
				TemplateFile: writeFile(`{name: Susie}`),
				OpsFiles:     []string{writeFile(`[{type: replace, path: /name, value: Bob}]`)},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(MatchYAML(`name: Bob`))
		})
	})

	It("fails when no variables are provided", func() {
		_, err := interpolate.Execute(interpolate.Options{
			TemplateFile:  writeFile(`{name: ((username))}`),
			ExpectAllKeys: true,
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Expected to find variables: username"))
	})
})

var _ = Describe("FromConfigFile", func() {
	It("does nothing if the struct does not have the ConfigFile field", func() {
		config := struct{}{}
		_, err := interpolate.FromConfigFile(&config, nil)
		Expect(err).NotTo(HaveOccurred())
	})

	When("ConfgileFile is defined on Options", func() {
		type options struct {
			ConfigFile string
			VarsFile   []string
			VarsEnv    []string
			Vars       []string
			Name       string `long:"name"`
		}
		type config struct {
			Options options
		}

		It("reads the config file and parses args into the struct", func() {
			c := config{
				Options: options{
					ConfigFile: writeFile(`name: Bob`),
				},
			}
			_, err := interpolate.FromConfigFile(&c, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Options.Name).To(Equal("Bob"))
		})

		It("can read values from vars-files", func() {
			c := config{
				Options: options{
					ConfigFile: writeFile(`name: ((name))`),
					VarsFile:   []string{writeFile(`name: Bob`)},
				},
			}
			_, err := interpolate.FromConfigFile(&c, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Options.Name).To(Equal("Bob"))
		})

		It("can read values from vars", func() {
			c := config{
				Options: options{
					ConfigFile: writeFile(`name: ((name))`),
					Vars:       []string{`name=Bob`},
				},
			}
			_, err := interpolate.FromConfigFile(&c, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Options.Name).To(Equal("Bob"))
		})

		It("can read values from vars-env", func() {
			c := config{
				Options: options{
					ConfigFile: writeFile(`name: ((name))`),
					VarsEnv:    []string{`OM_VAR`},
				},
			}
			err := os.Setenv("OM_VAR_name", "Bob")
			Expect(err).NotTo(HaveOccurred())
			defer os.Unsetenv("OM_VAR_name")

			_, err = interpolate.FromConfigFile(&c, os.Environ)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Options.Name).To(Equal("Bob"))
		})

		It("does nothing if the config file is not defined", func() {
			c := config{}
			_, err := interpolate.FromConfigFile(&c, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Options.Name).To(Equal(""))
		})
	})

	When("ConfigFile is a defined field", func() {
		type config struct {
			ConfigFile string
			VarsFile   []string
			VarsEnv    []string
			Vars       []string
			Name       string `long:"name"`
		}

		It("reads the config file and parses args into the struct", func() {
			c := config{
				ConfigFile: writeFile(`name: Bob`),
			}
			_, err := interpolate.FromConfigFile(&c, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Name).To(Equal("Bob"))
		})

		It("can read values from vars-files", func() {
			c := config{
				ConfigFile: writeFile(`name: ((name))`),
				VarsFile:   []string{writeFile(`name: Bob`)},
			}
			_, err := interpolate.FromConfigFile(&c, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Name).To(Equal("Bob"))
		})

		It("can read values from vars", func() {
			c := config{
				ConfigFile: writeFile(`name: ((name))`),
				Vars:   []string{`name=Bob`},
			}
			_, err := interpolate.FromConfigFile(&c, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Name).To(Equal("Bob"))
		})

		It("can read values from vars-env", func() {
			c := config{
				ConfigFile: writeFile(`name: ((name))`),
				VarsEnv:   []string{`OM_VAR`},
			}
			err := os.Setenv("OM_VAR_name", "Bob")
			Expect(err).NotTo(HaveOccurred())
			defer os.Unsetenv("OM_VAR_name")

			_, err = interpolate.FromConfigFile(&c, os.Environ)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Name).To(Equal("Bob"))
		})

		It("does nothing if the config file is not defined", func() {
			c := config{}
			_, err := interpolate.FromConfigFile(&c, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Name).To(Equal(""))
		})
	})
})

func writeFile(contents string) string {
	file, err := ioutil.TempFile("", "")
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(file.Name(), []byte(contents), 0777)
	Expect(err).NotTo(HaveOccurred())
	return file.Name()
}
