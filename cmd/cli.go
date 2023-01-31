package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/skalt/git-cc/pkg/config"
	"github.com/skalt/git-cc/pkg/parser"
)

var version string

func SetVersion(v string) {
	version = v
}

func versionMode() {
	fmt.Printf("git-cc %s\n", version)
}

// construct a shell `git commit` command with flags delegated from the git-cc
// cli
func getGitCommitCmd(cmd *cobra.Command) []string {
	commitCmd := []string{}
	noEdit, _ := cmd.Flags().GetBool("no-edit")
	message, _ := cmd.Flags().GetStringArray("message")
	flags := cmd.Flags()
	for _, name := range boolFlags {
		flag, _ := flags.GetBool(name)
		if flag {
			commitCmd = append(commitCmd, "--"+name)
		}
	}
	if noEdit || len(message) > 0 {
		commitCmd = append(commitCmd, "--no-edit")
	} else {
		commitCmd = append(commitCmd, "--edit")
	}
	return commitCmd
}

// run a potentially interactive `git commit`
func doCommit(message string, dryRun bool, commitParams []string) {
	f := config.GetCommitMessageFile()
	file, err := os.Create(f)
	if err != nil {
		log.Fatalf("unable to create %s: %+v", f, err)
	}
	_, err = file.Write([]byte(message))
	if err != nil {
		log.Fatalf("unable to write to %s: %+v", f, err)
	}
	if dryRun {
		fmt.Println(message)
	}
	cmd := append([]string{"git", "commit", "--message", message}, commitParams...)
	process := exec.Command(cmd[0], cmd[1:]...)
	process.Stdin = os.Stdin
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	if dryRun {
		fmt.Printf("would run: `%s`\n", strings.Join(cmd, " "))
		os.Exit(0)
	} else {
		err = process.Run()
		if err != nil {
			log.Fatalf("failed running `%+v`: %+v", cmd, err)
		} else {
			os.Exit(0)
		}
	}
}

// run the conventional-commit helper logic. This may/not break into the TUI.
func mainMode(cmd *cobra.Command, args []string, cfg *config.Cfg) {

	commitParams := getGitCommitCmd(cmd)
	committingAllChanges, _ := cmd.Flags().GetBool("all")
	if !cfg.DryRun && !committingAllChanges {
		buf := &bytes.Buffer{}
		process := exec.Command("git", "diff", "--name-only", "--cached")
		process.Stdout = buf
		err := process.Run()
		if err != nil {
			log.Fatalf("fatal: not a git repository (or any of the parent directories): .git; %+v", err)
		}
		if buf.String() == "" {
			log.Fatal("No files staged")
		}
	}

	var cc *parser.CC

	message, _ := cmd.Flags().GetStringArray("message")

	if len(message) > 0 {
		cc, _ = parser.ParseAsMuchOfCCAsPossible(strings.Join(message, "\n\n"))
	} else {
		cc, _ = parser.ParseAsMuchOfCCAsPossible((strings.Join(args, " ")))
	}
	valid := cc.MinimallyValid() &&
		cc.ValidCommitType(cfg.CommitTypes) &&
		(cc.ValidScope(cfg.Scopes) || cc.Scope == "")
	if !valid {
		choice := make(chan string, 1)
		m := initialModel(choice, cc, cfg)
		ui := tea.NewProgram(m, tea.WithInputTTY())
		if err := ui.Start(); err != nil {
			log.Fatal(err)
		}
		if result := <-choice; result == "" {
			close(choice)
			os.Exit(1) // no submission
		} else {
			f := config.GetCommitMessageFile()
			file, err := os.Create(f)
			if err != nil {
				log.Fatalf("unable to create fil %s: %+v", f, err)
			}
			_, err = file.Write([]byte(result))
			if err != nil {
				log.Fatalf("unable to write to file %s: %+v", f, err)
			}
			doCommit(result, cfg.DryRun, commitParams)
		}
	} else {
		doCommit(cc.ToString(), cfg.DryRun, commitParams)
	}
}

