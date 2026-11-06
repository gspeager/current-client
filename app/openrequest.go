package app

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/gspeager/current-client/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const openRequestedEvent = "repo:open-requested"

func init() {
	application.RegisterEvent[application.Void](openRequestedEvent)
}

// OpenRequestInfo is a repository to open, from a path argument, a
// current-client:// link or the file manager. Confirm means a link names a
// repository the app doesn't know, so it asks first: a repository's own settings
// can run programs as soon as Git reads it, and a link can come from a web page.
// Problem is set instead of a path for a link that can't be used.
type OpenRequestInfo struct {
	Path    string `json:"path"`
	Confirm bool   `json:"confirm"`
	Problem string `json:"problem"`
}

var openRequests = struct {
	sync.Mutex
	queue []OpenRequestInfo
	focus func()
}{}

// parseOpenArg reads one argument. ok is false for one that asks for nothing,
// such as a flag.
func parseOpenArg(arg, workingDir string) (OpenRequestInfo, bool) {
	if arg == "" || strings.HasPrefix(arg, "-") {
		return OpenRequestInfo{}, false
	}
	lower := strings.ToLower(arg)
	switch {
	case strings.HasPrefix(lower, "current-client:"):
		return parseOpenLink(arg), true
	case strings.HasPrefix(lower, "file://"):
		parsed, err := url.Parse(arg)
		if err != nil || parsed.Path == "" {
			return OpenRequestInfo{Problem: "This address isn't a folder on this computer."}, true
		}
		return OpenRequestInfo{Path: filepath.Clean(filepath.FromSlash(parsed.Path))}, true
	}
	if !filepath.IsAbs(arg) && !isNetworkPath(arg) {
		arg = filepath.Join(workingDir, arg)
	}
	return OpenRequestInfo{Path: filepath.Clean(arg)}, true
}

func parseOpenLink(link string) OpenRequestInfo {
	invalid := OpenRequestInfo{Problem: "This link isn't one Current Client can open."}
	parsed, err := url.Parse(link)
	if err != nil || parsed.Host != "open" {
		return invalid
	}
	path := parsed.Query().Get("path")
	if isNetworkPath(path) {
		return OpenRequestInfo{Problem: "A link can't open a network path: " + path}
	}
	if !filepath.IsAbs(path) {
		return invalid
	}
	path = filepath.Clean(path)
	return OpenRequestInfo{Path: path, Confirm: !knownRepository(path)}
}

// Reaching one would contact a server, which a link never does.
func isNetworkPath(path string) bool {
	return strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, "//")
}

func knownRepository(path string) bool {
	cfg := configOrDefault()
	return slices.Contains(cfg.RecentRepositories, path) || slices.Contains(cfg.OpenTabs, path)
}

// QueueOpenArgs takes command-line arguments, without the program's name.
func QueueOpenArgs(args []string, workingDir string) {
	added := false
	openRequests.Lock()
	for _, arg := range args {
		request, ok := parseOpenArg(arg, workingDir)
		if ok && !slices.Contains(openRequests.queue, request) {
			openRequests.queue = append(openRequests.queue, request)
			added = true
		}
	}
	focus := openRequests.focus
	openRequests.Unlock()
	if !added {
		return
	}
	if running := application.Get(); running != nil {
		running.Event.Emit(openRequestedEvent)
	}
	if focus != nil {
		focus()
	}
}

// TakeOpenRequests hands over what's waiting. It's a queue rather than only an
// event because the first request arrives before the frontend is listening.
func (s *RepositoryService) TakeOpenRequests() []OpenRequestInfo {
	openRequests.Lock()
	defer openRequests.Unlock()
	taken := openRequests.queue
	openRequests.queue = nil
	if taken == nil {
		return []OpenRequestInfo{}
	}
	return taken
}

// SingleInstance hands a second launch's arguments to the running app. The ID
// follows the settings folder, so a separate HOME runs a separate instance.
func SingleInstance() *application.SingleInstanceOptions {
	settings, err := config.DefaultPath()
	if err != nil {
		settings = "default"
	}
	sum := sha256.Sum256([]byte(settings))
	return &application.SingleInstanceOptions{
		UniqueID: "io.github.gspeager.current-client." + hex.EncodeToString(sum[:8]),
		OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
			if len(data.Args) > 1 {
				QueueOpenArgs(data.Args[1:], data.WorkingDir)
			}
		},
	}
}

// ListenForOpenRequests covers what the OS delivers to a running app rather than
// as arguments: macOS's links and folders dropped on the app.
func ListenForOpenRequests(current *application.App, window *application.WebviewWindow) {
	openRequests.Lock()
	openRequests.focus = func() {
		window.Show()
		window.Focus()
	}
	openRequests.Unlock()
	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = ""
	}
	current.Event.OnApplicationEvent(events.Common.ApplicationLaunchedWithUrl, func(event *application.ApplicationEvent) {
		QueueOpenArgs([]string{event.Context().URL()}, workingDir)
	})
	current.Event.OnApplicationEvent(events.Common.ApplicationOpenedWithFile, func(event *application.ApplicationEvent) {
		QueueOpenArgs([]string{event.Context().Filename()}, workingDir)
	})
}
