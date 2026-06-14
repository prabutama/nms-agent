//go:build !unix

package discovery

func ChownGeneratedArtifact(path string) error {
	return chownToServiceUser(path)
}

func chownToServiceUser(path string) error {
	return nil
}
