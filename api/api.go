package api

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//counterfeiter:generate -o ./fakes/logger.go --fake-name Logger . logger
type logger interface {
	Println(v ...interface{})
}

type Api struct {
	client                 httpClient
	unauthedClient         httpClient
	progressClient         httpClient
	unauthedProgressClient httpClient
	logger                 logger
}

func (a *Api) UpdateClients(input ApiInput) {
	api := &Api{
		client:                 input.Client,
		unauthedClient:         input.UnauthedClient,
		progressClient:         input.ProgressClient,
		unauthedProgressClient: input.UnauthedProgressClient,
		logger:                 input.Logger,
	}
	*a = *api
}

type ApiInput struct {
	Client                 httpClient
	UnauthedClient         httpClient
	ProgressClient         httpClient
	UnauthedProgressClient httpClient
	Logger                 logger
}

func New(input ApiInput) Api {
	return Api{
		client:                 input.Client,
		unauthedClient:         input.UnauthedClient,
		progressClient:         input.ProgressClient,
		unauthedProgressClient: input.UnauthedProgressClient,
		logger:                 input.Logger,
	}
}