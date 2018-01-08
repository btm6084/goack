package cmd

import (
	"encoding/json"
	"io/ioutil"

	"github.com/btm6084/utilities/inarray"
	homedir "github.com/mitchellh/go-homedir"
)

// Config contains the structure of the configuration file located at /home/$USER/.goackrc/config.json
type Config struct {
	IgnoreDirs []string `json:"ignore-dirs"`
}

// IgnoreDir returns true if the given filename appears in the IgnoreDirs array
func (c *Config) IgnoreDir(file string) bool {
	return inarray.Strings(file, c.IgnoreDirs) >= 0
}

func loadConfig() Config {
	// Set up configuration
	home, _ := homedir.Dir()
	file := home + "/.goackrc/config.json"

	var c Config
	var tmp map[string][]string

	data, _ := ioutil.ReadFile(file)
	json.Unmarshal(data, &tmp)

	c.IgnoreDirs = tmp["ignore-dir"]

	// ALWAYS block .git from consideration. CD into the .git directory if you want to search it.
	c.IgnoreDirs = append(c.IgnoreDirs, ".git")

	return c
}
