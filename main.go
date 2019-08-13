package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/olekukonko/tablewriter"
	"github.com/pivotal-cf/om/api"
	"github.com/pivotal-cf/om/extractor"
	"github.com/pivotal-cf/om/formcontent"
	"github.com/pivotal-cf/om/interpolate"
	"github.com/pivotal-cf/om/presenters"
	"github.com/pivotal-cf/om/renderers"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"time"

	"github.com/pivotal/uilive"
	"gopkg.in/yaml.v2"

	"github.com/pivotal-cf/om/commands"
	"github.com/pivotal-cf/om/network"
	"github.com/pivotal-cf/om/progress"

	_ "github.com/pivotal-cf/om/download_clients"
)

var version = "unknown"

var applySleepDurationString = "10s"

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type globalOptions struct {
	DecryptionPassphrase string `yaml:"decryption-passphrase" short:"d"  long:"decryption-passphrase" env:"OM_DECRYPTION_PASSPHRASE"             description:"Passphrase to decrypt the installation if the Ops Manager VM has been rebooted (optional for most commands)"`
	ClientID             string `yaml:"client-id"             short:"c"  long:"client-id"             env:"OM_CLIENT_ID"                           description:"Client ID for the Ops Manager VM (not required for unauthenticated commands)"`
	ClientSecret         string `yaml:"client-secret"         short:"s"  long:"client-secret"         env:"OM_CLIENT_SECRET"                       description:"Client Secret for the Ops Manager VM (not required for unauthenticated commands)"`
	Help                 bool   `                             short:"h"  long:"help"                                                               description:"prints this usage information"`
	Password             string `yaml:"password"              short:"p"  long:"password"              env:"OM_PASSWORD"                            description:"admin password for the Ops Manager VM (not required for unauthenticated commands)"`
	ConnectTimeout       int    `yaml:"connect-timeout"       short:"o"  long:"connect-timeout"       env:"OM_CONNECT_TIMEOUT"     default:"10"    description:"timeout in seconds to make TCP connections"`
	RequestTimeout       int    `yaml:"request-timeout"       short:"r"  long:"request-timeout"       env:"OM_REQUEST_TIMEOUT"     default:"1800"  description:"timeout in seconds for HTTP requests to Ops Manager"`
	SkipSSLValidation    bool   `yaml:"skip-ssl-validation"   short:"k"  long:"skip-ssl-validation"   env:"OM_SKIP_SSL_VALIDATION"                 description:"skip ssl certificate validation during http requests"`
	Target               string `yaml:"target"                short:"t"  long:"target"                env:"OM_TARGET"                              description:"location of the Ops Manager VM"`
	Trace                bool   `yaml:"trace"                            long:"trace"                 env:"OM_TRACE"                               description:"prints HTTP requests and response payloads"`
	Username             string `yaml:"username"              short:"u"  long:"username"              env:"OM_USERNAME"                            description:"admin username for the Ops Manager VM (not required for unauthenticated commands)"`
	Env                  string `                             short:"e"  long:"env"                                                              description:"env file with login credentials"`
	Version              bool   `                             short:"v"  long:"version"                                                          description:"prints the om release version"`
	VarsEnv              string `                             hidden:"true" long:"vars-env" env:"OM_VARS_ENV"      experimental:"true" description:"load vars from environment variables by specifying a prefix (e.g.: 'MY' to load MY_var=value)"`
}

var parser = flags.NewNamedParser("om", flags.PassDoubleDash)

func main() {
	parser.Usage = "[global options]"
	applySleepDuration, _ := time.ParseDuration(applySleepDurationString)

	stdout := log.New(os.Stdout, "", 0)
	stderr := log.New(os.Stderr, "", 0)

	var global globalOptions
	_, err := parser.AddGroup("global", "global", &global)
	if err != nil {
		stderr.Fatal(err)
	}

	args, err := parser.Parse()
	if err != nil {
		stderr.Fatal(err)
	}

	err = setEnvFileProperties(&global)
	if err != nil {
		stderr.Fatal(err)
	}

	if global.Version {
		args = []string{"version"}
	}

	if global.Help || len(args) == 0 {
		args = []string{"help"}
	}

	api, logWriter, form, metadataExtractor, presenter, envRendererFactory := setupLotsOfThings(global, stderr)
	addCommandsToParser(api, stdout, logWriter, applySleepDuration, presenter, global, envRendererFactory, stderr, form, metadataExtractor)

	if args[0] == "help" || args[0] == "" {
		fmt.Println("ॐ")
		fmt.Println("om helps you interact with an Ops Manager")
		fmt.Println("")
		parser.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	parser.Options = parser.Options | flags.HelpFlag
	_, err = parser.ParseArgs(args)
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			fmt.Println("ॐ")
			parser.WriteHelp(os.Stdout)
			os.Exit(0)
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(1)
		}
	}
}

