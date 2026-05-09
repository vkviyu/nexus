package main

import (
	"embed"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application menu (for macOS native menu bar)
	appMenu := createApplicationMenu(app)

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Nexus",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 30, G: 30, B: 30, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Menu:             appMenu,
		Bind: []any{
			app,
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: false,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
			},
			About: &mac.AboutInfo{
				Title:   "Nexus",
				Message: "API Testing & Mock Server Tool",
			},
			Preferences: &mac.Preferences{
				TabFocusesLinks:        mac.Disabled,
				TextInteractionEnabled: mac.Enabled,
				FullscreenEnabled:      mac.Enabled,
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

// createApplicationMenu creates the native application menu
func createApplicationMenu(app *App) *menu.Menu {
	appMenu := menu.NewMenu()

	// macOS App Menu (required for Cmd+Q etc)
	if runtime.GOOS == "darwin" {
		appMenu.Append(menu.AppMenu())
	}

	// File Menu
	fileMenu := appMenu.AddSubmenu("File")
	fileMenu.AddText("New Workspace", keys.CmdOrCtrl("n"), func(_ *menu.CallbackData) {
		// TODO: Implement new workspace
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Import...", keys.CmdOrCtrl("i"), func(_ *menu.CallbackData) {
		// TODO: Implement import
	})
	fileMenu.AddText("Export...", keys.CmdOrCtrl("e"), func(_ *menu.CallbackData) {
		// TODO: Implement export
	})

	// Edit Menu - 使用 Wails 内置的标准编辑菜单，支持剪贴板操作
	appMenu.Append(menu.EditMenu())

	// View Menu - 使用 Checkbox 菜单项显示当前状态
	viewMenu := appMenu.AddSubmenu("View")

	// Server Domains - 默认显示
	serverDomainsItem := viewMenu.AddCheckbox("Server Domains", true, keys.CmdOrCtrl("1"), func(cd *menu.CallbackData) {
		// Checkbox already toggled by native menu, emit the new state
		app.EmitViewSet("serverDomains", cd.MenuItem.Checked)
	})
	app.viewMenuItems.serverDomains = serverDomainsItem

	// Client Domains - 默认显示
	clientDomainsItem := viewMenu.AddCheckbox("Client Domains", true, keys.CmdOrCtrl("2"), func(cd *menu.CallbackData) {
		app.EmitViewSet("clientDomains", cd.MenuItem.Checked)
	})
	app.viewMenuItems.clientDomains = clientDomainsItem

	// Contract Editor - 默认隐藏
	contractEditorItem := viewMenu.AddCheckbox("Contract Editor", false, keys.CmdOrCtrl("3"), func(cd *menu.CallbackData) {
		app.EmitViewSet("contractEditor", cd.MenuItem.Checked)
	})
	app.viewMenuItems.contractEditor = contractEditorItem

	viewMenu.AddSeparator()
	viewMenu.AddText("Collapse All Panels", nil, func(cd *menu.CallbackData) {
		app.EmitViewAction("collapseAll")
	})
	viewMenu.AddText("Expand All Panels", nil, func(cd *menu.CallbackData) {
		app.EmitViewAction("expandAll")
	})

	// Help Menu
	helpMenu := appMenu.AddSubmenu("Help")
	helpMenu.AddText("Documentation", nil, func(_ *menu.CallbackData) {
		// TODO: Open documentation
	})
	helpMenu.AddText("About Nexus", nil, func(_ *menu.CallbackData) {
		// TODO: Show about dialog
	})

	return appMenu
}
