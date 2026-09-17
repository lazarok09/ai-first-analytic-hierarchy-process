package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lazarok09/ahp-method/internal/cliout"
	ahpmcp "github.com/lazarok09/ahp-method/internal/mcp"
	"github.com/lazarok09/ahp-method/internal/render"
	"github.com/lazarok09/ahp-method/internal/workspace"
	"github.com/spf13/cobra"
)

// Overridden at release via -ldflags "-X main.version=…".
var version = "0.3.0"

func main() {
	root := &cobra.Command{
		Use:   "ahp",
		Short: "Analytic Hierarchy Process CLI: CSV workspace, math, HTML report, MCP",
	}
	root.PersistentFlags().Bool("json", false, "Machine-readable JSON output (default when stdout is not a TTY)")
	root.AddCommand(
		cmdVersion(),
		cmdInit(),
		cmdCompute(),
		cmdRender(),
		cmdDoctor(),
		cmdValidate(),
		cmdStatus(),
		cmdTree(),
		cmdNext(),
		cmdOpen(),
		cmdPair(),
		cmdPlan(),
		cmdApply(),
		cmdGet(),
		cmdDescribe(),
		cmdCatalog(),
		cmdDocs(),
		cmdCompletion(),
		cmdCommitProposals(), // deprecated alias → apply
		cmdSetGoal(),
		cmdAddCriterion(),
		cmdAddAlternative(),
		cmdSetPairwise(), // deprecated alias → pair set
		cmdSetAttribute(),
		cmdMCP(),
	)
	if err := root.Execute(); err != nil {
		var ee *cliout.ExitError
		if !(errors.As(err, &ee) && ee != nil && ee.Err == nil) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(cliout.Code(err))
	}
}

func wsFlag(cmd *cobra.Command) string {
	v, _ := cmd.Flags().GetString("workspace")
	return v
}

func openWS(path string) (*workspace.Workspace, error) {
	root := path
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		root = workspace.Find(cwd)
	}
	ws := workspace.Open(root)
	if _, err := os.Stat(ws.TomlPath()); err != nil {
		return nil, fmt.Errorf("no ahp.toml under %s — run `ahp init` first", ws.Root)
	}
	return ws, nil
}

func addWorkspaceFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("workspace", "w", "", "Workspace directory (finds ahp.toml upward)")
}

func addIncludeFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("include-proposals", false, "Include proposal judgments in compute")
}

func cmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
}

func cmdInit() *cobra.Command {
	var title, description string
	c := &cobra.Command{
		Use:   "init [path]",
		Short: "Create ahp.toml and empty data/*.csv files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) == 1 {
				path = args[0]
			}
			ws := workspace.Open(path)
			if err := ws.EnsureLayout(); err != nil {
				return err
			}
			if err := ws.SaveMeta(workspace.Meta{Title: title, Description: description}); err != nil {
				return err
			}
			fmt.Printf("Initialized AHP workspace at %s\n", ws.Root)
			return nil
		},
	}
	c.Flags().StringVar(&title, "title", "Untitled decision", "Decision title")
	c.Flags().StringVar(&description, "description", "", "Decision description")
	return c
}

func cmdCompute() *cobra.Command {
	c := &cobra.Command{
		Use:   "compute",
		Short: "Solve eigenvectors, CR, synthesis; write output CSVs + HTML",
		RunE: func(cmd *cobra.Command, args []string) error {
			include, _ := cmd.Flags().GetBool("include-proposals")
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return err
			}
			result, err := ws.Compute(include)
			if err != nil {
				return err
			}
			paths, err := ws.WriteOutputs(result, render.HTML(result))
			if err != nil {
				return err
			}
			fmt.Printf("complete=%v consistent=%v\n", result.Complete, result.Consistent)
			for _, w := range result.Warnings {
				fmt.Println("warning:", w)
			}
			for _, r := range result.Ranking {
				fmt.Printf("%d. %s  %.4f\n", r.Rank, r.Name, r.Weight)
			}
			fmt.Println("report:", paths["report_html"])
			fmt.Println("weights:", paths["weights_csv"])
			fmt.Println("ranking:", paths["ranking_csv"])
			return nil
		},
	}
	addWorkspaceFlag(c)
	addIncludeFlag(c)
	return c
}

