package main

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/heroku/docker-registry-client/registry"
	digest "github.com/opencontainers/go-digest"
	"github.com/spf13/cobra"
)

func rmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   `rm REPOSITORY[:TAG]`,
		Short: `Delete repository or images in the registry.`,
		Example: strings.TrimPrefix(`
  docker-registry rm registry.example.com/my/repo:my-tag  # delete one image        from the repository
  docker-registry rm registry.example.com/my/repo         # delete one repository   from the registry
  docker-registry rm registry.example.com                 # delete all repositories from the registry`,
			"\n"),
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] == `` {
				return errors.New(`one and only one REPOSITORY[:TAG] argument is required.`)
			}
			server, name := splitServerAndName(args[0])
			reg, err := getRegistry(server)
			if err != nil {
				return err
			}

			if name == "" {
				return rmAllRepositories(reg)
			}
			name, tag := splitNameAndTag(name)
			if tag == "" {
				return rmRepository(reg, name, 0)
			}
			return rmImage(reg, name, tag)
		},
	}
	return cmd
}

func rmAllRepositories(reg *registry.Registry) error {
	repositories, err := reg.Repositories()
	if err != nil {
		return err
	}
	width := getMaxWidth(repositories, 8) + 3

	for _, repository := range repositories {
		if err := rmRepository(reg, repository, width); err != nil {
			return err
		}
	}
	return nil
}

func rmRepository(reg *registry.Registry, name string, width int) error {
	tags2digest, err := getTags2DigestMap(reg, name)
	if err != nil {
		return err
	}
	tagsList := make([]string, 0, len(tags2digest))
	for tags := range tags2digest {
		tagsList = append(tagsList, tags)
	}
	sort.Strings(tagsList)

	if width == 0 {
		width = 10
	}
	for _, tags := range tagsList {
		if err := reg.DeleteManifest(name, tags2digest[tags]); err != nil {
			return err
		}
		fmt.Printf("deleted %s/%-*s tags: %s\n", reg.URL, width, name, tags)
	}
	return nil
}

func rmImage(reg *registry.Registry, name, tag string) error {
	digest2tags, err := getDigest2TagsMap(reg, name)
	if err != nil {
		return err
	}
	fmt.Println(digest2tags)
	digest, hasOtherTags := getDigest(digest2tags, tag)

	if hasOtherTags {
		// put empty manifest to the tag to delete only the tag, not the underline image.
		if err := reg.PutManifest(name, tag, schema2.DeserializedManifest{}); err != nil {
			return err
		}
		newDigest, err := reg.ManifestDigest(name, tag)
		if err != nil {
			return err
		}
		if newDigest == digest {
			return errors.New("newDigest = digest ")
		}
		digest = newDigest
	}
	if err := reg.DeleteManifest(name, digest); err != nil {
		return err
	}
	fmt.Printf("deleted %s/%s tags: %s\n", reg.URL, name, tag)
	return nil
}

func getDigest2TagsMap(reg *registry.Registry, name string) (map[digest.Digest][]string, error) {
	tags, err := reg.Tags(name)
	if err != nil {
		return nil, err
	}
	m := make(map[digest.Digest][]string, len(tags))
	for _, tag := range tags {
		digest, err := getManifestV2Digest(reg, name, tag)
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return nil, err
		}

		if m[digest] == nil {
			m[digest] = []string{tag}
		} else {
			m[digest] = append(m[digest], tag)
		}
	}
	return m, nil
}

func getManifestV2Digest(reg *registry.Registry, repository, reference string) (digest.Digest, error) {
	url := reg.URL + fmt.Sprintf("/v2/%s/manifests/%s", repository, reference)
	reg.Logf("registry.manifest.head url=%s repository=%s reference=%s", url, repository, reference)

	req, err := http.NewRequest(`HEAD`, url, nil)
	if err != nil {
		return ``, err
	}

	req.Header.Set(`Accept`, schema2.MediaTypeManifest)
	resp, err := reg.Client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", err
	}
	return digest.Parse(resp.Header.Get("Docker-Content-Digest"))
}

func getTags2DigestMap(reg *registry.Registry, name string) (map[string]digest.Digest, error) {
	digest2tags, err := getDigest2TagsMap(reg, name)
	if err != nil {
		return nil, err
	}

	tags2digest := make(map[string]digest.Digest, len(digest2tags))
	for digest, tags := range digest2tags {
		tags2digest[strings.Join(tags, `, `)] = digest
	}
	return tags2digest, nil
}

func getDigest(digest2tags map[digest.Digest][]string, targetTag string) (digest.Digest, bool) {
	for digest, tags := range digest2tags {
		for _, tag := range tags {
			if tag == targetTag {
				return digest, len(tags) > 1
			}
		}
	}
	return ``, false
}
