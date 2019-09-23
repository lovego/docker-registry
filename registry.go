package main

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/heroku/docker-registry-client/registry"
	"github.com/lovego/docker_credentials"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:   `docker-registry`,
		Short: `manage images in the docker registry v2. (https://docs.docker.com/registry/spec/api/)`,
	}
	cmd.AddCommand(lsCmd())
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func getRegistryAndName(args []string) (*registry.Registry, string, error) {
	if len(args) != 1 {
		return nil, "", errors.New(`one and only one repository argument is required.`)
	}
	var repository = args[0]
	if repository == `` {
		return nil, "", errors.New(`repository can't be empty.`)
	}

	server, name := splitServerAndName(repository)
	username, password, err := docker_credentials.Of(server)
	if err != nil {
		return nil, "", err
	}

	reg, err := newRegistry("https://"+server, username, password)
	if err != nil {
		return nil, "", err
	}
	return reg, name, nil
}

func splitServerAndName(repository string) (string, string) {
	slice := strings.SplitN(repository, "/", 2)
	if len(slice) == 1 {
		return repository, ""
	}
	return slice[0], slice[1]
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
