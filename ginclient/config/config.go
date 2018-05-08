// Package config handles reading of the user configuration for the client.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/G-Node/gin-cli/ginclient/log"
	"github.com/shibukawa/configdir"
	"github.com/spf13/viper"
)

var configDirs = configdir.New("g-node", "gin")

// GinConfiguration holds the client configuration values
type GinConfiguration struct {
	GinHost    string
	GitHost    string
	GitUser    string
	GitHostKey string
	Bin        struct {
		Git      string
		GitAnnex string
		SSH      string
	}
	Annex struct {
		Exclude []string
		MinSize string
	}
}

// pathExists returns true if the path exists
func pathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func findreporoot(path string) (string, error) {
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	gitdir := filepath.Join(path, ".git")
	if pathExists(gitdir) {
		return path, nil
	}
	updir := filepath.Dir(path)
	if updir == path {
		// root reached
		return "", fmt.Errorf("Not a repository")
	}

	return findreporoot(updir)
}

// local configuration cache
var configuration GinConfiguration
var set = false

// Read loads in the configuration from the config file(s) and returns a populated GinConfiguration struct.
// The configuration is cached. Subsequent reads reuse the already loaded configuration.
func Read() GinConfiguration {
	if set {
		return configuration
	}
	viper.Reset()
	viper.SetTypeByDefaultValue(true)
	// Binaries
	viper.SetDefault("bin.git", "git")
	viper.SetDefault("bin.gitannex", "git-annex")
	viper.SetDefault("bin.ssh", "ssh")

	// Hosts
	viper.SetDefault("gin.address", "https://web.gin.g-node.org")
	viper.SetDefault("gin.port", "443")

	viper.SetDefault("git.address", "gin.g-node.org")
	viper.SetDefault("git.port", "22")
	viper.SetDefault("git.user", "git")
	viper.SetDefault("git.hostkey", "gin.g-node.org,141.84.41.216 ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBE5IBgKP3nUryEFaACwY4N3jlqDx8Qw1xAxU2Xpt5V0p9RNefNnedVmnIBV6lA3n+9kT1OSbyqA/+SgsQ57nHo0=")

	// annex filters
	viper.SetDefault("annex.minsize", "10M")

	// Merge in user config file
	confpath, _ := Path(false)
	configFileName := "config.yml"
	confpath = filepath.Join(confpath, configFileName)

	viper.SetConfigFile(confpath)
	cerr := viper.MergeInConfig()
	if cerr == nil {
		log.Write("Found config file %s", confpath)
	}

	configuration.Bin.Git = viper.GetString("bin.git")
	configuration.Bin.GitAnnex = viper.GetString("bin.gitannex")
	configuration.Bin.SSH = viper.GetString("bin.ssh")

	ginAddress := viper.GetString("gin.address")
	ginPort := viper.GetInt("gin.port")
	configuration.GinHost = fmt.Sprintf("%s:%d", ginAddress, ginPort)

	gitAddress := viper.GetString("git.address")
	gitPort := viper.GetInt("git.port")
	configuration.GitHost = fmt.Sprintf("%s:%d", gitAddress, gitPort)
	configuration.GitUser = viper.GetString("git.user")
	configuration.GitHostKey = viper.GetString("git.hostkey")

	// configuration file in the repository root (annex excludes and size threshold only)
	reporoot, err := findreporoot(".")
	if err == nil {
		confpath := filepath.Join(reporoot, configFileName)
		viper.SetConfigFile(confpath)
		cerr = viper.MergeInConfig()
		if cerr == nil {
			log.Write("Found config file %s", confpath)
		}
	}
	configuration.Annex.Exclude = viper.GetStringSlice("annex.exclude")
	configuration.Annex.MinSize = viper.GetString("annex.minsize")

	log.Write("configurationuration values")
	log.Write("%+v", configuration)

	// TODO: Validate URLs on config read
	set = true
	return configuration
}

// Path returns the configuration path where configuration files should be stored.
// If the GIN_CONFIG_DIR environment variable is set, its value is returned, otherwise the platform default is used.
// If create is true and the directory does not exist, the full path is created.
func Path(create bool) (string, error) {
	confpath := os.Getenv("GIN_CONFIG_DIR")
	if confpath == "" {
		confpath = configDirs.QueryFolders(configdir.Global)[0].Path
	}
	var err error
	if create {
		err = os.MkdirAll(confpath, 0755)
		if err != nil {
			return "", fmt.Errorf("could not create config directory %s", confpath)
		}
	}
	return confpath, err
}