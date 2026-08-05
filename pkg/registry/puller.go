package registry

import (
	"context"
	"fmt"
	"os"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ttyutil"
	"github.com/ferret-linux/otter/pkg/ui"
)

// Pull pulls the given image ref using the provided container manager.
//
// If force is false and the image is already present locally, the pull is
// skipped.
func Pull(
	ctx context.Context,
	cm containermanager.ContainerManager,
	imageRef string,
	platform string,
	force bool,
	progress *ui.Progress,
) error {
	if !force && cm.ImageExists(ctx, imageRef) {
		return nil
	}

	ui.DefaultLogger.Info("large images may take a while, please be patient...")
	progress.Next("pulling '%s'...", imageRef)

	var out containermanager.PullOutput
	var box *ui.LiveBox
	if ttyutil.IsInteractive() {
		box = ui.NewLiveBox(os.Stderr)
		box.Start()
		out = box
	}

	err := cm.PullImage(ctx, imageRef, platform, out)
	if box != nil {
		box.Close()
	}

	if err != nil {
		progress.Fail()
		return fmt.Errorf("failed to pull image '%s': %w", imageRef, err)
	}

	progress.Done()
	return nil
}
