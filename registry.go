package main

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/heroku/docker-registry-client/registry"
	"github.com/lovego/docker_credentials"
	"github.com/spf13/cobra"
)

func main() {
	cobra.EnableCommandSorting = false

	cmd := &cobra.Command{
		Use:   `docker-registry`,
		Short: `manage images in the docker registry v2. (https://docs.docker.com/registry/spec/api/)`,
	}
	cmd.AddCommand(lsCmd(), rmCmd(), manifestCmd(), blobCmd())
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func getRegistry(server string) (*registry.Registry, error) {
	username, password, err := docker_credentials.Of(server)
	if err != nil {
		return nil, err
	}

	reg, err := newRegistry("https://"+server, username, password)
	if err != nil {
		return nil, err
	}
	return reg, nil
}

func newRegistry(registryURL, username, password string) (*registry.Registry, error) {
	url := strings.TrimSuffix(registryURL, "/")
	transport := registry.WrapTransport(http.DefaultTransport, url, username, password)
	reg := &registry.Registry{
		URL:    url,
		Client: &http.Client{Transport: transport},
		Logf:   func(format string, args ...interface{}) {},
	}

	if err := reg.Ping(); err != nil {
		return nil, err
	}

	return reg, nil
}

func splitServerAndName(repository string) (string, string) {
	slice := strings.SplitN(repository, "/", 2)
	if len(slice) == 1 {
		return repository, ""
	}
	return slice[0], slice[1]
}

func splitNameAndTag(name string) (string, string) {
	slice := strings.SplitN(name, ":", 2)
	if len(slice) == 1 {
		return name, ""
	}
	return slice[0], slice[1]
}

func getMaxWidth(rows []string, min int) int {
	max := 0
	for _, row := range rows {
		if len(row) > max {
			max = len(row)
		}
	}
	if max < min {
		return min
	}
	return max
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	urlErr, ok := err.(*url.Error)
	if !ok || urlErr == nil || urlErr.Err == nil {
		return false
	}
	statusErr, ok := urlErr.Err.(*registry.HTTPStatusError)
	if !ok || statusErr == nil || statusErr.Response == nil {
		return false
	}
	return statusErr.Response.StatusCode == http.StatusNotFound
}
