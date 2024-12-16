package cmd

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/btm6084/utilities/fileutil"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	// VERSION is the current version of goack
	VERSION = `1.0.0`
)

var (
	printLock sync.Mutex

	c            chan bool
	openSearches = 0
	searchLimit  = 10

	filesProcessed = 0

	extensions = regexp.MustCompile(`[.]svg|jpg|png$`)
)

type options struct {
	After        int
	Before       int
	Config       Config
	FileNameOnly bool
	FollowSyms   bool
	Help         bool
	Insensitive  bool
	Inverse      bool
	// IsTerminal records whether we're writing to a terminal or not. e.g. when piping output.
	IsTerminal    bool
	ForceTerminal bool
	MatchOnly     bool
	NoColor       bool
	AllowBinary   bool
	Regex         *regexp.Regexp
	Skip          string
	Term          string
}

func init() {
	searchCmd.Flags().BoolP("help", "h", false, "Display Help Text")
	viper.BindPFlag("help", searchCmd.Flags().Lookup("help"))

	searchCmd.Flags().BoolP("nameonly", "l", false, "Display File Name Only")
	viper.BindPFlag("nameonly", searchCmd.Flags().Lookup("nameonly"))

	searchCmd.Flags().BoolP("follow", "f", false, "Follow Symlinks")
	viper.BindPFlag("follow", searchCmd.Flags().Lookup("follow"))

	searchCmd.Flags().BoolP("insensitive", "i", false, "Case Insensitive Search")
	viper.BindPFlag("insensitive", searchCmd.Flags().Lookup("insensitive"))

	searchCmd.Flags().BoolP("inverse", "v", false, "Print Only Lines that Do Not Match")
	viper.BindPFlag("inverse", searchCmd.Flags().Lookup("inverse"))

	searchCmd.Flags().BoolP("match-only", "m", false, "Print Only the Matching Text")
	viper.BindPFlag("match-only", searchCmd.Flags().Lookup("match-only"))

	searchCmd.Flags().BoolP("no-color", "", false, "Print Lines without Color")
	viper.BindPFlag("no-color", searchCmd.Flags().Lookup("no-color"))

	searchCmd.Flags().IntP("after", "A", 0, "Number of Lines to Print After Matches")
	viper.BindPFlag("after", searchCmd.Flags().Lookup("after"))

	searchCmd.Flags().IntP("before", "B", 0, "Number of Lines to Print Before Matches")
	viper.BindPFlag("before", searchCmd.Flags().Lookup("before"))

	searchCmd.Flags().IntP("context", "C", 0, "Number of Lines to Print Before and After Matches. Overrides Before and After Values")
	viper.BindPFlag("context", searchCmd.Flags().Lookup("context"))

	searchCmd.Flags().StringP("skip", "k", "", "Skip searching files whose filenames contain this string.")
	viper.BindPFlag("skip", searchCmd.Flags().Lookup("skip"))

	searchCmd.Flags().BoolP("binary", "b", false, "Allow searching binary files")
	viper.BindPFlag("binary", searchCmd.Flags().Lookup("binary"))

	searchCmd.Flags().BoolP("terminal", "t", false, "Force Terminal output even when piping")
	viper.BindPFlag("terminal", searchCmd.Flags().Lookup("terminal"))

	c = make(chan bool)
}

var searchCmd = &cobra.Command{
	Args:  cobra.RangeArgs(1, 2),
	Use:   "goack [flags] <search term> [directory]",
	Short: "Search for patterns in text files",
	Long: `Use regular expressions to search text. Defaults to current directory

Version: ` + VERSION,
	Run: search,
}

func search(cmd *cobra.Command, args []string) {
	opts := options{
		After:         viper.GetInt("after"),
		Before:        viper.GetInt("before"),
		Config:        loadConfig(),
		FileNameOnly:  viper.GetBool("nameonly"),
		FollowSyms:    viper.GetBool("follow"),
		Help:          viper.GetBool("help"),
		Insensitive:   viper.GetBool("insensitive"),
		Inverse:       viper.GetBool("inverse"),
		IsTerminal:    terminal.IsTerminal(int(os.Stdout.Fd())),
		ForceTerminal: viper.GetBool("terminal"),
		MatchOnly:     viper.GetBool("match-only"),
		AllowBinary:   viper.GetBool("binary"),
		NoColor:       viper.GetBool("no-color"),
		Skip:          viper.GetString("skip"),
		Term:          args[0],
	}

	ctx := viper.GetInt("context")
	if ctx > 0 {
		opts.After = ctx
		opts.Before = ctx
	}

	if opts.Insensitive {
		opts.Regex = regexp.MustCompile("((?i)" + args[0] + ")")
	} else {
		opts.Regex = regexp.MustCompile("(" + args[0] + ")")
	}

	// Don't color match-only results.
	if opts.MatchOnly {
		opts.NoColor = true
	}

	file := "."
	if len(args) > 1 {
		file = args[1]
	}

	if opts.Help {
		cmd.Help()
		return
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		processFile(os.Stdin, "stdin", &opts, false)
	} else {
		fileSystemSearch(file, &opts)

		// Wait for any open searches to wrap up.
		for i := 0; i < openSearches; i++ {
			<-c
		}
	}
}

