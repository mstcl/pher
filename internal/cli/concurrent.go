package cli

import (
	"context"
	"log/slog"

	"github.com/mstcl/pher/v3/internal/feed"
	"github.com/mstcl/pher/v3/internal/render"
	"github.com/mstcl/pher/v3/internal/state"
	"golang.org/x/sync/errgroup"
)

// runConcurrentJobs executes three jobs of the rest of the program concurrently
// as they are independent of each other:
//  1. Create the atom feed
//  2. Copy assets to the output directory
//  3. Copy static files to the output directory
//  4. Render all source files to HTML to the output directory
func runConcurrentJobs(ctx context.Context, s *state.State, logger *slog.Logger) error {
	// construct and render atom feeds
	constructFeedGroup, _ := errgroup.WithContext(ctx)
	constructFeedGroup.Go(func() error {
		atom, err := feed.Construct(s, logger)
		if err != nil {
			return err
		}

		return feed.Write(s, atom)
	},
	)

	logger.Info("created atom feed")

	// copy asset dirs/files over to output directory
	copyUserAssetsGroup, _ := errgroup.WithContext(ctx)
	copyUserAssetsGroup.Go(func() error {
		if err := copyUserAssets(ctx, s, logger); err != nil {
			return err
		}

		return nil
	},
	)

	logger.Info("synced user assets")

	// copy static content to the output directory
	copyStaticGroup, _ := errgroup.WithContext(ctx)
	copyStaticGroup.Go(func() error {
		if err := copyStatic(s, logger); err != nil {
			return err
		}

		return nil
	},
	)
	logger.Info("copied static files")

	// render all markdown files
	renderGroup, _ := errgroup.WithContext(ctx)
	renderGroup.Go(func() error {
		return render.Render(ctx, s, logger)
	})
	logger.Info("templated all source files")

	// wait for all goroutines to finish
	if err := constructFeedGroup.Wait(); err != nil {
		return err
	}

	if err := copyUserAssetsGroup.Wait(); err != nil {
		return err
	}

	if err := copyStaticGroup.Wait(); err != nil {
		return err
	}

	if err := renderGroup.Wait(); err != nil {
		return err
	}

	return nil
}
