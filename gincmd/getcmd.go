package gincmd

import (
	"fmt"
	"strings"

	ginclient "github.com/G-Node/gin-cli/ginclient"
	"github.com/G-Node/gin-cli/ginclient/config"
	"github.com/G-Node/gin-cli/git"
	"github.com/spf13/cobra"
)

func isValidRepoPath(path string) bool {
	return strings.Contains(path, "/")
}

func getRepo(cmd *cobra.Command, args []string) {
	jsonout, _ := cmd.Flags().GetBool("json")
	repostr := args[0]
	conf := config.Read()
	gincl := ginclient.New(conf.GinHost)
	requirelogin(cmd, gincl, !jsonout)

	if !isValidRepoPath(repostr) {
		Die(fmt.Sprintf("Invalid repository path '%s'. Full repository name should be the owner's username followed by the repository name, separated by a '/'.\nType 'gin help get' for information and examples.", repostr))
	}

	gincl.GitHost = conf.GitHost
	gincl.GitUser = conf.GitUser
	clonechan := make(chan git.RepoFileStatus)
	go gincl.CloneRepo(repostr, clonechan)
	formatOutput(clonechan, 0, jsonout)
	_, err := git.CommitIfNew("origin")
	CheckError(err)
}

// GetCmd sets up the 'get' repository subcommand
func GetCmd() *cobra.Command {
	description := "Download a remote repository to a new directory and initialise the directory with the default options. The local directory is referred to as the 'clone' of the repository."
	args := map[string]string{
		"<repopath>": "The repository path must be specified on the command line. A repository path is the owner's username, followed by a \"/\" and the repository name.",
	}
	examples := map[string]string{
		"Get and initialise the repository named 'example' owned by user 'alice'": "$ gin get alice/example",
		"Get and initialise the repository named 'eegdata' owned by user 'peter'": "$ gin get peter/eegdata",
	}
	var getRepoCmd = &cobra.Command{
		Use:     "get [--json] <repopath>",
		Short:   "Retrieve (clone) a repository from the remote server",
		Long:    formatdesc(description, args),
		Example: formatexamples(examples),
		Args:    cobra.ExactArgs(1),
		Run:     getRepo,
		DisableFlagsInUseLine: true,
	}
	getRepoCmd.Flags().Bool("json", false, "Print output in JSON format.")
	return getRepoCmd
}