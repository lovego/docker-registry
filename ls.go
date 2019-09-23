package main

import (
	"encoding/json"
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
	var long bool
	cmd := &cobra.Command{
		Use:   `ls <repository>`,
		Short: `List repositories or images in the registry.`,
		Long: `If repository is privided, all images of the repository is listed,
otherwise all repositories of the registry is listed.`,
		Example: `ls registry.example.com/my/repo`,
		RunE: func(c *cobra.Command, args []string) error {
			reg, name, err := getRegistryAndName(args)
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
			return lsImages(reg, name, long)
		},
	}
	cmd.Flags().BoolVarP(&long, `long`, `l`, false, `use a long listing format`)
	return cmd
}

func lsImages(reg *registry.Registry, name string, long bool) error {
	tags, err := reg.Tags(name)
	if err != nil {
		return err
	}
	if !long {
		fmt.Println(`TAG`)
		fmt.Println(strings.Join(tags, "\n"))
		return nil
	}
	return lsImagesLong(reg, name, tags)
}

func lsImagesLong(reg *registry.Registry, name string, tags []string) error {
	maxLen := 0
	for _, tag := range tags {
		if len(tag) > maxLen {
			maxLen = len(tag)
		}
	}
	if maxLen < 10 {
		maxLen = 10
	} else {
		maxLen += 3
	}

	const format1 = `%-*s`
	const format2 = "%-15s %-10s"
	const format3 = "%s\n"

	fmt.Printf(format1, maxLen, `TAG`)
	fmt.Printf(format2, `IMAGE ID`, `SIZE`)
	fmt.Printf(format3, `CREATED`)

	for _, tag := range tags {
		fmt.Printf(format1, maxLen, tag)

		v, err := reg.ManifestV2(name, tag)
		if err != nil {
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
