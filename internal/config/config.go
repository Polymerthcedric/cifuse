package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/Polymerthcedric/cifuse/internal/audit"
)

const Filename = ".cifuserc.yml"

type File struct {
	Ignore audit.Ignore `yaml:"ignore"`
}

func Load(path string) (audit.Ignore, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return audit.Ignore{}, err
	}
	var file File
	if err := yaml.Unmarshal(content, &file); err != nil {
		return audit.Ignore{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return file.Ignore, nil
}
