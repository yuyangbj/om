package download_clients

type Versioner interface {
	Version() string
}

type S3Version struct {
	path    string
	version string
}

func (s *S3Version) Version() string {
	return s.version
}

type PivnetVersion struct {
	releaseId int
	version   string
	slug      string
}

func (p *PivnetVersion) Version() string {
	return p.version
}