func cmdRender() *cobra.Command {
	c := &cobra.Command{
		Use:   "render",
		Short: "Recompute and write output/report.html",
		RunE:  cmdCompute().RunE,
	}
	addWorkspaceFlag(c)
	addIncludeFlag(c)
	return c
}

func cmdOpen() *cobra.Command {
	c := &cobra.Command{
		Use:   "open",
		Short: "Recompute and print report.html path (open in browser manually if needed)",
		RunE: func(cmd *cobra.Command, args []string) error {
			include, _ := cmd.Flags().GetBool("include-proposals")
			noCompute, _ := cmd.Flags().GetBool("no-compute")
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return err
			}
			report := filepath.Join(ws.OutputDir(), "report.html")
			if !noCompute {
				result, err := ws.Compute(include)
				if err != nil {
					return err
				}
				paths, err := ws.WriteOutputs(result, render.HTML(result))
				if err != nil {
					return err
				}
				report = paths["report_html"]
			}
			abs, _ := filepath.Abs(report)
			fmt.Println("report:", abs)
			fmt.Println("file://" + filepath.ToSlash(abs))
			return nil
		},
	}
	addWorkspaceFlag(c)
	addIncludeFlag(c)
	c.Flags().Bool("no-compute", false, "Do not recompute before opening")
	return c
}

func cmdSetGoal() *cobra.Command {
	c := &cobra.Command{
		Use:   "set-goal [title]",
		Short: "Set decision title/description in ahp.toml",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			desc, _ := cmd.Flags().GetString("description")
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return err
			}
			if err := ws.SaveMeta(workspace.Meta{Title: args[0], Description: desc}); err != nil {
				return err
			}
			fmt.Println("updated ahp.toml")
			return nil
		},
	}
	addWorkspaceFlag(c)
	c.Flags().String("description", "", "Description")
	return c
}

func cmdAddCriterion() *cobra.Command {
	c := &cobra.Command{
		Use:   "add-criterion [id] [name]",
		Short: "Create or update a criterion",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			parent, _ := cmd.Flags().GetString("parent")
			desc, _ := cmd.Flags().GetString("description")
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return err
			}
			if err := ws.UpsertCriterion(workspace.Criterion{
				ID: args[0], Name: args[1], ParentID: parent, Description: desc,
			}); err != nil {
				return err
			}
			fmt.Println("criterion", args[0])
			return nil
		},
	}
	addWorkspaceFlag(c)
	c.Flags().String("parent", "", "Parent criterion id")
	c.Flags().String("description", "", "Description")
	return c
}

func cmdAddAlternative() *cobra.Command {
	c := &cobra.Command{
		Use:   "add-alternative [id] [name]",
		Short: "Create or update an alternative",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			desc, _ := cmd.Flags().GetString("description")
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return err
			}
			if err := ws.UpsertAlternative(workspace.Alternative{
				ID: args[0], Name: args[1], Description: desc,
			}); err != nil {
				return err
			}
			fmt.Println("alternative", args[0])
			return nil
		},
	}
	addWorkspaceFlag(c)
	c.Flags().String("description", "", "Description")
	return c
}

func cmdSetAttribute() *cobra.Command {
	c := &cobra.Command{
		Use:   "set-attribute [alternative] [criterion] [value]",
		Short: "Write an objective attribute fact",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			unit, _ := cmd.Flags().GetString("unit")
			source, _ := cmd.Flags().GetString("source")
			note, _ := cmd.Flags().GetString("note")
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return err
			}
			if err := ws.UpsertAttribute(workspace.AttributeRow{
				AlternativeID: args[0], CriterionID: args[1], Value: args[2],
				Unit: unit, Source: source, Note: note,
			}); err != nil {
				return err
			}
			fmt.Printf("attribute %s/%s\n", args[0], args[1])
			return nil
		},
	}
	addWorkspaceFlag(c)
	c.Flags().String("unit", "", "Unit")
	c.Flags().String("source", "", "Source")
	c.Flags().String("note", "", "Note")
	return c
}

func cmdMCP() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run MCP stdio server for agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ahpmcp.Run()
		},
	}
}
