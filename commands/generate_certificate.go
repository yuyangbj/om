package commands

import (
	"github.com/pivotal-cf/om/api"
	"strings"
)

type GenerateCertificate struct {
	service generateCertificateService
	logger  logger
	Options struct {
		Domains string `long:"domains" short:"d" required:"true" description:"domains to generate certificates, delimited by comma, can include wildcard domains"`
	}
}

//go:generate counterfeiter -o ./fakes/generate_certificate_service.go --fake-name GenerateCertificateService . generateCertificateService
type generateCertificateService interface {
	GenerateCertificate(domains api.DomainsInput) (string, error)
}

func NewGenerateCertificate(service generateCertificateService, logger logger) *GenerateCertificate {
	return &GenerateCertificate{service: service, logger: logger}
}

func (g GenerateCertificate) Execute(args []string) error {
	domains := strings.Split(g.Options.Domains, ",")
	for i, domain := range domains {
		domains[i] = strings.TrimSpace(domain)
	}

	output, err := g.service.GenerateCertificate(api.DomainsInput{
		Domains: domains,
	})

	if err != nil {
		return err
	}

	g.logger.Printf(output)
	return nil
}
