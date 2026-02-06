package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/rapid-downloader/rapid/api"
	"github.com/rapid-downloader/rapid/services/downloader"
	"github.com/rapid-downloader/rapid/services/settings"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/thoriqadillah/gema"
	"go.uber.org/fx"
)

type progressBar struct {
	bar    *progressbar.ProgressBar
	chunks []float64
	mutex  sync.Mutex
}

func newProgressBar(item api.DownloadItem) *progressBar {
	return &progressBar{
		bar: progressbar.NewOptions64(item.Size,
			progressbar.OptionSetWriter(os.Stdout),
			progressbar.OptionShowBytes(true),
			progressbar.OptionFullWidth(),
			progressbar.OptionSetPredictTime(true),
			progressbar.OptionSetElapsedTime(true),
			progressbar.OptionShowElapsedTimeOnFinish(),
			progressbar.OptionOnCompletion(func() {
				fmt.Printf("\n File saved at %s\n", item.Filepath)
			}),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "=",
				SaucerHead:    ">",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		),
		chunks: make([]float64, item.ChunkLen),
	}
}

func (b *progressBar) update(p downloader.Progress) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.chunks[p.Index] = p.Progress
	avg := 0.0
	for _, chunk := range b.chunks {
		avg += chunk
	}

	avg /= float64(len(b.chunks))

	if avg >= 100 {
		b.bar.Finish()
		return
	}

	currentBytes := int64(avg * float64(b.bar.GetMax()) / 100)
	b.bar.Set64(currentBytes)
}

func download(rapid *downloader.DownloaderService, settings api.Setting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <url>",
		Short: "Download a file from a url",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) == 0 {
				return cmd.Help()
			}

			settings.DownloadLocation = cmd.Flag("download-path").Value.String()

			ctx := cmd.Context()
			var item api.DownloadItem
			errCh := make(chan error)

			go func() {
				url := args[0]
				item, err = rapid.Fetch(ctx, url, downloader.WithSetting(settings))
				if err != nil {
					errCh <- err
					return
				}

				bar := newProgressBar(item)
				errCh <- rapid.Download(ctx, item, bar.update)
			}()

			select {
			case <-ctx.Done():
				return rapid.Stop(item.Id)
			case err := <-errCh:
				return err
			}
		},
	}

	cmd.PersistentFlags().String("download-path", settings.DownloadLocation, "The download location")
	return cmd
}

func main() {
	godotenv.Load()

	ctx := context.Background()
	app := fx.New(
		fx.NopLogger,
		fx.Provide(settings.NewService),
		fx.Provide(downloader.NewService),
		gema.CommandModule("Rapid file downloader", download),
	)

	app.Start(ctx)
	app.Stop(ctx)
	<-app.Done()
}
