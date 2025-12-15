package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/conduktor/ctl/internal/cli"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template [kind]",
	Short: "Get a yaml example for a given kind",
	Long: `Get a yaml example for a given kind.

By default, uses embedded templates from the CLI (works offline).

With --from-server flag, fetches templates directly from the Conduktor Console server.
This ensures templates are always up-to-date with the server's schema.

Examples:
  conduktor template                        # List available kinds (embedded)
  conduktor template Topic                  # Get embedded template for Topic
  conduktor template --from-server          # List kinds from server
  conduktor template Topic --from-server    # Get template from server
  conduktor template Topic -o topic.yaml    # Save to file
`,
	Args: cobra.MaximumNArgs(1),
}

func initTemplate(rootContext cli.RootContext) {
	rootCmd.AddCommand(templateCmd)
	var file *string
	var edit *bool
	var apply *bool
	var fromServer *bool

	file = templateCmd.PersistentFlags().StringP("output", "o", "", "Write example to file")
	edit = templateCmd.PersistentFlags().BoolP("edit", "e", false, "Edit the YAML file post-creation; this works only with --output. It will use the EDITOR environment variable or nano if not set.")
	apply = templateCmd.PersistentFlags().BoolP("apply", "a", false, "Apply the YAML file post-editing; this works only with --edit.")
	fromServer = templateCmd.PersistentFlags().Bool("from-server", false, "Fetch template from server instead of using embedded defaults")

	templateCmd.PreRun = func(cmd *cobra.Command, args []string) {
		if edit != nil && *edit && (file == nil || *file == "") {
			fmt.Fprintln(os.Stderr, "Cannot use --edit without --output")
			os.Exit(10)
		}
		if apply != nil && *apply && (edit == nil || !*edit) {
			fmt.Fprintln(os.Stderr, "Cannot use --apply without --edit")
			os.Exit(11)
		}
	}

	templateCmd.Run = func(cmd *cobra.Command, args []string) {
		if *fromServer {
			runTemplateFromServer(rootContext, args, file, edit, apply)
		} else {
			runTemplateEmbedded(rootContext, args, file, edit, apply)
		}
	}

	// Add all kinds as subcommands for backward compatibility
	for name, kind := range rootContext.Catalog.Kind {
		kindName := name // capture for closure
		kindRef := kind  // capture for closure
		kindCmd := &cobra.Command{
			Use:     kindName,
			Short:   "Get a yaml example for resource of kind " + kindName,
			Args:    cobra.NoArgs,
			Long:    `Get a yaml example for resource of kind ` + kindName,
			Aliases: buildAlias(kindName),
			PreRun: func(cmd *cobra.Command, args []string) {
				if edit != nil && *edit && (file == nil || *file == "") {
					fmt.Fprintln(os.Stderr, "Cannot use --edit without --output")
					os.Exit(10)
				}
				if apply != nil && *apply && (edit == nil || !*edit) {
					fmt.Fprintln(os.Stderr, "Cannot use --apply without --edit")
					os.Exit(11)
				}
			},
			Run: func(cmd *cobra.Command, args []string) {
				if *fromServer {
					runTemplateFromServer(rootContext, []string{kindName}, file, edit, apply)
				} else {
					example := kindRef.GetLatestKindVersion().GetApplyExample()
					if example == "" {
						fmt.Fprintf(os.Stderr, "No template for kind %s\n", kindName)
						os.Exit(1)
					}
					writeTemplate(rootContext, kindName, example, file, edit, apply)
				}
			},
		}
		templateCmd.AddCommand(kindCmd)
	}
}

func runTemplateEmbedded(rootContext cli.RootContext, args []string, file *string, edit *bool, apply *bool) {
	// If no kind specified, list all available kinds
	if len(args) == 0 {
		fmt.Println("Available Kinds (use 'template <kind>' or 'template <kind> --from-server'):")
		for name := range rootContext.Catalog.Kind {
			fmt.Println("  " + name)
		}
		return
	}

	// Get template for specific kind
	kindName := args[0]
	kind, ok := rootContext.Catalog.Kind[kindName]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown kind: %s\n", kindName)
		os.Exit(1)
	}

	example := kind.GetLatestKindVersion().GetApplyExample()
	if example == "" {
		fmt.Fprintf(os.Stderr, "No template for kind %s\n", kindName)
		os.Exit(1)
	}

	writeTemplate(rootContext, kindName, example, file, edit, apply)
}

func runTemplateFromServer(rootContext cli.RootContext, args []string, file *string, edit *bool, apply *bool) {
	apiClient := rootContext.ConsoleAPIClient()
	httpClient := apiClient.Resty()
	baseURL := apiClient.BaseURL()
	// baseURL ends with /api, but template is at /public/v1/resources/template
	// So we need to strip /api and use /public/v1/resources/template
	templateURL := strings.TrimSuffix(baseURL, "/api") + "/public/v1/resources/template"

	// If no kind specified, list all available kinds
	if len(args) == 0 {
		var kinds []string
		resp, err := httpClient.R().
			SetHeader(cli.ApiVersionHeader, cli.ApiVersion).
			SetResult(&kinds).
			Get(templateURL)
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

		fmt.Println("Available Kinds (from server):")
		for _, k := range kinds {
			fmt.Println("  " + k)
		}
		return
	}

	// Fetch template for specific kind
	kind := args[0]
	resp, err := httpClient.R().
		SetHeader(cli.ApiVersionHeader, cli.ApiVersion).
		Get(templateURL + "/" + kind)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching template: %s\n", err)
		os.Exit(1)
	}
	if resp.IsError() {
		if strings.Contains(resp.String(), "Unsupported API version") {
			fmt.Fprintf(os.Stderr, "API version mismatch: %s\nPlease upgrade your CLI.\n", resp.String())
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error fetching template for kind '%s': %s\n", kind, resp.String())
		os.Exit(1)
	}

	example := resp.String()
	writeTemplate(rootContext, kind, example, file, edit, apply)
}

func writeTemplate(rootContext cli.RootContext, kindName string, example string, file *string, edit *bool, apply *bool) {
	if file == nil || *file == "" {
		fmt.Println("---")
		fmt.Println(example)
	} else {
		_, err := os.Stat(*file)
		if err == nil {
			fmt.Fprintf(os.Stderr, "File %s already exists. You can use conduktor template %s >> %s to append to existing file\n", *file, kindName, *file)
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
