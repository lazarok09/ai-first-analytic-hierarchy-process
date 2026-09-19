package main

import (
	"fmt"

	"github.com/lazarok09/ahp-method/internal/cliout"
	"github.com/lazarok09/ahp-method/internal/workspace"
	"github.com/spf13/cobra"
)

func cmdQuote() *cobra.Command {
	c := &cobra.Command{
		Use:   "quote",
		Short: "Multi-source quotes and pick-best into attributes",
	}
	c.AddCommand(cmdQuoteAdd(), cmdQuoteList(), cmdQuotePick())
	return c
}

func cmdQuoteAdd() *cobra.Command {
	c := &cobra.Command{
		Use:   "add [alternative] [criterion] [value]",
		Short: "Add/replace a quote (alt×criterion×source)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			unit, _ := cmd.Flags().GetString("unit")
			source, _ := cmd.Flags().GetString("source")
			note, _ := cmd.Flags().GetString("note")
			url, _ := cmd.Flags().GetString("url")
			p := cliout.FromCmd(cmd)
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			item := workspace.QuoteRow{
				AlternativeID: args[0], CriterionID: args[1], Value: args[2],
				Unit: unit, Source: source, Note: note, URL: url,
			}
			if err := ws.UpsertQuote(item); err != nil {
				return cliout.Wrap(cliout.ExitUsage, err)
			}
			if p.JSON {
				return p.PrintJSON(item)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "quote %s/%s @ %s = %s\n", args[0], args[1], source, args[2])
			return nil
		},
	}
	addWorkspaceFlag(c)
	c.Flags().String("unit", "", "Unit (e.g. BRL)")
	c.Flags().String("source", "", "Named local source (required)")
	c.Flags().String("note", "", "Note")
	c.Flags().String("url", "", "Optional product/page URL")
	_ = c.MarkFlagRequired("source")
	return c
}

func cmdQuoteList() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List quotes.csv",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := cliout.FromCmd(cmd)
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			items, err := ws.Quotes()
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			if p.JSON {
				return p.PrintJSON(items)
			}
			if len(items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no quotes")
				return nil
			}
			for _, q := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%s/%s  %s %s  %s\n", q.AlternativeID, q.CriterionID, q.Value, q.Unit, q.Source)
			}
			return nil
		},
	}
	addWorkspaceFlag(c)
	return c
}

func cmdQuotePick() *cobra.Command {
	c := &cobra.Command{
		Use:   "pick [criterion]",
		Short: "Pick best quote per alt into attributes (prefer lower|higher)",
		Long: `For each alternative with quotes on the criterion, write attributes.csv
from the best source (min if --prefer lower, max if higher). Discarded
quotes are recorded in the attribute note for audit.`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			prefer, _ := cmd.Flags().GetString("prefer")
			p := cliout.FromCmd(cmd)
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			res, err := ws.PickQuote(args[0], prefer)
			if err != nil {
				return cliout.Wrap(cliout.ExitUsage, err)
			}
			if p.JSON {
				return p.PrintJSON(res)
			}
			fmt.Fprintln(cmd.OutOrStdout(), res.Summary)
			for _, a := range res.Picked {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s = %s %s (%s)\n", a.AlternativeID, a.Value, a.Unit, a.Source)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Next:  ahp rate --criterion", args[0], "--prefer", prefer, "[--stretch]")
			return nil
		},
	}
	addWorkspaceFlag(c)
	c.Flags().String("prefer", "lower", "lower|higher")
	return c
}

func cmdShare() *cobra.Command {
	c := &cobra.Command{
		Use:   "share",
		Short: "Pasteable ranking summary (WhatsApp-friendly text)",
		Long: `Print top-N ranking with attribute highlights and URLs found in
attribute/quote source or note. Use --whatsapp for the text block only.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			top, _ := cmd.Flags().GetInt("top")
			wa, _ := cmd.Flags().GetBool("whatsapp")
			inc, _ := cmd.Flags().GetBool("include-proposals")
			p := cliout.FromCmd(cmd)
			ws, err := openWS(wsFlag(cmd))
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			sum, err := ws.Share(inc, top)
			if err != nil {
				return cliout.Wrap(cliout.ExitIO, err)
			}
			if p.JSON && !wa {
				return p.PrintJSON(sum)
			}
			fmt.Fprintln(cmd.OutOrStdout(), sum.WhatsAppText)
			return nil
		},
	}
	addWorkspaceFlag(c)
	c.Flags().Int("top", 5, "How many alternatives to include")
	c.Flags().Bool("whatsapp", false, "Print WhatsApp text only (default text mode)")
	c.Flags().Bool("include-proposals", false, "Include proposal judgments in compute")
	return c
}
