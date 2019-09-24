package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/heroku/docker-registry-client/registry"
	digest "github.com/opencontainers/go-digest"
	"github.com/spf13/cobra"
)

func lsCmd() *cobra.Command {
	var onlyTags bool
	cmd := &cobra.Command{
		Use:   `ls REPOSITORY`,
		Short: `List repositories or images in the registry.`,
		Example: strings.TrimPrefix(`
  docker-registry ls registry.example.com                # list all repositories of the registry
  docker-registry ls registry.example.com/my/repo        # list all images       of the repository
  docker-registry ls registry.example.com/my/repo:my-tag # list one image        of the repository`,
			"\n"),
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] == `` {
				return errors.New(`one and only one REPOSITORY argument is required.`)
			}
			server, name := splitServerAndName(args[0])
			reg, err := getRegistry(server)
			if err != nil {
				return err
			}
			if name == "" {
				repositories, err := reg.Repositories()
				if err != nil {
					return err
				}
				fmt.Println(strings.Join(repositories, "\n"))
				return nil
			}
			return lsImages(reg, name, onlyTags)
		},
	}
	cmd.Flags().BoolVarP(&onlyTags, `tags`, `t`, false, `list only tags`)
	return cmd
}

func lsImages(reg *registry.Registry, name string, onlyTags bool) error {
	var tags []string
	name, tag := splitNameAndTag(name)
	if tag == `` {
		if _tags, err := reg.Tags(name); err != nil {
			return err
		} else {
			tags = _tags
		}
	} else {
		tags = []string{tag}
	}
	if onlyTags {
		fmt.Println(`TAG`)
		fmt.Println(strings.Join(tags, "\n"))
		return nil
	}
	return lsImagesLong(reg, name, tags)
}

func lsImagesLong(reg *registry.Registry, name string, tags []string) error {
	width := getMaxWidth(tags, 5) + 3

	const format1 = `%-*s`
	const format2 = "%-15s %-10s"
	const format3 = "%s\n"

	fmt.Printf(format1, width, `TAG`)
	fmt.Printf(format2, `IMAGE ID`, `SIZE`)
	fmt.Printf(format3, `CREATED`)

	for _, tag := range tags {
		fmt.Printf(format1, width, tag)

		v, err := reg.ManifestV2(name, tag)
		if err != nil {
			if isNotFound(err) {
				fmt.Println(`*** Not Found ***`)
				continue
			}
			return err
		}
		digest := v.Manifest.Config.Digest
		fmt.Printf(format2, getImageId(string(digest)), getImageSize(v.Manifest))

		created, err := getImageCreatedTime(reg, name, digest)
		if err != nil {
			return err
		}
		fmt.Printf(format3, created)
	}
	return nil
}

func getImageCreatedTime(reg *registry.Registry, name string, digest digest.Digest) (string, error) {
	response, err := reg.DownloadBlob(name, digest)
	body, err := ioutil.ReadAll(response)
	if err != nil {
		return ``, err
	}

	var data struct{ Created time.Time }
	if err := json.Unmarshal(body, &data); err != nil {
		return ``, err
	}
	return data.Created.Local().Format(time.RFC3339), nil
}

func getImageId(digest string) string {
	index := strings.Index(digest, ":")
	if index < 0 {
		return digest
	}
	return digest[index+1:][:12]
}

func getImageSize(manifest schema2.Manifest) string {
	var size = manifest.Config.Size
	for _, layer := range manifest.Layers {
		size += layer.Size
	}
	return readableSize(size)
}

func readableSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%dB", size)
	}
	div, exp := unit, 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(size)/float64(div), "KMGTPE"[exp])
}
