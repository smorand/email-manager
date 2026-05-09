package cli

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

//go:embed skill.md
var embeddedSkill string

const (
	skillConfigDir = "email-manager"
	skillRulesFile = "regles-tri.md"
	skillDirPerm   = 0o755
	skillFilePerm  = 0o644
)

var (
	skillCmd = &cobra.Command{
		Use:   "skill",
		Short: "Print the agent skill (mode d'emploi for AI agents)",
		Long: "Print the embedded skill documentation explaining how an AI " +
			"agent should drive email-manager, then concatenate any user " +
			"knowledge files from ~/.config/email-manager/.",
		RunE: runSkill,
	}

	skillLearnCmd = &cobra.Command{
		Use:   "learn",
		Short: "Persist a tri rule learned from the user (requires --rule)",
		RunE:  runSkillLearn,
	}
)

func setupSkillCommand() {
	skillLearnCmd.Flags().StringVar(&ruleText, "rule", "", "Rule text to append (required)")
	_ = skillLearnCmd.MarkFlagRequired("rule")

	skillCmd.AddCommand(skillLearnCmd)
}

func runSkill(cmd *cobra.Command, args []string) error {
	fmt.Println(embeddedSkill)

	dir, err := userConfigDir()
	if err != nil {
		return err
	}

	files, err := listSkillKnowledgeFiles(dir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}

	fmt.Println()
	fmt.Println("---")
	fmt.Println()
	fmt.Printf("# Connaissance utilisateur (depuis %s)\n\n", dir)

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot read %s: %v\n", f, err)
			continue
		}
		fmt.Printf("## %s\n\n", filepath.Base(f))
		fmt.Println(strings.TrimRight(string(content), "\n"))
		fmt.Println()
	}
	return nil
}

func runSkillLearn(cmd *cobra.Command, args []string) error {
	if strings.TrimSpace(ruleText) == "" {
		return fmt.Errorf("--rule must not be empty")
	}

	dir, err := userConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, skillDirPerm); err != nil {
		return fmt.Errorf("error creating config directory %s: %w", dir, err)
	}

	path := filepath.Join(dir, skillRulesFile)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, skillFilePerm)
	if err != nil {
		return fmt.Errorf("error opening %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	stamp := time.Now().Format("2006-01-02 15:04")
	entry := fmt.Sprintf("- [%s] %s\n", stamp, strings.TrimSpace(ruleText))
	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("error appending rule to %s: %w", path, err)
	}

	fmt.Fprintf(os.Stderr, "Rule appended to %s\n", path)
	return nil
}

// userConfigDir returns ~/.config/email-manager (or $XDG_CONFIG_HOME/email-manager
// when set), independent of platform. We deliberately don't use os.UserConfigDir
// because on macOS it returns ~/Library/Application Support, which is not what
// users expect for a CLI tool whose docs reference ~/.config/email-manager.
func userConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, skillConfigDir), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error resolving home directory: %w", err)
	}
	return filepath.Join(home, ".config", skillConfigDir), nil
}

func listSkillKnowledgeFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error reading config directory %s: %w", dir, err)
	}

	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		paths = append(paths, filepath.Join(dir, name))
	}
	sort.Strings(paths)
	return paths, nil
}
