package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
	"github.com/idursun/jjui/internal/ui/actions"
)

func getConfigFilePath() string {
	var configDirs []string

	// useful during development or other non-standard setups.
	if dir := os.Getenv("JJUI_CONFIG_DIR"); dir != "" {
		if s, err := os.Stat(dir); err == nil && s.IsDir() {
			configDirs = append(configDirs, dir)
		}
	}

	// os.UserConfigDir() already does this for linux leaving darwin to handle
	if runtime.GOOS == "darwin" {
		configDirs = append(configDirs, path.Join(os.Getenv("HOME"), ".config"))
		xdgConfigDir := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigDir != "" {
			configDirs = append(configDirs, xdgConfigDir)
		}
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	configDirs = append(configDirs, configDir)

	for _, dir := range configDirs {
		configPath := filepath.Join(dir, "jjui", "config.toml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	if len(configDirs) > 0 {
		return filepath.Join(configDirs[0], "jjui", "config.toml")
	}
	return ""
}

func loadDefaultConfig() *Config {
	data, err := configFS.ReadFile("default/config.toml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: no embedded default config found: %v\n", err)
		os.Exit(1)
	}

	config := &Config{}
	if err := config.Load(string(data)); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: failed to load embedded default config: %v\n", err)
		os.Exit(1)
	}

	return config
}

func (c *Config) Load(data string) error {
	var err error

	_, err = toml.Decode(data, c)
	if err != nil {
		return err
	}
	for id, action := range c.Actions {
		if action.Id == "" {
			action.Id = id
			c.Actions[id] = action
		}
	}

	actions.Registry = c.Actions

	return nil
}

func LoadConfigFile() ([]byte, error) {
	configFile := getConfigFilePath()
	_, err := os.Stat(configFile)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func loadTheme(data []byte, base map[string]Color) (map[string]Color, error) {
	colors := make(map[string]Color)
	for key, color := range base {
		colors[key] = color
	}
	err := toml.Unmarshal(data, &colors)
	if err != nil {
		return nil, err
	}
	return colors, nil
}

func LoadEmbeddedTheme(name string) (map[string]Color, error) {
	embeddedPath := "default/" + name + ".toml"
	data, err := configFS.ReadFile(embeddedPath)
	if err != nil {
		return nil, err
	}
	return loadTheme(data, nil)
}

func LoadTheme(name string, base map[string]Color) (map[string]Color, error) {
	configFilePath := getConfigFilePath()
	themeFile := filepath.Join(filepath.Dir(configFilePath), "themes", name+".toml")

	data, err := os.ReadFile(themeFile)
	if err != nil {
		return nil, err
	}
	return loadTheme(data, base)
}
