package main

import (
	"encoding/json"
	"os"
)

const GIM_PATH = "/.gim/"
const HIGHLIGHTING_PATH = GIM_PATH + "highlighting/"
const EDITOR_CONFIG_PATH = GIM_PATH + "config.config"

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
}

// EditorConfig TODO: don't know if this is the best way to go about this
type EditorConfig struct {
	LineNumberColor ColorConfig `json:"line_number"`
	BackgroundColor ColorConfig `json:"background_color"`
	LineNumberWidth int         `json:"line_number_width"`
	TabWidth        int         `json:"tab_width"`
}

func ReadHighlightingConfig(path string) (*HighlightingConfig, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(homedir + HIGHLIGHTING_PATH + path)
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

func EnsureGimFolderExists() error {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path := homedir + GIM_PATH

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			return err
		}
	} else {
		return err
	}
	return nil
}

func getDefaultEditorConfigValues() *EditorConfig {
	return &EditorConfig{
		LineNumberColor: ColorConfig{Color: [3]int{128, 128, 128}},
		BackgroundColor: ColorConfig{Color: [3]int{0, 0, 0}},
		LineNumberWidth: 4,
		TabWidth:        4,
	}
}

func createDefaultEditorConfig() (*EditorConfig, error) {
	config := getDefaultEditorConfigValues()

	homedir, err := os.UserHomeDir()

	if err != nil {
		return config, err
	}

	file, err := os.Create(homedir + EDITOR_CONFIG_PATH)

	if err != nil {
		return config, err
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ") // Set indentation for readability
	err = encoder.Encode(config)
	return config, err
}

func getDefaultHighlightingConfigValues() *HighlightingConfig {
	tokenConfig := TokensConfig{
		Tokens: []string{},
		Color:  [3]int{254, 254, 254},
	}
	colorConfig := ColorConfig{Color: tokenConfig.Color}

	return &HighlightingConfig{
		Literals: tokenConfig,
		Digits:   colorConfig,
		Strings:  colorConfig,
		Default:  colorConfig,
		BuiltIns: tokenConfig,
		Types:    tokenConfig,
		Keywords: tokenConfig,
		Comment:  tokenConfig,
	}
}

func ensureHighlightingFolderExists() error {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path := homedir + HIGHLIGHTING_PATH
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			return err
		}
	} else {
		return err
	}
	return nil
}

func createDefaultHighlightingConfig() (*HighlightingConfig, error) {
	config := getDefaultHighlightingConfigValues()

	homedir, err := os.UserHomeDir()

	if err != nil {
		return config, err
	}

	err = ensureHighlightingFolderExists()
	if err != nil {
		return config, err
	}

	file, err := os.Create(homedir + HIGHLIGHTING_PATH + "default.json")
	if err != nil {
		return config, err
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ") // Set indentation for readability
	err = encoder.Encode(config)
	return config, err
}

func ReadEditorConfig() (*EditorConfig, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(homedir + EDITOR_CONFIG_PATH)
	if err != nil {
		config, err := createDefaultEditorConfig()
		return config, err
	}
	defer f.Close()

	var config EditorConfig
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&config)

	if err != nil {
		return getDefaultEditorConfigValues(), err
	}

	return &config, nil
}
