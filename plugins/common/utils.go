package common

import (
	"errors"
	artifactoryUtils "github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jfrog/jfrog-cli-core/v2/common/cliutils"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"golang.org/x/exp/slices"
)

const (
	minSplit              = "min-split"
	DownloadMinSplitKb    = 5120
	DownloadSplitCount    = 3
	DownloadMaxSplitCount = 15
)

func GetStringsArrFlagValue(c *components.Context, flagName string) (resultArray []string) {
	if c.IsFlagSet(flagName) {
		resultArray = append(resultArray, strings.Split(c.GetStringFlagValue(flagName), ";")...)
	}
	return
}

// If `fieldName` exist in the cli args, read it to `field` as an array split by `;`.
func OverrideArrayIfSet(field *[]string, c *components.Context, fieldName string) {
	if c.IsFlagSet(fieldName) {
		*field = append([]string{}, strings.Split(c.GetStringFlagValue(fieldName), ";")...)
	}
}

// If `fieldName` exist in the cli args, read it to `field` as a int.
func OverrideIntIfSet(field *int, c *components.Context, fieldName string) {
	if c.IsFlagSet(fieldName) {
		value, err := strconv.ParseInt(c.GetStringFlagValue(fieldName), 0, 64)
		if err != nil {
			return
		}
		*field = int(value)
	}
}

// If `fieldName` exist in the cli args, read it to `field` as a string.
func OverrideStringIfSet(field *string, c *components.Context, fieldName string) {
	if c.IsFlagSet(fieldName) {
		*field = c.GetStringFlagValue(fieldName)
	}
}

// Get a secret value from a flag or from stdin.
func HandleSecretInput(c *components.Context, stringFlag, stdinFlag string) (secret string, err error) {
	return cliutils.HandleSecretInput(stringFlag, c.GetStringFlagValue(stringFlag), stdinFlag, c.GetBoolFlagValue(stdinFlag))
}

func RunCmdWithDeprecationWarning(cmdName, oldSubcommand string, c *components.Context,
	cmd func(c *components.Context) error) error {
	cliutils.LogNonNativeCommandDeprecation(cmdName, oldSubcommand)
	return cmd(c)
}

func GetThreadsCount(c *components.Context) (threads int, err error) {
	return cliutils.GetThreadsCount(c.GetStringFlagValue("threads"))
}

func GetPrintCurrentCmdHelp(c *components.Context) func() error {
	return func() error {
		return c.PrintCommandHelp(c.CommandName)
	}
}

// This function checks whether the command received --help as a single option.
// If it did, the command's help is shown and true is returned.
// This function should be used iff the SkipFlagParsing option is used.
func ShowCmdHelpIfNeeded(c *components.Context, args []string) (bool, error) {
	return cliutils.ShowCmdHelpIfNeeded(args, GetPrintCurrentCmdHelp(c))
}

func PrintHelpAndReturnError(msg string, context *components.Context) error {
	return cliutils.PrintHelpAndReturnError(msg, GetPrintCurrentCmdHelp(context))
}

func WrongNumberOfArgumentsHandler(context *components.Context) error {
	return cliutils.WrongNumberOfArgumentsHandler(len(context.Arguments), GetPrintCurrentCmdHelp(context))
}

func ExtractArguments(context *components.Context) []string {
	return slices.Clone(context.Arguments)
}

// Return a sorted list of a command's flags by a given command key.
func GetCommandFlags(cmdKey string, commandToFlags map[string][]string, flagsMap map[string]components.Flag) []components.Flag {
	flagList, ok := commandToFlags[cmdKey]
	if !ok {
		log.Error("The command \"", cmdKey, "\" is not found in commands flags map.")
		return nil
	}
	return buildAndSortFlags(flagList, flagsMap)
}

func buildAndSortFlags(keys []string, flagsMap map[string]components.Flag) (flags []components.Flag) {
	for _, flag := range keys {
		flags = append(flags, flagsMap[flag])
	}
	sort.Slice(flags, func(i, j int) bool { return flags[i].GetName() < flags[j].GetName() })
	return
}

// This function indicates whether the command should be executed without
// confirmation warning or not.
// If the --quiet option was sent, it is used to determine whether to prompt the confirmation or not.
// If not, the command will prompt the confirmation, unless the CI environment variable was set to true.
func GetQuietValue(c *components.Context) bool {
	if c.IsFlagSet("quiet") {
		return c.GetBoolFlagValue("quiet")
	}

	return getCiValue()
}

// Return true if the CI environment variable was set to true.
func getCiValue() bool {
	var ci bool
	var err error
	if ci, err = clientutils.GetBoolEnvValue(coreutils.CI, false); err != nil {
		return false
	}
	return ci
}

// Get project key from flag or environment variable
func GetProject(c *components.Context) string {
	projectKey := c.GetStringFlagValue("project")
	return getOrDefaultEnv(projectKey, coreutils.Project)
}

// Return argument if not empty or retrieve from environment variable
func getOrDefaultEnv(arg, envKey string) string {
	if arg != "" {
		return arg
	}
	return os.Getenv(envKey)
}

func CreateDownloadConfiguration(c *components.Context) (downloadConfiguration *artifactoryUtils.DownloadConfiguration, err error) {
	downloadConfiguration = new(artifactoryUtils.DownloadConfiguration)
	downloadConfiguration.MinSplitSize, err = getMinSplit(c, DownloadMinSplitKb)
	if err != nil {
		return nil, err
	}
	downloadConfiguration.SplitCount, err = getSplitCount(c, DownloadSplitCount, DownloadMaxSplitCount)
	if err != nil {
		return nil, err
	}
	downloadConfiguration.Threads, err = GetThreadsCount(c)
	if err != nil {
		return nil, err
	}
	downloadConfiguration.SkipChecksum = c.GetBoolFlagValue("skip-checksum")
	downloadConfiguration.Symlink = true
	return
}

func getMinSplit(c *components.Context, defaultMinSplit int64) (minSplitSize int64, err error) {
	minSplitSize = defaultMinSplit
	if c.GetStringFlagValue(minSplit) != "" {
		minSplitSize, err = strconv.ParseInt(c.GetStringFlagValue(minSplit), 10, 64)
		if err != nil {
			err = errors.New("The '--min-split' option should have a numeric value. " + GetDocumentationMessage())
			return 0, err
		}
	}

	return minSplitSize, nil
}

func GetDocumentationMessage() string {
	return "You can read the documentation at " + coreutils.JFrogHelpUrl + "jfrog-cli"
}

func getSplitCount(c *components.Context, defaultSplitCount, maxSplitCount int) (splitCount int, err error) {
	splitCount = defaultSplitCount
	err = nil
	if c.GetStringFlagValue("split-count") != "" {
		splitCount, err = strconv.Atoi(c.GetStringFlagValue("split-count"))
		if err != nil {
			err = errors.New("The '--split-count' option should have a numeric value. " + GetDocumentationMessage())
		}
		if splitCount > maxSplitCount {
			err = errors.New("The '--split-count' option value is limited to a maximum of " + strconv.Itoa(maxSplitCount) + ".")
		}
		if splitCount < 0 {
			err = errors.New("the '--split-count' option cannot have a negative value")
		}
	}
	return
}
