package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/lazarok09/ahp-method/internal/workspace"
	"github.com/spf13/cobra"
)

func cmdJournal() *cobra.Command {
	c := &cobra.Command{
		Use:   "journal",
		Short: "Inspect compute run history under tmp/journal/",
		Long: `Per-run snapshots of data/*.csv + results (weights, ranking, compute.json).

Journaling is off by default. Enable with --journal on compute, AHP_JOURNAL=1,
or [journal] enabled = true in ahp.toml.

  ahp journal list
  ahp journal show latest
  ahp journal path <id>
  ahp journal prune --keep 20`,
	}
	c.AddCommand(cmdJournalList(), cmdJournalShow(), cmdJournalPath(), cmdJournalPrune())
	return c
}

func cmdJournalList() *cobra.Command {
	c := &cobra.Command{
		Use:           "list",
		Short:         "List journal runs (newest first)",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runJournalList,
	}
	addWorkspaceFlag(c)
	c.Flags().Int("limit", 20, "Max entries (0 = all)")
	return c
}

func runJournalList(cmd *cobra.Command, args []string) error {
	p := cliout.FromCmd(cmd)
	limit, _ := cmd.Flags().GetInt("limit")
	ws, err := openWS(wsFlag(cmd))
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	entries, err := ws.JournalList(limit)
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	if p.JSON {
		return p.PrintJSON(entries)
	}
	if len(entries) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "(no journal entries — enable with ahp compute --journal)")
		return nil
	}
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, []string{
			e.ID, e.Command,
			strconv.FormatBool(e.Complete), strconv.FormatBool(e.Consistent),
			e.LeaderID, fmt.Sprintf("%.4f", e.LeaderWeight), e.CreatedAt,
		})
	}
	return p.PrintTable([]string{"id", "command", "complete", "consistent", "leader", "weight", "created"}, rows)
}

func cmdJournalShow() *cobra.Command {
	c := &cobra.Command{
		Use:           "show [id|latest]",
		Short:         "Show meta + summary for a journal run",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runJournalShow,
	}
	addWorkspaceFlag(c)
	return c
}

func runJournalShow(cmd *cobra.Command, args []string) error {
	p := cliout.FromCmd(cmd)
	id := "latest"
	if len(args) > 0 {
		id = args[0]
	}
	ws, err := openWS(wsFlag(cmd))
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	entry, err := ws.JournalShow(id)
	if err != nil {
		return cliout.Wrap(cliout.ExitIO, err)
	}
	if p.JSON {
		return p.PrintJSON(entry)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "id: %s\n", entry.Meta.ID)
	fmt.Fprintf(cmd.OutOrStdout(), "created: %s\n", entry.Meta.CreatedAt)
	fmt.Fprintf(cmd.OutOrStdout(), "command: %s\n", entry.Meta.Command)
	fmt.Fprintf(cmd.OutOrStdout(), "title: %s\n", entry.Meta.WorkspaceTitle)
	fmt.Fprintf(cmd.OutOrStdout(), "complete=%v consistent=%v\n", entry.Meta.Complete, entry.Meta.Consistent)
	fmt.Fprintf(cmd.OutOrStdout(), "dir: %s\n", entry.Dir)
	if len(entry.Summary.Top) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "top:")
		for _, r := range entry.Summary.Top {
			fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s  %.4f\n", r.Rank, r.Name, r.Weight)
		}
	}
	if entry.Summary.WorstCR != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "worst_cr: %s  %.4f\n", entry.Summary.WorstCR.Matrix, entry.Summary.WorstCR.CR)
	}
	return nil
}

func cmdJournalPath() *cobra.Command {
	c := &cobra.Command{
		Use:           "path [id|latest]",
		Short:         "Print absolute path of a journal run directory",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := "latest"
			if len(args) > 0 {
				id = args[0]
			}
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			path, err := ws.JournalPath(id)
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
	addWorkspaceFlag(c)
	return c
}

func cmdJournalPrune() *cobra.Command {
	c := &cobra.Command{
		Use:           "prune",
		Short:         "Delete oldest journal runs beyond --keep",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := cliout.FromCmd(cmd)
			keep, _ := cmd.Flags().GetInt("keep")
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			removed, err := ws.JournalPrune(keep)
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			if p.JSON {
				return p.PrintJSON(map[string]any{"removed": removed, "keep": keep})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed %d journal run(s) (keep=%d)\n", removed, keep)
			return nil
		},
	}
	addWorkspaceFlag(c)
	c.Flags().Int("keep", 50, "Max runs to retain")
	return c
}

// journalOptsFromCmd builds WriteOutputsOpts from --journal / --no-journal flags.
func journalOptsFromCmd(cmd *cobra.Command, command string) workspace.WriteOutputsOpts {
	opts := workspace.WriteOutputsOpts{
		Command:    command,
		Argv:       append([]string(nil), os.Args...),
		AHPVersion: version,
	}
	if cmd.Flags().Changed("journal") {
		v, _ := cmd.Flags().GetBool("journal")
		opts.Journal = &v
	}
	if cmd.Flags().Changed("no-journal") {
		if no, _ := cmd.Flags().GetBool("no-journal"); no {
			f := false
			opts.Journal = &f
		}
	}
	return opts
}

func addJournalFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("journal", false, "Snapshot this run under tmp/journal/")
	cmd.Flags().Bool("no-journal", false, "Disable journaling for this run")
}
