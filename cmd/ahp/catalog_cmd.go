package main

import (
	"fmt"
	"strings"

	"github.com/lazarok09/ahp-method/internal/catalog"
	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/spf13/cobra"
)

func cmdCatalog() *cobra.Command {
	c := &cobra.Command{
		Use:   "catalog",
		Short: "List CLI↔MCP command catalog (source of truth for docs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := cliout.FromCmd(cmd)
			entries := catalog.All()
			if p.JSON {
				return p.PrintJSON(entries)
			}
			rows := make([][]string, 0, len(entries))
			for _, e := range entries {
				rows = append(rows, []string{e.ID, e.CLI, e.MCP, e.Short})
			}
			return p.PrintTable([]string{"id", "cli", "mcp", "short"}, rows)
		},
	}
	return c
}

func cmdDocs() *cobra.Command {
	c := &cobra.Command{
		Use:   "docs [id|cli|mcp]",
		Short: "Show catalog documentation for a command",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := cliout.FromCmd(cmd)
			if len(args) == 0 {
				entries := catalog.All()
				if p.JSON {
					return p.PrintJSON(entries)
				}
				for _, e := range entries {
					fmt.Fprintf(cmd.OutOrStdout(), "%-16s  %s\n", e.ID, e.Short)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "\nTip: ahp docs <id>   or   ahp catalog --json")
				return nil
			}
			e, ok := catalog.Lookup(args[0])
			if !ok {
				// try partial CLI match
				q := args[0]
				for _, cand := range catalog.All() {
					if strings.Contains(cand.CLI, q) || strings.Contains(cand.ID, q) {
						e, ok = cand, true
						break
					}
				}
			}
			if !ok {
				return cliout.NewExitError(cliout.ExitUsage, "unknown catalog entry %q — try: ahp catalog", args[0])
			}
			if p.JSON {
				return p.PrintJSON(e)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n%s\n", e.CLI, e.Short)
			if e.MCP != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "MCP: %s\n", e.MCP)
			}
			if e.Long != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", e.Long)
			}
			if len(e.Examples) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "\nExamples:")
				for _, ex := range e.Examples {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", ex)
				}
			}
			return nil
		},
	}
	return c
}

func cmdCompletion() *cobra.Command {
	c := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `To load completions:

  # bash
  source <(ahp completion bash)

  # zsh
  source <(ahp completion zsh)

  # fish
  ahp completion fish | source

  # powershell
  ahp completion powershell | Out-String | Invoke-Expression`,
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return root.GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return root.GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return cliout.NewExitError(cliout.ExitUsage, "unsupported shell %q", args[0])
			}
		},
	}
	return c
}
