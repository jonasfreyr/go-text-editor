package main

import (
	"encoding/json"
	"os"
)

const HIGHLIGHTING_PATH = "./highlighting/"

type TokensConfig struct {
	Tokens []string `json:"tokens"`
	Color  [3]int   `json:"color"`
}

type ColorConfig struct {
	Color [3]int `json:"color"`
}

type HighlightingConfig struct {
	Literals TokensConfig `json:"literals"`
	Digits   ColorConfig  `json:"digits"`
	Strings  ColorConfig  `json:"strings"`
	Default  ColorConfig  `json:"default"`
	BuiltIns TokensConfig `json:"built_ins"`
	Types    TokensConfig `json:"types"`
	Keywords TokensConfig `json:"keywords"`
	Comment  TokensConfig `json:"comment"`
	LineNr   ColorConfig  `json:"lineNr"`
}

func ReadHighlightingConfig(path string) (*HighlightingConfig, error) {
	f, err := os.Open(HIGHLIGHTING_PATH + path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var config HighlightingConfig
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&config)

	if err != nil {
		return nil, err
	}

	return &config, nil
}

func ReadEditorConfig(path string) {
	// TODO: do the stuff
}