func setupLotsOfThings(global globalOptions, stderr *log.Logger) (api.Api, *commands.LogWriter, *formcontent.Form, extractor.MetadataExtractor, *presenters.MultiPresenter, renderers.Factory) {
	var err error

	requestTimeout := time.Duration(global.RequestTimeout) * time.Second
	connectTimeout := time.Duration(global.ConnectTimeout) * time.Second

	var unauthenticatedClient, authedClient, authedCookieClient, unauthenticatedProgressClient, authedProgressClient httpClient

	unauthenticatedClient = network.NewUnauthenticatedClient(global.Target, global.SkipSSLValidation, requestTimeout, connectTimeout)

	authedClient, err = network.NewOAuthClient(global.Target, global.Username, global.Password, global.ClientID, global.ClientSecret, global.SkipSSLValidation, false, requestTimeout, connectTimeout)
	if err != nil {
		stderr.Fatal(err)
	}

	if global.DecryptionPassphrase != "" {
		authedClient = network.NewDecryptClient(authedClient, unauthenticatedClient, global.DecryptionPassphrase, os.Stderr)
	}

	authedCookieClient, err = network.NewOAuthClient(global.Target, global.Username, global.Password, global.ClientID, global.ClientSecret, global.SkipSSLValidation, true, requestTimeout, connectTimeout)
	if err != nil {
		stderr.Fatal(err)
	}

	liveWriter := uilive.New()
	liveWriter.Out = os.Stderr

	unauthenticatedProgressClient = network.NewProgressClient(unauthenticatedClient, progress.NewBar(), liveWriter)
	authedProgressClient = network.NewProgressClient(authedClient, progress.NewBar(), liveWriter)

	if global.Trace {
		unauthenticatedClient = network.NewTraceClient(unauthenticatedClient, os.Stderr)
		unauthenticatedProgressClient = network.NewTraceClient(unauthenticatedProgressClient, os.Stderr)
		authedClient = network.NewTraceClient(authedClient, os.Stderr)
		authedCookieClient = network.NewTraceClient(authedCookieClient, os.Stderr)
		authedProgressClient = network.NewTraceClient(authedProgressClient, os.Stderr)
	}

	api := api.New(api.ApiInput{
		Client:                 authedClient,
		UnauthedClient:         unauthenticatedClient,
		ProgressClient:         authedProgressClient,
		UnauthedProgressClient: unauthenticatedProgressClient,
		Logger:                 stderr,
	})

	logWriter := commands.NewLogWriter(os.Stdout)
	tableWriter := tablewriter.NewWriter(os.Stdout)

	form := formcontent.NewForm()

	metadataExtractor := extractor.MetadataExtractor{}

	presenter := presenters.NewPresenter(presenters.NewTablePresenter(tableWriter), presenters.NewJSONPresenter(os.Stdout))
	envRendererFactory := renderers.NewFactory(renderers.NewEnvGetter())

	return api, logWriter, form, metadataExtractor, presenter, envRendererFactory
}

