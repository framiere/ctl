package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/conduktor/ctl/internal/cli"
	"github.com/spf13/cobra"
)

// FIX #8: Use shared constants from cli package instead of duplicating

var templateCmd = &cobra.Command{
	Use:   "template [kind]",
	Short: "Get a yaml example for a given kind",
	Args:  cobra.MaximumNArgs(1),
	Long:  `If kind is not provided it will list all available kinds`,
}

func initTemplate(rootContext cli.RootContext) {
	rootCmd.AddCommand(templateCmd)
	var file *string
	var edit *bool
	var apply *bool
	file = templateCmd.PersistentFlags().StringP("output", "o", "", "Write example to file")
	edit = templateCmd.PersistentFlags().BoolP("edit", "e", false, "Edit the YAML file post-creation; this works only with --output. It will use the EDITOR environment variable or nano if not set.")
	apply = templateCmd.PersistentFlags().BoolP("apply", "a", false, "Apply the YAML file post-editing; this works only with --edit.")

	templateCmd.Run = func(cmd *cobra.Command, args []string) {
		if edit != nil && *edit && (file == nil || *file == "") {
			fmt.Fprintln(os.Stderr, "Cannot use --edit without --output")
			os.Exit(10)
		}
		if apply != nil && *apply && (edit == nil || !*edit) {
			fmt.Fprintln(os.Stderr, "Cannot use --apply without --edit")
			os.Exit(11)
		}

		httpClient := rootContext.ConsoleAPIClient().Resty()

		if len(args) == 0 {
			var kinds []string
			resp, err := httpClient.R().
				SetHeader(cli.ApiVersionHeader, cli.ApiVersion).
				SetResult(&kinds).
				Get("/api/v1/resources/template/kinds")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching kinds: %s\n", err)
				os.Exit(1)
			}
			if resp.IsError() {
				if strings.Contains(resp.String(), "Unsupported API version") {
					fmt.Fprintf(os.Stderr, "API version mismatch: %s\nPlease upgrade your CLI.\n", resp.String())
					os.Exit(1)
				}
				fmt.Fprintf(os.Stderr, "Error fetching kinds: %s\n", resp.String())
				os.Exit(1)
			}

			fmt.Println("Available Commands (Kinds):")
			for _, k := range kinds {
				fmt.Println("  " + k)
			}
			return
		}

		kind := args[0]
		resp, err := httpClient.R().
			SetHeader(cli.ApiVersionHeader, cli.ApiVersion).
			Get("/api/v1/resources/template/" + kind)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching template: %s\n", err)
			os.Exit(1)
		}
		if resp.IsError() {
			if strings.Contains(resp.String(), "Unsupported API version") {
				fmt.Fprintf(os.Stderr, "API version mismatch: %s\nPlease upgrade your CLI.\n", resp.String())
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Error fetching template: %s\n", resp.String())
			os.Exit(1)
		}

		example := resp.String()

		if file == nil || *file == "" {
			fmt.Println("---")
			fmt.Println(example)
		} else {
			_, err := os.Stat(*file)
			if err == nil {
				fmt.Fprintf(os.Stderr, "File %s already exists. You can use conduktor template %s >> %s to append to existing file\n", *file, kind, *file)
				os.Exit(2)
			}
			f, err := os.Create(*file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating file %s: %s\n", *file, err)
				os.Exit(3)
			}
			defer f.Close()
			w := bufio.NewWriter(f)
			if apply != nil && *apply {
				_, err = w.WriteString(AutoApplyWarningMessage)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error writing to file %s: %s\n", *file, err)
					os.Exit(4)
				}
			}
			_, err = w.WriteString("---\n")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to file %s: %s\n", *file, err)
				os.Exit(4)
			}
			_, err = w.WriteString(example)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to file %s: %s\n", *file, err)
				os.Exit(4)
			}
			err = w.Flush()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to file %s: %s\n", *file, err)
				os.Exit(5)
			}
			editAndApply(rootContext, edit, file, apply)
		}
	}
}

func editAndApply(rootContext cli.RootContext, edit *bool, file *string, apply *bool) {
	if edit != nil && *edit {
		// Run editor on the file
		err := runEditor(*file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Editor error: %s\n", err)
			os.Exit(7)
		}

		recursiveFolder := false
		if apply != nil && *apply {
			runApply(rootContext, []string{*file}, recursiveFolder)
		}
	}
}
