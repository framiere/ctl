package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/conduktor/ctl/pkg/client"
	"github.com/conduktor/ctl/pkg/resource"
)

// Note: ApiVersion, ApiVersionHeader, DefaultTimeout, ValidateStrategy are defined in constants.go

type ApplyHandlerContext struct {
	FilePaths       []string
	DryRun          bool
	PrintDiff       bool
	RecursiveFolder bool
	MaxParallel     int    // Deprecated: parallelism is now handled server-side
	Strategy        string // "fail-fast" or "continue-on-error"
	NoProgress      bool   // suppress progress output (for CI)
	AssumeYes       bool   // skip safety prompt on large batches
}

type ApplyResult struct {
	Resource     resource.Resource
	UpsertResult client.Result
	Err          error
}

type ApplyHandler struct {
	rootCtx RootContext
}

func NewApplyHandler(rootCtx RootContext) *ApplyHandler {
	return &ApplyHandler{rootCtx: rootCtx}
}

type ResourceDefinition struct {
	OriginalPath string `json:"originalPath"`
	Content      string `json:"content"`
}

type BatchApplyRequest struct {
	Resources []ResourceDefinition `json:"resources"`
	DryRun    bool                 `json:"dryRun"`
	PrintDiff bool                 `json:"printDiff"`
	Strategy  string               `json:"strategy"`
}

type BatchApplyResponse struct {
	Token string `json:"token"`
}

type BatchApplyStatusResponse struct {
	Token              string             `json:"token"`
	Status             string             `json:"status"`
	Results            []BatchApplyResult `json:"results"`
	Error              *string            `json:"error"`
	Outcome            *string            `json:"outcome"`
	TotalResources     int                `json:"totalResources"`
	ProcessedResources int                `json:"processedResources"`
	SuccessCount       int                `json:"successCount"`
	FailureCount       int                `json:"failureCount"`
}

type BatchApplyResult struct {
	ResourceName  string  `json:"resourceName"`
	ResourceKind  string  `json:"resourceKind"`
	Status        string  `json:"status"`
	Diff          *string `json:"diff"`
	Error         *string `json:"error"`
	DiffTruncated bool    `json:"diffTruncated"`
	RetryCount    int     `json:"retryCount"`
}

var ErrCancelled = fmt.Errorf("operation cancelled by user")