func addCommandsToParser(api api.Api, stdout *log.Logger, logWriter *commands.LogWriter, applySleepDuration time.Duration, presenter *presenters.MultiPresenter, global globalOptions, envRendererFactory renderers.Factory, stderr *log.Logger, form *formcontent.Form, metadataExtractor extractor.MetadataExtractor) {
	addCommand("activate-certificate-authority", "activates a certificate authority on the Ops Manager", "This authenticated command activates an existing certificate authority on the Ops Manager", commands.NewActivateCertificateAuthority(api, stdout))
	addCommand("apply-changes", "triggers an install on the Ops Manager targeted", "This authenticated command kicks off an install of any staged changes on the Ops Manager.", commands.NewApplyChanges(api, api, logWriter, stdout, applySleepDuration))
	addCommand(
		"assign-stemcell",
		"assigns an uploaded stemcell to a product in the targeted Ops Manager",
		"This command will assign an already uploaded stemcell to a specific product in Ops Manager.\n"+
			"It is recommended to use \"upload-stemcell --floating=false\" before using this command.",
		commands.NewAssignStemcell(api, stdout),
	)
	addCommand(
		"assign-multi-stemcell",
		"assigns multiple uploaded stemcells to a product in the targeted Ops Manager 2.6+",
		"This command will assign multiple already uploaded stemcells to a specific product in Ops Manager 2.6+.\n"+
			"It is recommended to use \"upload-stemcell --floating=false\" before using this command.",
		commands.NewAssignMultiStemcell(api, stdout),
	)
	addCommand("available-products", "list available products", "This authenticated command lists all available products.", commands.NewAvailableProducts(api, presenter, stdout))
	addCommand("bosh-env", "prints bosh environment variables", "This prints bosh environment variables to target bosh director. You can invoke it directly to see its output, or use it directly with an evaluate-type command:\nOn posix system: eval \"$(om bosh-env)\"\nOn powershell: iex $(om bosh-env | Out-String)", commands.NewBoshEnvironment(api, stdout, global.Target, envRendererFactory))
	addCommand("certificate-authorities", "lists certificates managed by Ops Manager", "lists certificates managed by Ops Manager", commands.NewCertificateAuthorities(api, presenter))
	addCommand("certificate-authority", "prints requested certificate authority", "prints requested certificate authority", commands.NewCertificateAuthority(api, presenter, stdout))
	addCommand("config-template", "**EXPERIMENTAL** generates a config template from a Pivnet product", "**EXPERIMENTAL** this command generates a product configuration template from a .pivotal file on Pivnet", commands.NewConfigTemplate(commands.DefaultProvider()))
	addCommand("configure-authentication", "configures Ops Manager with an internal userstore and admin user account", "This unauthenticated command helps setup the internal userstore authentication mechanism for your Ops Manager.", commands.NewConfigureAuthentication(api, stdout))
	addCommand("configure-director", "configures the director", "This authenticated command configures the director.", commands.NewConfigureDirector(os.Environ, api, stdout))
	addCommand("configure-ldap-authentication", "configures Ops Manager with LDAP authentication", "This unauthenticated command helps setup the authentication mechanism for your Ops Manager with LDAP.", commands.NewConfigureLDAPAuthentication(api, stdout))
	addCommand("configure-product", "configures a staged product", "This authenticated command configures a staged product", commands.NewConfigureProduct(os.Environ, api, global.Target, stdout))
	addCommand("configure-saml-authentication", "configures Ops Manager with SAML authentication", "This unauthenticated command helps setup the authentication mechanism for your Ops Manager with SAML.", commands.NewConfigureSAMLAuthentication(api, stdout))
	addCommand("create-certificate-authority", "creates a certificate authority on the Ops Manager", "This authenticated command creates a certificate authority on the Ops Manager with the given cert and key", commands.NewCreateCertificateAuthority(api, presenter))
	addCommand("create-vm-extension", "creates/updates a VM extension", "This creates/updates a VM extension", commands.NewCreateVMExtension(os.Environ, api, stdout))
	addCommand("credential-references", "list credential references for a deployed product", "This authenticated command lists credential references for deployed products.", commands.NewCredentialReferences(api, presenter, stdout))
	addCommand("credentials", "fetch credentials for a deployed product", "This authenticated command fetches credentials for deployed products.", commands.NewCredentials(api, presenter, stdout))
	addCommand("curl", "issues an authenticated API request", "This command issues an authenticated API request as defined in the arguments", commands.NewCurl(api, stdout, stderr))
	addCommand("delete-certificate-authority", "deletes a certificate authority on the Ops Manager", "This authenticated command deletes an existing certificate authority on the Ops Manager", commands.NewDeleteCertificateAuthority(api, stdout))
	addCommand("delete-installation", "deletes all the products on the Ops Manager targeted", "This authenticated command deletes all the products installed on the targeted Ops Manager.", commands.NewDeleteInstallation(api, logWriter, stdout, os.Stdin, applySleepDuration))
	addCommand("delete-ssl-certificate", "deletes certificate applied to Ops Manager", "This authenticated command deletes a custom certificate applied to Ops Manager and reverts to the auto-generated cert", commands.NewDeleteSSLCertificate(api, stdout))
	addCommand("delete-product", "deletes a product from the Ops Manager", "This command deletes the named product from the targeted Ops Manager", commands.NewDeleteProduct(api))
	addCommand("delete-unused-products", "deletes unused products on the Ops Manager targeted", "This command deletes unused products in the targeted Ops Manager", commands.NewDeleteUnusedProducts(api, stdout))
	addCommand("deployed-manifest", "prints the deployed manifest for a product", "This authenticated command prints the deployed manifest for a ", commands.NewDeployedManifest(api, stdout))
	addCommand("deployed-products", "lists deployed products", "This authenticated command lists all deployed products.", commands.NewDeployedProducts(presenter, api))
	addCommand("diagnostic-report", "reports current state of your Ops Manager", "retrieve a diagnostic report with general information about the state of your Ops Manager.", commands.NewDiagnosticReport(presenter, api))
	addCommand("download-product", "downloads a specified product file from Pivotal Network", "This command attempts to download a single product file from Pivotal Network. The API token used must be associated with a user account that has already accepted the EULA for the specified product", commands.NewDownloadProduct(os.Environ, stdout, stderr, os.Stderr))
	addCommand("errands", "list errands for a product", "This authenticated command lists all errands for a product.", commands.NewErrands(presenter, api))
	addCommand("export-installation", "exports the installation of the target Ops Manager", "This command will export the current installation of the target Ops Manager.", commands.NewExportInstallation(api, stderr))
	addCommand("generate-certificate", "generates a new certificate signed by Ops Manager's root CA", "This authenticated command generates a new RSA public/private certificate signed by Ops Manager’s root CA certificate", commands.NewGenerateCertificate(api, stdout))
	addCommand("generate-certificate-authority", "generates a certificate authority on the Opsman", "This authenticated command generates a certificate authority on the Ops Manager", commands.NewGenerateCertificateAuthority(api, presenter))
	addCommand("import-installation", "imports a given installation to the Ops Manager targeted", "This unauthenticated command attempts to import an installation to the Ops Manager targeted.", commands.NewImportInstallation(form, api, global.DecryptionPassphrase, stdout))
	addCommand("installation-log", "output installation logs", "This authenticated command retrieves the logs for a given installation.", commands.NewInstallationLog(api, stdout))
	addCommand("installations", "list recent installation events", "This authenticated command lists all recent installation events.", commands.NewInstallations(api, presenter))
	addCommand("interpolate", "Interpolates variables into a manifest", "Interpolates variables into a manifest", commands.NewInterpolate(os.Environ, stdout))
	addCommand("pending-changes", "lists pending changes", "This authenticated command lists all pending changes.", commands.NewPendingChanges(presenter, api))
	addCommand("pre-deploy-check", "**EXPERIMENTAL** lists pending changes", "**EXPERIMENTAL** This authenticated command lists all pending changes.", commands.NewPreDeployCheck(presenter, api, stdout))
	addCommand("regenerate-certificates", "deletes all non-configurable certificates in Ops Manager so they will automatically be regenerated on the next apply-changes", "This authenticated command deletes all non-configurable certificates in Ops Manager so they will automatically be regenerated on the next apply-changes", commands.NewRegenerateCertificates(api, stdout))
	addCommand("stage-product", "stages a given product in the Ops Manager targeted", "This command attempts to stage a product in the Ops Manager", commands.NewStageProduct(api, stdout))
	addCommand("ssl-certificate", "gets certificate applied to Ops Manager", "This authenticated command gets certificate applied to Ops Manager", commands.NewSSLCertificate(api, presenter))
	addCommand("staged-config", "**EXPERIMENTAL** generates a config from a staged product", "This command generates a config from a staged product that can be passed in to om configure-product (Note: credentials are not available and will appear as '***')", commands.NewStagedConfig(api, stdout))
	addCommand("staged-director-config", "**EXPERIMENTAL** generates a config from a staged director", "This command generates a config from a staged director that can be passed in to om configure-director", commands.NewStagedDirectorConfig(api, stdout))
	addCommand("staged-manifest", "prints the staged manifest for a product", "This authenticated command prints the staged manifest for a product", commands.NewStagedManifest(api, stdout))
	addCommand("staged-products", "lists staged products", "This authenticated command lists all staged products.", commands.NewStagedProducts(presenter, api))
	addCommand("tile-metadata", "prints tile metadata", "This command prints metadata about the given tile", commands.NewTileMetadata(stdout))
	addCommand(
		"unstage-product",
		"unstages a given product from the Ops Manager targeted",
		"This command attempts to unstage a product from the Ops Manager",
		commands.NewUnstageProduct(api, stdout),
	)
	addCommand(
		"update-ssl-certificate",
		"updates the SSL Certificate on the Ops Manager",
		"This authenticated command updates the SSL Certificate on the Ops Manager with the given cert and key",
		commands.NewUpdateSSLCertificate(api, stdout),
	)
	addCommand(
		"upload-product",
		"uploads a given product to the Ops Manager targeted",
		"This command attempts to upload a product to the Ops Manager",
		commands.NewUploadProduct(form, metadataExtractor, api, stdout),
	)
	addCommand(
		"upload-stemcell",
		"uploads a given stemcell to the Ops Manager targeted",
		"This command will upload a stemcell to the target Ops Manager. Unless the force flag is used, if the stemcell already exists that upload will be skipped",
		commands.NewUploadStemcell(form, api, stdout),
	)
	addCommand(
		"version",
		"prints the om release version",
		"This command prints the om release version number.",
		commands.NewVersion(version, os.Stdout),
	)
}

