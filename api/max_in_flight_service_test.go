package api_test

import (
	"io/ioutil"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/om/api"
	"github.com/pivotal-cf/om/api/fakes"
)

var _ = Describe("MaxInFlightService", func() {
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

	Describe("UpdateStagedProductMaxInFlight", func() {
		It("updates max-in-flight for jobs", func() {
			client.DoStub = func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(nil),
				}, nil
			}

			maxInFlight := api.MaxInFlightProperties{
				Properties: map[string]interface{}{
					"a_job-guid": 1,
				},
			}

			err := service.UpdateStagedProductMaxInFlight("some-product-guid", maxInFlight)
			Expect(err).NotTo(HaveOccurred())

			Expect(client.DoCallCount()).To(Equal(1))
			request := client.DoArgsForCall(0)
			Expect("PUT").To(Equal(request.Method))
			Expect("application/json").To(Equal(request.Header.Get("Content-Type")))
			Expect("/api/v0/staged/products/some-product-guid/max_in_flight").To(Equal(request.URL.Path))
			reqBytes, err := ioutil.ReadAll(request.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(reqBytes).To(MatchJSON(`
            {
                "max_in_flight": {
                    "a_job-guid": 1
                }
            }`,
			))
		})
	})
})
