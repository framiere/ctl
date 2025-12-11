package cmd

import (
	"fmt"
	"os"

	"github.com/conduktor/ctl/internal/cli"
	"github.com/spf13/cobra"
)

func initBatchApply(rootContext cli.RootContext) {
	var filePath *[]string
	var recursiveFolder *bool
	var dryRunFlag *bool
	var printDiffFlag *bool
	var strategyFlag *string
	var noProgress *bool
	var assumeYes *bool

	var batchApplyCmd = &cobra.Command{
		Use:   "batch-apply",
		Short: "Apply resources using server-side batch processing",
		Long: `Apply resources using server-side batch processing.

This command sends all resources to the server in a single request,
allowing the server to handle ordering, parallelism, and retries.

The server processes resources asynchronously and the CLI polls for progress.
This is more efficient for large numbers of resources and provides better
reliability through server-side retry logic.

Strategies:
  fail-fast:         Stop on first error (default)
  continue-on-error: Continue processing remaining resources after errors

Exit codes:
  0: All resources applied successfully
  1: All resources failed
  2: Partial success (some succeeded, some failed)
`,
		Run: func(cmd *cobra.Command, args []string) {
			runBatchApply(rootContext, *filePath, *recursiveFolder, *dryRunFlag, *printDiffFlag, *strategyFlag, *noProgress, *assumeYes)
		},
	}

	rootCmd.AddCommand(batchApplyCmd)

	filePath = batchApplyCmd.
		PersistentFlags().StringArrayP("file", "f", make([]string, 0), FILE_ARGS_DOC)

	dryRunFlag = batchApplyCmd.
		PersistentFlags().Bool("dry-run", false, "Test potential changes without the effects being applied")

	printDiffFlag = batchApplyCmd.
		PersistentFlags().Bool("print-diff", false, "Print the diff between the current resource and the one to be applied")

	recursiveFolder = batchApplyCmd.
		PersistentFlags().BoolP("recursive", "r", false, "Apply all .yaml or .yml files in the specified folder and its subfolders.")

	strategyFlag = batchApplyCmd.
		PersistentFlags().String("strategy", "fail-fast", "Apply strategy: fail-fast or continue-on-error")

	noProgress = batchApplyCmd.
		PersistentFlags().Bool("no-progress", false, "Do not display live progress (useful for CI logs)")

	assumeYes = batchApplyCmd.
		PersistentFlags().Bool("yes", false, "Skip confirmation when applying a large number of resources")

	_ = batchApplyCmd.MarkPersistentFlagRequired("file")

	batchApplyCmd.PreRun = func(cmd *cobra.Command, args []string) {
		if *strategyFlag != "fail-fast" && *strategyFlag != "continue-on-error" {
			fmt.Fprintf(os.Stderr, "Error: --strategy must be one of [fail-fast, continue-on-error]\n")
			os.Exit(1)
		}
	}
}

func runBatchApply(rootContext cli.RootContext, filePath []string, recursiveFolder bool, dryRun bool, printDiff bool, strategy string, noProgress bool, assumeYes bool) {
	handler := cli.NewBatchApplyHandler(rootContext)

	cmdCtx := cli.BatchApplyHandlerContext{
		FilePaths:       filePath,
		DryRun:          dryRun,
		PrintDiff:       printDiff,
		RecursiveFolder: recursiveFolder,
		Strategy:        strategy,
		NoProgress:      noProgress,
		AssumeYes:       assumeYes,
	}

	results, err := handler.Handle(cmdCtx)
	if err != nil {
		if err == cli.ErrCancelled {
			fmt.Fprintln(os.Stderr, "Operation cancelled")
			os.Exit(130) // Standard exit code for SIGINT
		}
		fmt.Fprintf(os.Stderr, "Error during batch apply: %s\n", err)
		os.Exit(1)
	}

	successes := 0
	failures := 0
	for _, result := range results {
		if result.Err != nil {
			fmt.Fprintf(os.Stderr, "Could not apply resource %s/%s: %s\n", result.Resource.Kind, result.Resource.Name, result.Err)
			failures++
		} else if result.UpsertResult.UpsertResult != "" {
			fmt.Printf("%s", result.UpsertResult.Diff)
			fmt.Printf("%s/%s: %s\n", result.Resource.Kind, result.Resource.Name, result.UpsertResult.UpsertResult)
			successes++
		}
	}

	if failures > 0 {
		if successes > 0 {
			os.Exit(2) // partial success
		}
		os.Exit(1)
	}
}
