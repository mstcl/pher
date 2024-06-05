package config

import (
	"bytes"
	"fmt"
	"io"
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
	file, err := os.Open(f)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	defer file.Close()
	return handleConfig(file)
}

func handleConfig(file io.Reader) (*Config, error) {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(buf.Bytes(), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
