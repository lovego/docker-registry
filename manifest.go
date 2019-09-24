package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/spf13/cobra"
)

func manifestCmd() *cobra.Command {
	var v1 bool
	cmd := &cobra.Command{
		Use:   `manifest REPOSITORY[:TAG]`,
		Short: `Show manifest of image in the registry.`,
		Example: strings.TrimPrefix(`
  docker-registry manifest registry.example.com/my/repo:my-tag
  docker-registry manifest registry.example.com/my/repo          # use default "latest" tag`, "\n"),
		RunE: func(c *cobra.Command, args []string) error {
			var server, name string
			if len(args) == 1 {
				server, name = splitServerAndName(args[0])
			}
			if server == `` {
				return errors.New(`one and only one REPOSITORY[:TAG] argument is required.`)
			}
			if name == `` {
				return errors.New(`repository path is required in the agument.`)
			}

			reg, err := getRegistry(server)
			if err != nil {
				return err
			}
			name, tag := splitNameAndTag(name)
			if tag == `` {
				tag = `latest`
			}
			if contetDigest, manifest, err := getManifest(reg, name, tag, v1); err != nil {
				return err
			} else {
				fmt.Println(`Docker-Content-Digest`, contetDigest, "\n", manifest)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&v1, `v1`, false, `show manifest in v1 format, default to v2.`)
	return cmd
}

func getManifest(
	reg *registry.Registry, repository, reference string, v1 bool,
) (string, string, error) {
	url := reg.URL + fmt.Sprintf(`/v2/%s/manifests/%s`, repository, reference)
	reg.Logf(`registry.manifest.get url=%s repository=%s reference=%s`, url, repository, reference)

	req, err := http.NewRequest(`GET`, url, nil)
	if err != nil {
		return ``, ``, err
	}

	var mediaType = schema2.MediaTypeManifest
	if v1 {
		mediaType = schema1.MediaTypeManifest
	}
	req.Header.Set(`Accept`, mediaType)
	resp, err := reg.Client.Do(req)
	if err != nil {
		return ``, ``, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ``, ``, err
	}
	return resp.Header.Get(`Docker-Content-Digest`), string(body), nil
}
