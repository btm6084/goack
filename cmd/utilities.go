package cmd

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/btm6084/utilities/inarray"
	homedir "github.com/mitchellh/go-homedir"
)

// Config contains the structure of the configuration file located at /home/$USER/.goackrc/config.json
type Config struct {
	IgnoreDirs []string `json:"ignore-dirs"`
	IgnoreExts []string `json:"ignore-exts"`

	extensions *regexp.Regexp
}

// IgnoreDir returns true if the given filename appears in the IgnoreDirs array
func (c *Config) IgnoreDir(file string) bool {
	return inarray.Strings(file, c.IgnoreDirs) >= 0
}

// IgnoreExt returns true if a file should be ignored based on its extension.
func (c *Config) IgnoreExt(file string) bool {
	if filepath.Ext(file) == "" {
		return false
	}

	return c.extensions.MatchString(filepath.Ext(file))
}

func loadConfig() Config {
	// Set up configuration
	home, _ := homedir.Dir()
	file := home + "/.goackrc/config.json"

	var c Config

	data, _ := ioutil.ReadFile(file)
	json.Unmarshal(data, &c)

	c.extensions = regexp.MustCompile(`[.]` + strings.Join(c.IgnoreExts, "|") + `$`)

	// ALWAYS block .git from consideration. CD into the .git directory if you want to search it.
	c.IgnoreDirs = append(c.IgnoreDirs, ".git")

	return c
}
