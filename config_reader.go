package main

import (
	"encoding/json"
	"os"
)

type TokensConfig struct {
	Tokens []string `json:"tokens"`
	Color  [3]int   `json:"color"`
}

type ColorConfig struct {
	Color [3]int `json:"color"`
}

type JSONConfig struct {
	Literals TokensConfig `json:"literals"`
	Digits   ColorConfig  `json:"digits"`
	Strings  ColorConfig  `json:"strings"`
	Default  ColorConfig  `json:"default"`
	BuiltIns TokensConfig `json:"built_ins"`
	Types    TokensConfig `json:"types"`
	Keywords TokensConfig `json:"keywords"`
	Comment  TokensConfig `json:"comment"`
}

func ReadConfig(path string) (*JSONConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var config JSONConfig
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&config)

	if err != nil {
		return nil, err
	}

	return &config, nil
}
