package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/conduktor/ctl/internal/cli"
	"github.com/spf13/cobra"
)

func initTemplateServer(rootContext cli.RootContext) {
	var file *string
	var edit *bool
	var apply *bool

	var templateServerCmd = &cobra.Command{
		Use:   "template-server [kind]",
		Short: "Get a yaml example for a given kind from the server",
		Long: `Fetch YAML templates directly from the Conduktor Console server.

If kind is not provided, it will list all available kinds from the server.
This command requires a connection to the server and ensures templates
are always up-to-date with the server's schema.

Examples:
  conduktor template-server                    # List all available kinds
  conduktor template-server Topic              # Get template for Topic
  conduktor template-server Topic -o topic.yaml  # Save to file
`,
		Args: cobra.MaximumNArgs(1),
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
			runTemplateServer(rootContext, args, file, edit, apply)
		},
	}

	rootCmd.AddCommand(templateServerCmd)

	file = templateServerCmd.PersistentFlags().StringP("output", "o", "", "Write example to file")
	edit = templateServerCmd.PersistentFlags().BoolP("edit", "e", false, "Edit the YAML file post-creation; this works only with --output. It will use the EDITOR environment variable or nano if not set.")
	apply = templateServerCmd.PersistentFlags().BoolP("apply", "a", false, "Apply the YAML file post-editing; this works only with --edit.")
}

func runTemplateServer(rootContext cli.RootContext, args []string, file *string, edit *bool, apply *bool) {
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

		fmt.Println("Available Kinds:")
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

	if file == nil || *file == "" {
		fmt.Println("---")
		fmt.Println(example)
	} else {
		_, err := os.Stat(*file)
		if err == nil {
			fmt.Fprintf(os.Stderr, "File %s already exists. You can use conduktor template-server %s >> %s to append to existing file\n", *file, kind, *file)
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
