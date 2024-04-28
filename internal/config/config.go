package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Title         string       `yaml:"title"`
	Description   string       `yaml:"description"`
	Url           string       `yaml:"url"`
	AuthorName    string       `yaml:"authorName"`
	AuthorEmail   string       `yaml:"authorEmail"`
	RootCrumb     string       `yaml:"rootCrumb"`
	CodeHighlight bool         `yaml:"false"`
	Footer        []FooterLink `yaml:"footer"`
	Head          string       `yaml:"head"`
}

type FooterLink struct {
	Href string `yaml:"href"`
	Text string `yaml:"text"`
}

func Read(f string) (*Config, error) {
	b, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := Config{
		CodeHighlight: true,
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}
