package tfc

import (
	"context"
	"errors"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
)

// ErrRunStateVersionScanLimit is returned by FindStateVersionForRun when the
// scan of the workspace's state versions reaches its page cap without being
// able to definitively determine whether the run has an associated state
// version.
var ErrRunStateVersionScanLimit = errors.New("could not determine the state version for this run within the most recent 2000 state versions")

// maxStateVersionScanPages bounds FindStateVersionForRun's pagination as a
// safety backstop against pathological workspaces with a huge number of
// state versions newer than the run being searched for.
const maxStateVersionScanPages = 20

func (c *Client) ListRuns(ctx context.Context) ([]*tfe.Run, error) {
	ws, err := c.GetWorkspace(ctx)
	if err != nil {
		return nil, err
	}
	list, err := c.tfe.Runs.List(ctx, ws.ID, &tfe.RunListOptions{
		ListOptions: tfe.ListOptions{PageSize: 10},
	})
	if err != nil {
		return nil, fmt.Errorf("listing runs: %w", err)
	}
	return list.Items, nil
}

func (c *Client) GetLatestRun(ctx context.Context) (*tfe.Run, error) {
	ws, err := c.tfe.Workspaces.ReadWithOptions(ctx, c.cfg.Organization, c.cfg.WorkspaceName, &tfe.WorkspaceReadOptions{
		Include: []tfe.WSIncludeOpt{tfe.WSCurrentRun},
	})
	if err != nil {
		return nil, fmt.Errorf("getting workspace: %w", err)
	}
	if ws.CurrentRun == nil {
		return nil, fmt.Errorf("workspace has no current run")
	}
	return ws.CurrentRun, nil
}

func (c *Client) GetRun(ctx context.Context, id string) (*tfe.Run, error) {
	if id == "" || id == "latest" {
		return c.GetLatestRun(ctx)
	}
	run, err := c.tfe.Runs.Read(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting run %s: %w", id, err)
	}
	return run, nil
}

// FindStateVersionForRun locates the state version associated with run by
// paginating through the workspace's state versions newest-first. Runs do
// not have a native relation back to their state version, so this is the
// only way to resolve it.
//
// The newest-first ordering (and that sv.Run.ID is populated on List results
// without an explicit include) is not documented by the HCP Terraform API;
// it was confirmed empirically against a live workspace before relying on it
// for the early-termination logic below.
//
// It returns (nil, nil) when it can definitively determine that the run has
// no associated state version (the normal, expected outcome for plan-only,
// errored, or discarded runs). It returns (nil, ErrRunStateVersionScanLimit)
// when the scan hit its page cap without resolving either way.
func (c *Client) FindStateVersionForRun(ctx context.Context, run *tfe.Run) (*tfe.StateVersion, error) {
	for page := 1; page <= maxStateVersionScanPages; page++ {
		list, err := c.tfe.StateVersions.List(ctx, &tfe.StateVersionListOptions{
			ListOptions:  tfe.ListOptions{PageNumber: page, PageSize: 100},
			Organization: c.cfg.Organization,
			Workspace:    c.cfg.WorkspaceName,
		})
		if err != nil {
			return nil, fmt.Errorf("listing state versions: %w", err)
		}

		for _, sv := range list.Items {
			if sv.Run != nil && sv.Run.ID == run.ID {
				return sv, nil
			}
			if sv.CreatedAt.Before(run.CreatedAt) {
				// This and every subsequent (older) state version predate
				// the run, so it's now definitive that the run has no
				// associated state version.
				return nil, nil
			}
		}

		// Naturally exhausted the workspace's state versions without a
		// match or a cutoff: definitively none.
		if list.Pagination == nil || len(list.Items) == 0 || list.NextPage == 0 || page >= list.TotalPages {
			return nil, nil
		}
	}
	return nil, ErrRunStateVersionScanLimit
}