func setEnvFileProperties(global *globalOptions) error {
	if global.Env == "" {
		return nil
	}

	var opts globalOptions
	_, err := os.Open(global.Env)
	if err != nil {
		return fmt.Errorf("env file does not exist: %s", err)
	}

	contents, err := interpolate.Execute(interpolate.Options{
		TemplateFile:  global.Env,
		EnvironFunc:   os.Environ,
		VarsEnvs:      []string{global.VarsEnv},
		ExpectAllKeys: false,
	})
	if err != nil {
		return err
	}

	err = yaml.UnmarshalStrict(contents, &opts)
	if err != nil {
		return fmt.Errorf("could not parse env file: %s", err)
	}

	if global.ClientID == "" {
		global.ClientID = opts.ClientID
	}
	if global.ClientSecret == "" {
		global.ClientSecret = opts.ClientSecret
	}
	if global.Password == "" {
		global.Password = opts.Password
	}
	if global.ConnectTimeout == 10 && opts.ConnectTimeout != 0 {
		global.ConnectTimeout = opts.ConnectTimeout
	}
	if global.RequestTimeout == 1800 && opts.RequestTimeout != 0 {
		global.RequestTimeout = opts.RequestTimeout
	}
	if global.SkipSSLValidation == false {
		global.SkipSSLValidation = opts.SkipSSLValidation
	}
	if global.Target == "" {
		global.Target = opts.Target
	}
	if global.Trace == false {
		global.Trace = opts.Trace
	}
	if global.Username == "" {
		global.Username = opts.Username
	}
	if global.DecryptionPassphrase == "" {
		global.DecryptionPassphrase = opts.DecryptionPassphrase
	}

	err = checkForVars(global)
	if err != nil {
		return fmt.Errorf("found problem in --env file: %s", err)
	}

	return nil
}