func redoMessage(cmd *cobra.Command) {
	flags := cmd.Flags()
	msg, _ := flags.GetStringArray("message")
	if len(msg) > 0 {
		log.Fatal("-m|--message is incompatible with --redo")
	}
	commitMessagePath := config.GetCommitMessageFile()
	data, err := os.ReadFile(commitMessagePath)
	// TODO: ignore commented lines in git-commit file
	if err != nil {
		fmt.Printf("file not found: %s", commitMessagePath)
		os.Exit(127)
	}
	empty := true
	for _, line := range strings.Split(
		strings.Trim(string(data), "\r\n\t "),
		"\n",
	) {
		if !strings.HasPrefix(strings.TrimLeft(line, " \t\r\n"), "#") {
			empty = false
			break // ok
		}
	}
	if empty {
		log.Fatalf("Empty commit message: %s", commitMessagePath)
	}

	flags.Set("message", string(data))
}

var Cmd = &cobra.Command{
	Use:   "git-cc",
	Short: "write conventional commits",
	// not using cobra subcommands since they prevent passing arbitrary arguments
	Run: func(cmd *cobra.Command, args []string) {
		flags := cmd.Flags()
		if version, _ := flags.GetBool("version"); version {
			versionMode()
			os.Exit(0)
		} else if genCompletion, _ := flags.GetBool("generate-shell-completion"); genCompletion {
			generateShellCompletion(cmd, args)
			os.Exit(0)
		} else if genManPage, _ := flags.GetBool("generate-man-page"); genManPage {
			generateManPage(cmd, args)
			os.Exit(0)
		} else {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			cfg, err := config.Init(dryRun)
			if err != nil {
				log.Fatal(err)
			}
			if redo, _ := flags.GetBool("redo"); redo {
				redoMessage(cmd)
			}
			mainMode(cmd, args, cfg)
		}
	},
}

func init() {
	flags := Cmd.Flags()
	flags.BoolP("help", "h", false, "print the usage of git-cc")
	flags.Bool("dry-run", false, "Only print the resulting conventional commit message; don't commit.")
	flags.Bool("redo", false, "Reuse your last commit message")
	flags.StringArrayP("message", "m", []string{}, "pass a complete conventional commit. If valid, it'll be committed without editing.")
	flags.Bool("version", false, "print the version")
	// TODO: accept more of git commit's flags; see https://git-scm.com/docs/git-commit
	// likely: --cleanup=<mode>
	// more difficult, and possibly better done manually: --amend, -C <commit>
	// --reuse-message=<commit>, -c <commit>, --reedit-message=<commit>,
	// --fixup=<commit>, --squash=<commit>
	flags.String("author", "", "delegated to git-commit")
	flags.String("date", "", "delegated to git-commit")
	flags.BoolP("all", "a", false, "see the git-commit docs for --all|-a")
	flags.BoolP("signoff", "s", false, "see the git-commit docs for --signoff|-s")
	flags.Bool("no-gpg-sign", false, "see the git-commit docs for --no-gpg-sign")
	flags.Bool("no-post-rewrite", false, "Bypass the post-rewrite hook")
	flags.Bool("no-edit", false, "Use the selected commit message without launching an editor.")
	flags.BoolP("no-verify", "n", false, "Bypass git hooks")
	flags.Bool("verify", true, "Ensure git hooks run")
	// https://git-scm.com/docs/git-commit#Documentation/git-commit.txt---no-verify
	flags.Bool("no-signoff", true, "Don't add a a `Signed-off-by` trailer to the commit message")
	flags.Bool("generate-man-page", false, "Generate a man page in your manpath")
	flags.Bool(
		"generate-shell-completion",
		false,
		"print a bash/zsh/fish/powershell completion script to stdout",
	)
}
