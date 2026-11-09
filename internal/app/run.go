package app

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// Run creates the app and opens the window. It blocks until the window closes.
func Run(assets embed.FS) error {
	a, err := New()
	if err != nil {
		return err
	}
	return wails.Run(&options.App{
		Title:            "HandSSHake",
		Width:            1180,
		Height:           760,
		MinWidth:         900,
		MinHeight:        560,
		BackgroundColour: &options.RGBA{R: 15, G: 17, B: 21, A: 1},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        a.startup,
		OnShutdown:       a.shutdown,
		Bind:             []interface{}{a},
	})
}