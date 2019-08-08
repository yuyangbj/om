package commands_test

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/om/api"
	"github.com/pivotal-cf/om/commands"
	"github.com/pivotal-cf/om/commands/fakes"
)

var _ = Describe("DeleteCertificateAuthority", func() {
	var (
		fakeService *fakes.DeleteCertificateAuthorityService
		fakeLogger  *fakes.Logger
		command     *commands.DeleteCertificateAuthority
	)

	BeforeEach(func() {
		fakeService = &fakes.DeleteCertificateAuthorityService{}
		fakeLogger = &fakes.Logger{}
		command = commands.NewDeleteCertificateAuthority(fakeService, fakeLogger)
	})

	Describe("Execute", func() {
		It("deletes the specified certificate authority", func() {
			err := executeCommand(command, []string{
				"--id", "some-certificate-authority-id",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeService.DeleteCertificateAuthorityCallCount()).To(Equal(1))
			Expect(fakeService.DeleteCertificateAuthorityArgsForCall(0)).To(Equal(api.DeleteCertificateAuthorityInput{
				GUID: "some-certificate-authority-id",
			}))

			Expect(fakeLogger.PrintfCallCount()).To(Equal(1))
			format, content := fakeLogger.PrintfArgsForCall(0)
			Expect(fmt.Sprintf(format, content...)).To(Equal("Certificate authority 'some-certificate-authority-id' deleted\n"))
		})

		Context("failure cases", func() {
			Context("when the service fails to delete a certificate", func() {
				It("returns an error", func() {
					fakeService.DeleteCertificateAuthorityReturns(errors.New("failed to delete certificate"))

					err := executeCommand(command, []string{
						"--id", "some-certificate-authority-id",
					})
					Expect(err).To(MatchError("failed to delete certificate"))
				})
			})
			Context("when an unknown flag is provided", func() {
				It("returns an error", func() {
					err := executeCommand(command, []string{"--badflag"})
					Expect(err).To(MatchError("unknown flag `badflag'"))
				})
			})
			Context("when the id flag is not provided", func() {
				It("returns an error", func() {
					err := executeCommand(command, []string{})
					Expect(err.Error()).To(MatchRegexp("the required flag.*--id"))
				})
			})
		})
	})
})