func checkForVars(opts *globalOptions) error {
	var errBuffer []string

	interpolateRegex := regexp.MustCompile(`\(\(.*\)\)`)

	if interpolateRegex.MatchString(opts.DecryptionPassphrase) {
		errBuffer = append(errBuffer, "* use OM_DECRYPTION_PASSPHRASE environment variable for the decryption-passphrase value")
	}

	if interpolateRegex.MatchString(opts.ClientID) {
		errBuffer = append(errBuffer, "* use OM_CLIENT_ID environment variable for the client-id value")
	}

	if interpolateRegex.MatchString(opts.ClientSecret) {
		errBuffer = append(errBuffer, "* use OM_CLIENT_SECRET environment variable for the client-secret value")
	}

	if interpolateRegex.MatchString(opts.Password) {
		errBuffer = append(errBuffer, "* use OM_PASSWORD environment variable for the password value")
	}

	if interpolateRegex.MatchString(opts.Target) {
		errBuffer = append(errBuffer, "* use OM_TARGET environment variable for the target value")
	}

	if interpolateRegex.MatchString(opts.Username) {
		errBuffer = append(errBuffer, "* use OM_USERNAME environment variable for the username value")
	}

	if len(errBuffer) > 0 {
		errBuffer = append([]string{"env file contains YAML placeholders. Pleases provide them via interpolation or environment variables."}, errBuffer...)
		errBuffer = append(errBuffer, "Or, to enable interpolation of env.yml with variables from env-vars,")
		errBuffer = append(errBuffer, "set the OM_VARS_ENV env var and put export the needed vars.")

		return fmt.Errorf(strings.Join(errBuffer, "\n"))
	}

	return nil
}

func addCommand(
	name string,
	short string,
	long string,
	command interface{},
) {
	_, err := parser.AddCommand(name, short, long, command)
	if err != nil {
		panic(err)
	}
}