func (h *ApplyHandler) Handle(cmdCtx ApplyHandlerContext) ([]ApplyResult, error) {
	resources, err := LoadResourcesFromFiles(cmdCtx.FilePaths, h.rootCtx.Strict, cmdCtx.RecursiveFolder)
	if err != nil {
		return nil, err
	}

	if len(resources) == 0 {
		return []ApplyResult{}, nil
	}

	var defs []ResourceDefinition
	for _, res := range resources {
		defs = append(defs, ResourceDefinition{
			OriginalPath: res.FilePath,
			Content:      string(res.Json),
		})
	}

	if !cmdCtx.AssumeYes && len(defs) > 50 {
		return nil, fmt.Errorf("refusing to apply %d resources without --yes; re-run with --yes to proceed", len(defs))
	}

	httpClient := h.rootCtx.ConsoleAPIClient().Resty()

	strategy := cmdCtx.Strategy
	if strategy == "" {
		strategy = "fail-fast"
	}

	// FIX #5: Validate strategy before sending to server
	if err := ValidateStrategy(strategy); err != nil {
		return nil, err
	}

	req := BatchApplyRequest{
		Resources: defs,
		DryRun:    cmdCtx.DryRun,
		PrintDiff: cmdCtx.PrintDiff,
		Strategy:  strategy,
	}

	var resp BatchApplyResponse
	postCtx, cancelPost := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancelPost()

	r, err := httpClient.R().
		SetContext(postCtx).
		SetHeader(ApiVersionHeader, ApiVersion).
		SetBody(req).
		SetResult(&resp).
		Post("/api/v1/resources/batch-apply")

	if err != nil {
		return nil, err
	}

	if serverVersion := r.Header().Get(ApiVersionHeader); serverVersion != "" && serverVersion != ApiVersion {
		fmt.Fprintf(os.Stderr, "Warning: Server API version (%s) differs from CLI version (%s). Consider upgrading.\n", serverVersion, ApiVersion)
	}

	if r.IsError() {
		if strings.Contains(r.String(), "Unsupported API version") {
			return nil, fmt.Errorf("API version mismatch: %s\nPlease upgrade your CLI to a compatible version.", r.String())
		}
		return nil, fmt.Errorf("server error: %s", r.String())
	}

	token := resp.Token
	if cmdCtx.DryRun {
		fmt.Printf("Batch apply (DRY RUN) started with token: %s\n", token)
	} else {
		fmt.Printf("Batch apply started with token: %s\n", token)
	}
	fmt.Printf("Strategy: %s\n", strategy)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var finalResults []ApplyResult
	lastProcessed := 0
	cancelled := false

	// Use shared constants for polling intervals
	pollInterval := InitialPollInterval

	for {
		if ctx.Err() != nil && !cancelled {
			cancelled = true
			fmt.Println("\nCancelling batch apply...")
			cancelCtx, cancelCancel := context.WithTimeout(context.Background(), CancelTimeout)
			_, _ = httpClient.R().
				SetContext(cancelCtx).
				SetHeader(ApiVersionHeader, ApiVersion).
				Delete("/api/v1/resources/batch-apply/" + token)
			cancelCancel()
		}

		var status BatchApplyStatusResponse
		pollCtx, cancelPoll := context.WithTimeout(ctx, DefaultTimeout)
		r, err := httpClient.R().
			SetContext(pollCtx).
			SetHeader(ApiVersionHeader, ApiVersion).
			SetResult(&status).
			Get("/api/v1/resources/batch-apply/" + token)
		cancelPoll()

		if err != nil {
			if cancelled {
				return finalResults, ErrCancelled
			}
			return nil, err
		}
		if r.IsError() {
			if cancelled {
				return finalResults, ErrCancelled
			}
			return nil, fmt.Errorf("server error polling status: %s", r.String())
		}

		if !cmdCtx.NoProgress {
			// FIX #10: Clear line with ANSI escape before overwriting to prevent flicker
			if status.TotalResources > 0 {
				progress := float64(status.ProcessedResources) / float64(status.TotalResources) * 100
				fmt.Printf("\033[2K\rProgress: %d/%d (%.0f%%) - Status: %s",
					status.ProcessedResources, status.TotalResources, progress, status.Status)

				if status.ProcessedResources > lastProcessed {
					for i := lastProcessed; i < len(status.Results) && i < status.ProcessedResources; i++ {
						res := status.Results[i]
						icon := "+"
						if res.Error != nil {
							icon = "x"
						}
						fmt.Printf("\n  %s %s/%s: %s", icon, res.ResourceKind, res.ResourceName, res.Status)
						if res.RetryCount > 0 {
							fmt.Printf(" (retries: %d)", res.RetryCount)
						}
						if res.Diff != nil && *res.Diff != "" {
							fmt.Printf("\n    Diff:\n%s", indentString(*res.Diff, "      "))
							if res.DiffTruncated {
								fmt.Printf("\n    [diff truncated]")
							}
						}
						if res.Error != nil {
							fmt.Printf(" - %s", *res.Error)
						}
					}
					lastProcessed = status.ProcessedResources
					// Reset poll interval when progress is made
					pollInterval = InitialPollInterval
				}
			} else {
				fmt.Printf("\033[2K\rStatus: %s", status.Status)
			}
		}

		if status.Status == "Completed" || status.Status == "Cancelled" {
			fmt.Println()

			if !cmdCtx.NoProgress {
				if status.Outcome != nil {
					fmt.Printf("\nOutcome: %s\n", *status.Outcome)
				}
				fmt.Printf("Summary: %d succeeded, %d failed out of %d total\n",
					status.SuccessCount, status.FailureCount, status.TotalResources)
			}

			for _, res := range status.Results {
				result := ApplyResult{
					Resource: resource.Resource{Kind: res.ResourceKind, Name: res.ResourceName},
				}
				if res.Error != nil {
					result.Err = fmt.Errorf("%s", *res.Error)
				} else {
					result.UpsertResult = client.Result{UpsertResult: res.Status}
					if res.Diff != nil {
						result.UpsertResult.Diff = *res.Diff
					}
				}
				finalResults = append(finalResults, result)
			}

			if status.Status == "Cancelled" {
				return finalResults, ErrCancelled
			}
			if status.Error != nil {
				return finalResults, fmt.Errorf("batch failed: %s", *status.Error)
			}
			break
		}

		// Exponential backoff with cap
		time.Sleep(pollInterval)
		if pollInterval < MaxPollInterval {
			pollInterval = pollInterval * 2
			if pollInterval > MaxPollInterval {
				pollInterval = MaxPollInterval
			}
		}
	}

	return finalResults, nil
}

func indentString(s string, indent string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

// FIX #2: applyResources is the legacy local-apply method, kept for backwards compatibility with tests
// In production, Handle() uses server-side batch apply instead
func (h *ApplyHandler) applyResources(resources []resource.Resource, applyFunc func(*resource.Resource, bool, bool) (client.Result, error), cmdCtx ApplyHandlerContext) []ApplyResult {
	results := make([]ApplyResult, len(resources))
	maxParallel := cmdCtx.MaxParallel
	if maxParallel <= 0 {
		maxParallel = 1
	}

	sem := make(chan struct{}, maxParallel)
	done := make(chan int, len(resources))

	for i, res := range resources {
		sem <- struct{}{}
		go func(idx int, r resource.Resource) {
			defer func() {
				<-sem
				done <- idx
			}()
			result, err := applyFunc(&r, cmdCtx.DryRun, cmdCtx.PrintDiff)
			results[idx] = ApplyResult{
				Resource:     r,
				UpsertResult: result,
				Err:          err,
			}
		}(i, res)
	}

	for range resources {
		<-done
	}

	return results
}
