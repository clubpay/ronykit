package boxship

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type WaitStrategy string

const (
	WaitForLog           WaitStrategy = "waitForLog"
	WaitForHttp          WaitStrategy = "waitForHttp"
	WaitForExit          WaitStrategy = "waitForExit"
	WaitForHealthCheck   WaitStrategy = "waitForHealthCheck"
	WaitForListeningPort WaitStrategy = "waitForListeningPort"
	WaitForExec          WaitStrategy = "waitForExec"
	WaitForSQL           WaitStrategy = "waitForSQL"
)

// RegistryCred is a dictionary to hold information if Boxship needs to pull the image from
// private docker repositories.
type RegistryCred map[string]string

// GitAuth holds information for authentication with the git repository.
type GitAuth struct {
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

type SetupFile struct {
	DefaultGitAuth      GitAuth                  `yaml:"gitAuth"`
	DefaultRegistryCred RegistryCred             `yaml:"registryCred"`
	Networks            []string                 `yaml:"networks"`
	Containers          map[string]ContainerDesc `yaml:"containers"`
}

var regexEnvKeys = regexp.MustCompile(`\${([a-zA-Z0-9_-]{1,32})}`)

func parseYamlFile(yamlFile string, lookupFunc func(string) string) (*SetupFile, error) {
	yamlBytes, err := os.ReadFile(yamlFile)
	if err != nil {
		return nil, err
	}

	// replace dynamic variables with ENV values
	foundDynamicVars := regexEnvKeys.FindAll(yamlBytes, -1)
	for idx := range foundDynamicVars {
		x := bytes.Trim(foundDynamicVars[idx], "${}")

		foundX := lookupFunc(string(x))
		if foundX == "" {
			return nil, fmt.Errorf("could not find dynamic variable: %s", string(x))
		} else {
			yamlBytes = bytes.ReplaceAll(yamlBytes, foundDynamicVars[idx], []byte(foundX))
		}
	}

	setup := &SetupFile{}

	err = yaml.Unmarshal(yamlBytes, setup)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("path: %s", yamlFile))
	}

	return setup, nil
}

func readYamlDir(
	ctx context.Context,
	dirPath string,
	f func(ctx context.Context, yamlFilePath string) error,
) error {
	fileInfo, err := os.Stat(dirPath)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("must be a directory")
	}

	return filepath.WalkDir(
		dirPath,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			fileInfo, err := d.Info()
			if err != nil {
				return err
			}

			switch strings.ToLower(filepath.Ext(fileInfo.Name())) {
			case "yml", "yaml", ".yml", ".yaml":
				err = f(ctx, path)
				if err != nil {
					return err
				}
			}

			return nil
		},
	)
}
