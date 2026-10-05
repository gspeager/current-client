package app

import (
	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func Services() []application.Service {
	return []application.Service{
		application.NewService(&GitService{}),
		application.NewService(&RepositoryService{}),
		application.NewService(&StatusService{}),
		application.NewService(&DiffService{}),
		application.NewService(&CommitService{}),
		application.NewService(&BranchService{}),
		application.NewService(&RemoteService{}),
		application.NewService(&StashService{}),
		application.NewService(&TagService{}),
		application.NewService(&GitFlowService{}),
		application.NewService(&BlameService{}),
		application.NewService(&ContributorsService{}),
		application.NewService(&ConflictService{}),
		application.NewService(&ReflogService{}),
		application.NewService(&BisectService{}),
		application.NewService(&PatchService{}),
		application.NewService(&ActivityService{}),
		application.NewService(&DashboardService{}),
		application.NewService(&WatchService{}),
		application.NewService(&HistoryService{}),
		application.NewService(&IdentityService{}),
		application.NewService(&SettingsService{}),
		application.NewService(&PlatformService{}),
		application.NewService(&CompareService{}),
		application.NewService(&ChangelogService{}),
		application.NewService(&UndoService{}),
		application.NewService(&OverlapService{}),
		application.NewService(&SearchService{}),
		application.NewService(&WorktreeService{}),
	}
}

// Startup applies saved settings before any service runs and returns the saved
// window size, or zeros when there is none.
func Startup() (width, height int) {
	cfg, _, err := loadCurrentConfig()
	if err != nil {
		return 0, 0
	}
	gitexec.SetDefaultBinary(cfg.GitExecutablePath)
	return cfg.WindowWidth, cfg.WindowHeight
}

// SetSettingsFolder keeps an embedding app's settings (recent repositories,
// tabs, window size) apart from Current Client's. Call it before Startup.
func SetSettingsFolder(name string) {
	config.SetDirName(name)
}

// TitleBarTheme is the theme to create the window with: the saved theme, and
// never SystemDefault, whose own updates would undo SetDarkTitleBar.
func TitleBarTheme() application.Theme {
	switch configOrDefault().Theme {
	case "light":
		return application.Light
	case "system":
		if !systemPrefersDark() {
			return application.Light
		}
	}
	return application.Dark
}
