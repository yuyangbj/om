package commands_test

import (
	"fmt"

	"github.com/pivotal-cf/om/api"
	"github.com/pivotal-cf/om/commands"
	"github.com/pivotal-cf/om/commands/fakes"
	presenterfakes "github.com/pivotal-cf/om/presenters/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Certificate Authorities", func() {
	var (
		certificateAuthorities            *commands.CertificateAuthorities
		fakeCertificateAuthoritiesService *fakes.CertificateAuthoritiesService
		fakePresenter                     *presenterfakes.FormattedPresenter
	)

	BeforeEach(func() {
		fakeCertificateAuthoritiesService = &fakes.CertificateAuthoritiesService{}
		fakePresenter = &presenterfakes.FormattedPresenter{}
		certificateAuthorities = commands.NewCertificateAuthorities(fakeCertificateAuthoritiesService, fakePresenter)
	})

	Describe("Execute", func() {
		var certificateAuthoritiesOutput []api.CA

		BeforeEach(func() {
			certificateAuthoritiesOutput = []api.CA{
				{
					GUID:      "some-guid",
					Issuer:    "Pivotal",
					CreatedOn: "2017-01-09",
					ExpiresOn: "2021-01-09",
					Active:    true,
					CertPEM:   "-----BEGIN CERTIFICATE-----\nMIIC+zCCAeOgAwIBAgI....",
				},
				{
					GUID:      "other-guid",
					Issuer:    "Customer",
					CreatedOn: "2017-01-10",
					ExpiresOn: "2021-01-10",
					Active:    false,
					CertPEM:   "-----BEGIN CERTIFICATE-----\nMIIC+zCCAeOgAwIBBhI....",
				},
			}

			fakeCertificateAuthoritiesService.ListCertificateAuthoritiesReturns(
				api.CertificateAuthoritiesOutput{CAs: certificateAuthoritiesOutput},
				nil,
			)
		})

		It("prints the certificate authorities to a table", func() {
			err := executeCommand(certificateAuthorities, []string{}, nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeCertificateAuthoritiesService.ListCertificateAuthoritiesCallCount()).To(Equal(1))

			Expect(fakePresenter.PresentCertificateAuthoritiesCallCount()).To(Equal(1))
			Expect(fakePresenter.PresentCertificateAuthoritiesArgsForCall(0)).To(Equal(certificateAuthoritiesOutput))
		})

		Context("when the format flag is provided", func() {
			It("calls the presenter to set the json format", func() {
				err := executeCommand(certificateAuthorities, []string{
					"--format", "json",
				}, nil)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakePresenter.SetFormatCallCount()).To(Equal(1))
				Expect(fakePresenter.SetFormatArgsForCall(0)).To(Equal("json"))
			})
		})

		Context("when the flag cannot parsed", func() {
			It("returns an error", func() {
				err := executeCommand(certificateAuthorities, []string{"--bogus", "nothing"}, nil)
				Expect(err).To(MatchError(
					"unknown flag `bogus'",
				))
			})
		})

		Context("when request for certificate authorities fails", func() {
			It("returns an error", func() {
				fakeCertificateAuthoritiesService.ListCertificateAuthoritiesReturns(
					api.CertificateAuthoritiesOutput{},
					fmt.Errorf("could not get certificate authorities"),
				)

				err := executeCommand(certificateAuthorities, []string{}, nil)
				Expect(err).To(MatchError("could not get certificate authorities"))
			})
		})
	})
})
