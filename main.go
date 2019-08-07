package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/pivotal-cf/om/api"
	"github.com/pivotal-cf/om/formcontent"
	"github.com/pivotal-cf/om/interpolate"
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
	VarsEnv              string `                                                                     env:"OM_VARS_ENV"      experimental:"true" description:"load vars from environment variables by specifying a prefix (e.g.: 'MY' to load MY_var=value)"`
}

func main() {
	//applySleepDuration, _ := time.ParseDuration(applySleepDurationString)

	stdout := log.New(os.Stdout, "", 0)
	stderr := log.New(os.Stderr, "", 0)

	var global globalOptions

	parser := flags.NewNamedParser("om", flags.PrintErrors|flags.PassAfterNonOption|flags.PassDoubleDash)
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

	//logWriter := commands.NewLogWriter(os.Stdout)
	//tableWriter := tablewriter.NewWriter(os.Stdout)

	form := formcontent.NewForm()

	//metadataExtractor := extractor.MetadataExtractor{}

	//presenter := presenters.NewPresenter(presenters.NewTablePresenter(tableWriter), presenters.NewJSONPresenter(os.Stdout))
	//envRendererFactory := renderers.NewFactory(renderers.NewEnvGetter())

	//parser.AddCommand("activate-certificate-authority", "", "", commands.NewActivateCertificateAuthority(api, stdout))
	//parser.AddCommand("apply-changes", "", "", commands.NewApplyChanges(api, api, logWriter, stdout, applySleepDuration))
	//parser.AddCommand("assign-stemcell", "", "", commands.NewAssignStemcell(api, stdout))
	//parser.AddCommand("assign-multi-stemcell", "", "", commands.NewAssignMultiStemcell(api, stdout))
	//parser.AddCommand("available-products", "", "", commands.NewAvailableProducts(api, presenter, stdout))
	//parser.AddCommand("bosh-env", "", "", commands.NewBoshEnvironment(api, stdout, global.Target, envRendererFactory))
	//parser.AddCommand("certificate-authorities", "", "", commands.NewCertificateAuthorities(api, presenter))
	//parser.AddCommand("certificate-authority", "", "", commands.NewCertificateAuthority(api, presenter, stdout))
	//parser.AddCommand("config-template", "", "", commands.NewConfigTemplate(commands.DefaultProvider()))
	//parser.AddCommand("configure-authentication", "", "", commands.NewConfigureAuthentication(api, stdout))
	//parser.AddCommand("configure-director", "", "", commands.NewConfigureDirector(os.Environ, api, stdout))
	//parser.AddCommand("configure-ldap-authentication", "", "", commands.NewConfigureLDAPAuthentication(api, stdout))
	//parser.AddCommand("configure-product", "", "", commands.NewConfigureProduct(os.Environ, api, global.Target, stdout))
	//parser.AddCommand("configure-saml-authentication", "", "", commands.NewConfigureSAMLAuthentication(api, stdout))
	//parser.AddCommand("create-certificate-authority", "", "", commands.NewCreateCertificateAuthority(api, presenter))
	//parser.AddCommand("create-vm-extension", "", "", commands.NewCreateVMExtension(os.Environ, api, stdout))
	//parser.AddCommand("credential-references", "", "", commands.NewCredentialReferences(api, presenter, stdout))
	//parser.AddCommand("credentials", "", "", commands.NewCredentials(api, presenter, stdout))
	//parser.AddCommand("curl", "", "", commands.NewCurl(api, stdout, stderr))
	//parser.AddCommand("delete-certificate-authority", "", "", commands.NewDeleteCertificateAuthority(api, stdout))
	//parser.AddCommand("delete-installation", "", "", commands.NewDeleteInstallation(api, logWriter, stdout, os.Stdin, applySleepDuration))
	//parser.AddCommand("delete-ssl-certificate", "", "", commands.NewDeleteSSLCertificate(api, stdout))
	//parser.AddCommand("delete-product", "", "", commands.NewDeleteProduct(api))
	//parser.AddCommand("delete-unused-products", "", "", commands.NewDeleteUnusedProducts(api, stdout))
	//parser.AddCommand("deployed-manifest", "", "", commands.NewDeployedManifest(api, stdout))
	//parser.AddCommand("deployed-products", "", "", commands.NewDeployedProducts(presenter, api))
	//parser.AddCommand("diagnostic-report", "", "", commands.NewDiagnosticReport(presenter, api))
	//parser.AddCommand("download-product", "", "", commands.NewDownloadProduct(os.Environ, stdout, stderr, os.Stderr))
	//parser.AddCommand("errands", "", "", commands.NewErrands(presenter, api))
	//parser.AddCommand("export-installation", "", "", commands.NewExportInstallation(api, stderr))
	//parser.AddCommand("generate-certificate", "", "", commands.NewGenerateCertificate(api, stdout))
	//parser.AddCommand("generate-certificate-authority", "", "", commands.NewGenerateCertificateAuthority(api, presenter))
	//parser.AddCommand("import-installation", "", "", commands.NewImportInstallation(form, api, global.DecryptionPassphrase, stdout))
	//parser.AddCommand("installation-log", "", "", commands.NewInstallationLog(api, stdout))
	//parser.AddCommand("installations", "", "", commands.NewInstallations(api, presenter))
	//parser.AddCommand("interpolate", "", "", commands.NewInterpolate(os.Environ, stdout))
	//parser.AddCommand("pending-changes", "", "", commands.NewPendingChanges(presenter, api))
	//parser.AddCommand("pre-deploy-check", "", "", commands.NewPreDeployCheck(presenter, api, stdout))
	//parser.AddCommand("regenerate-certificates", "", "", commands.NewRegenerateCertificates(api, stdout))
	//parser.AddCommand("stage-product", "", "", commands.NewStageProduct(api, stdout))
	//parser.AddCommand("ssl-certificate", "", "", commands.NewSSLCertificate(api, presenter))
	//parser.AddCommand("staged-config", "", "", commands.NewStagedConfig(api, stdout))
	//parser.AddCommand("staged-director-config", "", "", commands.NewStagedDirectorConfig(api, stdout))
	//parser.AddCommand("staged-manifest", "", "", commands.NewStagedManifest(api, stdout))
	//parser.AddCommand("staged-products", "", "", commands.NewStagedProducts(presenter, api))
	//parser.AddCommand("tile-metadata", "", "", commands.NewTileMetadata(stdout))
	//parser.AddCommand("unstage-product", "", "", commands.NewUnstageProduct(api, stdout))
	//parser.AddCommand("update-ssl-certificate", "", "", commands.NewUpdateSSLCertificate(api, stdout))
	//parser.AddCommand("upload-product", "", "", commands.NewUploadProduct(form, metadataExtractor, api, stdout))
	parser.AddCommand(
		"upload-stemcell",
		"uploads a given stemcell to the Ops Manager targeted",
		"This command will upload a stemcell to the target Ops Manager. Unless the force flag is used, if the stemcell already exists that upload will be skipped",
		commands.NewUploadStemcell(form, api, stdout),
	)
	parser.AddCommand(
		"version",
		"prints the om release version",
		"This command prints the om release version number.",
		commands.NewVersion(version, os.Stdout),
	)

	if args[0] == "help" || args[0] == "" {
		parser.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	parser.Options = parser.Options | flags.HelpFlag
	_, err = parser.ParseArgs(args)
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}
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