// Recursively decend through a directory and search each regular file found.
// If file is a regular file, search it and return.
func fileSystemSearch(file string, opts *options) {
	filesProcessed++
	if !opts.FollowSyms && fileutil.IsSymlink(file) {
		return
	}

	if fileutil.IsDir(file) {
		if opts.Config.IgnoreDir(file) {
			return
		}

		files, err := ioutil.ReadDir(file)
		if err != nil {
			log.Println(err)
			return
		}

		for _, f := range files {
			if opts.Skip != "" && strings.Contains(f.Name(), opts.Skip) {
				continue
			}
			if opts.Config.IgnoreDir(f.Name()) || opts.Config.IgnoreExt(f.Name()) {
				continue
			}

			fileSystemSearch(strings.TrimRight(file, "/")+"/"+strings.TrimLeft(f.Name(), "/"), opts)
		}

		return
	}

	if fileutil.IsFile(file) {
		openSearches++
		if openSearches > searchLimit {
			<-c
			openSearches--
		}

		go fileSearch(file, opts)
		return
	}
}

func fileSearch(file string, opts *options) {
	f, err := os.Open(file)
	if err != nil {
		c <- false
		return
	}
	defer f.Close()

	processFile(f, file, opts, true)
}

func processFile(f *os.File, fileName string, opts *options, async bool) {
	var lines []string
	var lineNums []int
	var lineNum int

	reader := bufio.NewReader(f)
	for {
		// Read the next line of the file.
		s, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			if async {
				c <- false
			}
			return
		}

		// We don't search binary files.
		if !opts.AllowBinary && fileutil.IsBinaryData([]byte(s)) {
			if async {
				c <- false
			}
			return
		}

		// Break if there's nothing to do.
		if len(s) == 0 && err == io.EOF {
			break
		}

		lineNum++
		lines = append(lines, s)

		// Find any matches in a non-inverse situation.
		if !opts.Inverse && opts.Regex.MatchString(s) {
			lineNums = append(lineNums, lineNum)

			if err == io.EOF {
				break
			}

			continue
		}

		// Find any matches in an inverse situation.
		if opts.Inverse && !opts.Regex.MatchString(s) {
			lineNums = append(lineNums, lineNum)

			if err == io.EOF {
				break
			}

			continue
		}
	}

	if len(lineNums) > 0 {
		Print(fileName, lines, lineNums, opts)
	}

	if async {
		c <- true
	}
}

// Execute performs the root command.
func Execute() {
	if err := searchCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Print the matching items from a given file.
func Print(file string, lines []string, lineNums []int, opts *options) {
	file = strings.TrimPrefix(file, "./")

	if opts.IsTerminal && !opts.NoColor {
		file = "\x1b[96m" + file + "\x1b[0m"
	}

	printLock.Lock()

	// Suppress output of the filename when piping / non-terminal.
	if opts.IsTerminal || opts.ForceTerminal {
		fmt.Println(file)
	}

	// Don't process contents if FilenameOnly
	if opts.FileNameOnly {
		printLock.Unlock()
		return
	}

	for _, n := range lineNums {
		// Before
		for i := opts.Before; i > 0; i-- {
			if n-i-1 >= 0 {
				if opts.MatchOnly {
					matches := getMatchingText(lines[n-i-1], opts.Regex)
					for _, m := range matches {
						writeLine(m, strconv.Itoa(n-i)+"-", opts)
					}
				} else {
					writeLine(lines[n-i-1], strconv.Itoa(n-i)+"-", opts)
				}
			}
		}

		if opts.MatchOnly {
			matches := getMatchingText(lines[n-1], opts.Regex)
			for _, m := range matches {
				writeLine(m, strconv.Itoa(n)+"-", opts)
			}
		} else {
			writeLine(lines[n-1], strconv.Itoa(n)+":", opts)
		}

		// After
		for i := 1; i <= opts.After; i++ {
			if n+i < len(lines) {
				if opts.MatchOnly {
					matches := getMatchingText(lines[n+1], opts.Regex)
					for _, m := range matches {
						writeLine(m, strconv.Itoa(n+i)+"-", opts)
					}
				} else {
					writeLine(lines[n+i], strconv.Itoa(n+i)+"+", opts)
				}
			}
		}

		if opts.Before > 0 || opts.After > 0 {
			fmt.Println()
		}

	}

	fmt.Println()
	printLock.Unlock()
}

// getMatchingText returns ONLY the matching text.
func getMatchingText(s string, re *regexp.Regexp) []string {
	var matches []string
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		matches = append(matches, m[0])
	}
	return matches
}

func writeLine(s, l string, opts *options) {
	s = strings.TrimRight(s, "\n")

	if opts.IsTerminal && !opts.NoColor {
		s = opts.Regex.ReplaceAllString(s, "\x1b[30;42m$1\x1b[0m")
		l = "\x1b[93m" + l + "\x1b[0m"
	}

	// Suppress output of line numbers when piping / non-terminal.
	if opts.IsTerminal || opts.ForceTerminal {
		fmt.Println(l, s)
	} else {
		fmt.Println(s)
	}
}
