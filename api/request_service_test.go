package api_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pivotal-cf/om/api"
	"github.com/pivotal-cf/om/api/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RequestService", func() {
	Describe("Curl", func() {
		var (
			client  *fakes.HttpClient
			service api.Api
		)

		BeforeEach(func() {
			client = &fakes.HttpClient{}
			service = api.New(api.ApiInput{
				Client: client,
			})
		})

		It("makes a request against the api and returns a response", func() {
			client.DoReturns(&http.Response{
				StatusCode: http.StatusTeapot,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Body: ioutil.NopCloser(strings.NewReader("some-response-body")),
			}, nil)

			output, err := service.Curl(api.RequestServiceCurlInput{
				Method:  "PUT",
				Path:    "/api/v0/api/endpoint",
				Data:    strings.NewReader("some-request-body"),
				Headers: http.Header{"Content-Type": []string{"application/json"}},
			})
			Expect(err).ToNot(HaveOccurred())

			request := client.DoArgsForCall(0)
			Expect(request.Method).To(Equal("PUT"))
			Expect(request.URL.Path).To(Equal("/api/v0/api/endpoint"))
			Expect(request.Header).To(Equal(http.Header{
				"Content-Type": []string{"application/json"},
			}))

			body, err := ioutil.ReadAll(request.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(body)).To(Equal("some-request-body"))

			Expect(output.StatusCode).To(Equal(http.StatusTeapot))
			Expect(output.Headers).To(Equal(http.Header{
				"Content-Type": []string{"application/json"},
			}))

			body, err = ioutil.ReadAll(output.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(body)).To(Equal("some-response-body"))
		})

		Context("failure cases", func() {
			When("the request cannot be constructed", func() {
				It("returns an error", func() {
					_, err := service.Curl(api.RequestServiceCurlInput{
						Method: "PUT",
						Path:   "%%%",
						Data:   strings.NewReader("some-request-body"),
					})

					Expect(err).To(MatchError(ContainSubstring("failed constructing request:")))
				})
			})

			When("the request cannot be made", func() {
				It("returns an error", func() {
					client.DoReturns(&http.Response{}, errors.New("boom"))

					_, err := service.Curl(api.RequestServiceCurlInput{
						Method: "PUT",
						Path:   "/api/v0/api/endpoint",
						Data:   strings.NewReader("some-request-body"),
					})

					Expect(err).To(MatchError(ContainSubstring("failed submitting request:")))
				})
			})
		})
	})
})
