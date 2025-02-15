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

var _ = Describe("PendingChangesService", func() {
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

	Describe("ListStagedPendingChanges", func() {
		It("lists pending changes", func() {
			var path string
			client.DoStub = func(req *http.Request) (*http.Response, error) {
				path = req.URL.Path

				return &http.Response{StatusCode: http.StatusOK,
					Body: ioutil.NopCloser(strings.NewReader(`{
						"product_changes": [{
							"guid":"product-123",
							"errands":[
								{ "name":"errand-1", "post_deploy":"true" }
							],
							"action":"install",
                            "extra_field": "needs to be preserved"
						},
						{
							"guid":"product-234",
							"errands":[
								{ "name":"errand-3", "post_deploy":"true" }
							],
							"action":"update",
                            "completeness_checks": {"configuration_complete": true, "stemcell_present": false, "configurable_properties_valid": true}
						}]
				  }`)),
				}, nil
			}

			output, err := service.ListStagedPendingChanges()
			Expect(err).ToNot(HaveOccurred())

			Expect(output.ChangeList).To(ConsistOf([]api.ProductChange{
				{
					GUID: "product-123",
					Errands: []api.Errand{
						{Name: "errand-1", PostDeploy: "true"},
					},
					Action: "install",
				},
				{
					GUID:   "product-234",
					Action: "update",
					Errands: []api.Errand{
						{Name: "errand-3", PostDeploy: "true"},
					},
					CompletenessChecks: &api.CompletenessChecks{
						ConfigurationComplete:       true,
						StemcellPresent:             false,
						ConfigurablePropertiesValid: true,
					},
				},
			},
			))

			Expect(path).To(Equal("/api/v0/staged/pending_changes"))

			Expect(output.FullReport).To(MatchJSON(`[{
							"guid":"product-123",
							"errands":[
								{ "name":"errand-1", "post_deploy":"true" }
							],
							"action":"install",
                            "extra_field": "needs to be preserved"
						},
						{
							"guid":"product-234",
							"errands":[
								{ "name":"errand-3", "post_deploy":"true" }
							],
							"action":"update",
                            "completeness_checks": {"configuration_complete": true, "stemcell_present": false, "configurable_properties_valid": true}
						}]`))
		})

		Describe("errors", func() {
			Context("the client can't connect to the server", func() {
				It("returns an error", func() {
					client.DoReturns(&http.Response{}, errors.New("some error"))
					_, err := service.ListStagedPendingChanges()
					Expect(err).To(MatchError(ContainSubstring("could not send api request")))
				})
			})

			When("the server won't fetch pending changes", func() {
				It("returns an error", func() {
					client.DoReturns(&http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       ioutil.NopCloser(strings.NewReader(`{}`)),
					}, nil)

					_, err := service.ListStagedPendingChanges()
					Expect(err).To(MatchError(ContainSubstring("request failed")))
				})
			})

			When("the response is not JSON", func() {
				It("returns an error", func() {
					client.DoReturns(&http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(strings.NewReader(`asdf`)),
					}, nil)

					_, err := service.ListStagedPendingChanges()
					Expect(err).To(MatchError(ContainSubstring("could not unmarshal")))
				})
			})
		})
	})
})
