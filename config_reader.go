package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func JoinPath(paths ...string) string {
	return filepath.Join(paths...)
}

var HOME_PATH = "."

const GIM_PATH = ".gim"

var HIGHLIGHTING_PATH = JoinPath(GIM_PATH, "highlighting")
var EDITOR_CONFIG_PATH = JoinPath(GIM_PATH, "config.config")

var config *EditorConfig

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
	FolderColor     ColorConfig `json:"folder_color"`
	FileColor       ColorConfig `json:"file_color"`
}

func InitHomeFolder() {
	homeDir, err := os.UserHomeDir()
	if err == nil && !DEBUG_MODE {
		HOME_PATH = homeDir
	}
}

func GetGimPath() string {
	return JoinPath(HOME_PATH, GIM_PATH)
}

func getHomePath() string {
	return HOME_PATH
}

func ReadHighlightingConfig(path string) (*HighlightingConfig, error) {
	homedir := getHomePath()

	f, err := os.Open(JoinPath(homedir, HIGHLIGHTING_PATH, path))
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
	path := JoinPath(getHomePath(), GIM_PATH)

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
		LineNumberWidth: 5,
		TabWidth:        4,
		FolderColor:     ColorConfig{Color: [3]int{104, 151, 187}},
		FileColor:       ColorConfig{Color: [3]int{254, 254, 254}},
	}
}

func createDefaultEditorConfig() (*EditorConfig, error) {
	config := getDefaultEditorConfigValues()

	file, err := os.Create(JoinPath(getHomePath(), EDITOR_CONFIG_PATH))

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
	path := JoinPath(getHomePath(), HIGHLIGHTING_PATH)
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

	err := ensureHighlightingFolderExists()
	if err != nil {
		return config, err
	}

	file, err := os.Create(JoinPath(getHomePath(), HIGHLIGHTING_PATH, "default.json"))
	if err != nil {
		return config, err
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ") // Set indentation for readability
	err = encoder.Encode(config)
	return config, err
}

func GetEditorConfig() *EditorConfig {
	if config == nil {
		return getDefaultEditorConfigValues()
	}

	return config
}

func ReadEditorConfig() {
	f, err := os.Open(JoinPath(getHomePath(), EDITOR_CONFIG_PATH))
	if err != nil {
		config, err = createDefaultEditorConfig()
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&config)

	if err != nil {
		return
	}

}
