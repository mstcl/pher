package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Title         string       `yaml:"title"`
	Description   string       `yaml:"description"`
	Url           string       `yaml:"url"`
	AuthorName    string       `yaml:"authorName"`
	AuthorEmail   string       `yaml:"authorEmail"`
	RootCrumb     string       `yaml:"rootCrumb"`
	Head          string       `yaml:"head"`
	Footer        []FooterLink `yaml:"footer"`
	CodeHighlight bool         `yaml:"codeHighlight"`
	IsExt         bool         `yaml:"keepExtension"`
}

type FooterLink struct {
	Href string `yaml:"href"`
	Text string `yaml:"text"`
}

func DefaultConfig() Config {
	return Config{
		CodeHighlight: true,
		IsExt:         true,
		RootCrumb:     "~",
	}
}

func Read(f string) (*Config, error) {
	b, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}
