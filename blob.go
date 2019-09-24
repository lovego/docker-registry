package main

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/heroku/docker-registry-client/registry"
	digest "github.com/opencontainers/go-digest"
	"github.com/spf13/cobra"
)

func blobCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     `blob REPOSITORY DIGEST`,
		Short:   `Display blob content of repository in the registry. Redirect should be used for binary object.`,
		Example: `docker-registry blob registry.example.com/my/repo sha256:2ca708c1c9ccc509b070f226d6e4712604e0c48b55d7d8f5adc9be4a4d36029a`,
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) != 2 {
				return errors.New(`two arguments required.`)
			}
			server, name := splitServerAndName(args[0])
			if server == `` {
				return errors.New(`one and only one REPOSITORY[:TAG] argument is required.`)
			}
			if name == `` {
				return errors.New(`repository path is required in the first agument.`)
			}

			reg, err := getRegistry(server)
			if err != nil {
				return err
			}

			if blob, err := getBlob(reg, name, args[1]); err != nil {
				return err
			} else {
				fmt.Println(blob)
			}
			return nil
		},
	}
	return cmd
}

func getBlob(reg *registry.Registry, repository, digestStr string) (string, error) {
	digest := digest.Digest(digestStr)
	if err := digest.Validate(); err != nil {
		return "", err
	}
	reader, err := reg.DownloadBlob(repository, digest)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
