package commands_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/om/api"
	"github.com/pivotal-cf/om/commands"
	"github.com/pivotal-cf/om/commands/fakes"
	"io/ioutil"
	"os"
)

var _ = Describe("AssignStemcell", func() {
	var (
		fakeService *fakes.AssignStemcellService
		logger      *fakes.Logger
		command     *commands.AssignStemcell
	)

	BeforeEach(func() {
		fakeService = &fakes.AssignStemcellService{}
		logger = &fakes.Logger{}
		command = commands.NewAssignStemcell(fakeService, logger)
	})

	Context("when --stemcell exists for the specified product", func() {
		BeforeEach(func() {
			fakeService.ListStemcellsReturns(api.ProductStemcells{
				Products: []api.ProductStemcell{
					{
						GUID:                  "cf-guid",
						ProductName:           "cf",
						StagedForDeletion:     false,
						StagedStemcellVersion: "",
						AvailableVersions: []string{
							"1234.5", "1234.6", "1234.99",
						},
					},
				},
			}, nil)
		})

		It("assigns the stemcell", func() {
			err := executeCommand(command, []string{"--product", "cf", "--stemcell", "1234.6"}, nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeService.ListStemcellsCallCount()).To(Equal(1))
			Expect(fakeService.AssignStemcellCallCount()).To(Equal(1))

			Expect(fakeService.AssignStemcellArgsForCall(0)).To(Equal(api.ProductStemcells{
				Products: []api.ProductStemcell{
					{
						GUID:                  "cf-guid",
						StagedStemcellVersion: "1234.6",
					},
				},
			}))
		})

		Context("when --stemcell latest is used", func() {
			It("assign the latest stemcell available", func() {
				err := executeCommand(command, []string{"--product", "cf", "--stemcell", "latest"}, nil)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeService.ListStemcellsCallCount()).To(Equal(1))
				Expect(fakeService.AssignStemcellCallCount()).To(Equal(1))

				Expect(fakeService.AssignStemcellArgsForCall(0)).To(Equal(api.ProductStemcells{
					Products: []api.ProductStemcell{
						{
							GUID:                  "cf-guid",
							StagedStemcellVersion: "1234.99",
						},
					},
				}))
			})
		})
	})

	Context("when there is no --stemcell provided", func() {
		BeforeEach(func() {
			fakeService.ListStemcellsReturns(api.ProductStemcells{
				Products: []api.ProductStemcell{
					{
						GUID:                  "cf-guid",
						ProductName:           "cf",
						StagedForDeletion:     false,
						StagedStemcellVersion: "",
						AvailableVersions: []string{
							"1234.5", "1234.6", "1234.99",
						},
					},
				},
			}, nil)
		})

		It("defaults to latest", func() {
			err := executeCommand(command, []string{"--product", "cf"}, nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeService.ListStemcellsCallCount()).To(Equal(1))
			Expect(fakeService.AssignStemcellCallCount()).To(Equal(1))

			Expect(fakeService.AssignStemcellArgsForCall(0)).To(Equal(api.ProductStemcells{
				Products: []api.ProductStemcell{
					{
						GUID:                  "cf-guid",
						StagedStemcellVersion: "1234.99",
					},
				},
			}))
		})
	})

	Context("when config file is provided", func() {
		var configFile *os.File

		BeforeEach(func() {
			var err error

			fakeService.ListStemcellsReturns(api.ProductStemcells{
				Products: []api.ProductStemcell{
					{
						GUID:                  "cf-guid",
						ProductName:           "cf",
						StagedForDeletion:     false,
						StagedStemcellVersion: "",
						AvailableVersions: []string{
							"1234.5", "1234.6", "1234.99",
						},
					},
				},
			}, nil)

			configContent := `
product: cf
stemcell: "1234.6"
`
			configFile, err = ioutil.TempFile("", "")
			Expect(err).NotTo(HaveOccurred())

			_, err = configFile.WriteString(configContent)
			Expect(err).NotTo(HaveOccurred())
		})

		It("reads configuration from config file", func() {
			err := executeCommand(command, []string{"--config", configFile.Name()}, nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeService.ListStemcellsCallCount()).To(Equal(1))
			Expect(fakeService.AssignStemcellCallCount()).To(Equal(1))

			Expect(fakeService.AssignStemcellArgsForCall(0)).To(Equal(api.ProductStemcells{
				Products: []api.ProductStemcell{
					{
						GUID:                  "cf-guid",
						StagedStemcellVersion: "1234.6",
					},
				},
			}))
		})
	})

	Context("when given stemcell version is not available", func() {
		BeforeEach(func() {
			fakeService.ListStemcellsReturns(api.ProductStemcells{
				Products: []api.ProductStemcell{
					{
						GUID:                  "cf-guid",
						ProductName:           "cf",
						StagedForDeletion:     false,
						StagedStemcellVersion: "",
						AvailableVersions: []string{
							"1234.5", "1234.6",
						},
					},
				},
			}, nil)
		})
		It("returns an error with the available stemcells", func() {
			err := executeCommand(command, []string{"--product", "cf", "--stemcell", "1234.1"}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stemcell version 1234.1 not found in Ops Manager"))
			Expect(err.Error()).To(ContainSubstring("Available Stemcells for \"cf\": 1234.5, 1234.6"))

			Expect(fakeService.ListStemcellsCallCount()).To(Equal(1))
			Expect(fakeService.AssignStemcellCallCount()).To(Equal(0))
		})
	})

	Context("when the product is not found but the stemcell exists", func() {
		BeforeEach(func() {
			fakeService.ListStemcellsReturns(api.ProductStemcells{
				Products: []api.ProductStemcell{
					{
						GUID:                  "different-product-guid",
						ProductName:           "different-product",
						StagedForDeletion:     false,
						StagedStemcellVersion: "",
						AvailableVersions:     []string{"1234.5"},
					},
				},
			}, nil)
		})

		It("returns an error", func() {
			err := executeCommand(command, []string{"--product", "cf", "--stemcell", "1234.5"}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not list product stemcell: product \"cf\" not found"))

			Expect(fakeService.ListStemcellsCallCount()).To(Equal(1))
			Expect(fakeService.AssignStemcellCallCount()).To(Equal(0))
		})
	})

	Context("when the product is staged for deletion", func() {
		BeforeEach(func() {
			fakeService.ListStemcellsReturns(api.ProductStemcells{
				Products: []api.ProductStemcell{
					{
						GUID:                  "cf-guid",
						ProductName:           "cf",
						StagedForDeletion:     true,
						StagedStemcellVersion: "",
						AvailableVersions:     []string{},
					},
				},
			}, nil)
		})

		It("returns an error", func() {
			err := executeCommand(command, []string{"--product", "cf", "--stemcell", "1234.5"}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not assign stemcell: product \"cf\" is staged for deletion"))

			Expect(fakeService.ListStemcellsCallCount()).To(Equal(1))
			Expect(fakeService.AssignStemcellCallCount()).To(Equal(0))
		})
	})

	Context("when no available stemcell returned from api", func() {
		BeforeEach(func() {
			fakeService.ListStemcellsReturns(api.ProductStemcells{
				Products: []api.ProductStemcell{
					{
						GUID:                    "cf-guid",
						ProductName:             "cf",
						StagedForDeletion:       false,
						StagedStemcellVersion:   "",
						RequiredStemcellVersion: "1234.9",
						AvailableVersions:       []string{},
					},
				},
			}, nil)
		})

		It("returns an error", func() {
			err := executeCommand(command, []string{"--product", "cf", "--stemcell", "1234.5"}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no stemcells are available for \"cf\"."))
			Expect(err.Error()).To(ContainSubstring("minimum required stemcell version is: 1234.9"))
			Expect(err.Error()).To(ContainSubstring("upload-stemcell, and try again"))

			Expect(fakeService.ListStemcellsCallCount()).To(Equal(1))
			Expect(fakeService.AssignStemcellCallCount()).To(Equal(0))
		})
	})

	Context("when an unknown flag is provided", func() {
		It("returns an error", func() {
			err := executeCommand(command, []string{"--badflag"}, nil)
			Expect(err).To(MatchError("unknown flag `badflag'"))
		})
	})

	Context("when the product flag is not provided", func() {
		It("returns an error", func() {
			err := executeCommand(command, []string{}, nil)
			Expect(err.Error()).To(MatchRegexp("the required flag.*--product"))
		})
	})
})
