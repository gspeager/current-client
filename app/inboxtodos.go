//go:build !nohost

package app

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
)

// The host's to-do list is account-wide, so it is cached per host (an empty
// repository path) by project, and only the projects open in the app are
// kept: nothing about another repository stays in memory.
const (
	cacheTodos       = "todos"
	cacheTodosNotice = "todosNotice"
)

// refreshTodos asks for the host's to-do list once per refresh and keeps the
// to-dos about the open repositories. A host that won't list them for this
// token leaves a notice for the Inbox instead.
func refreshTodos(ctx context.Context, provider hostapi.Provider, info HostInfo) {
	list, ok := provider.(hostapi.TodoList)
	if !ok {
		return
	}
	hostClient.Cache.ClearKind(info.Hostname, "", cacheTodos)
	hostClient.Cache.ClearKind(info.Hostname, "", cacheTodosNotice)
	todos, err := list.Todos(ctx)
	if err != nil {
		hostClient.Cache.Put(info.Hostname, "", cacheTodosNotice, "", err.Error())
		return
	}
	open := openProjects(info)
	byProject := map[string][]hostapi.Todo{}
	for _, todo := range todos {
		if project, ok := open[strings.ToLower(todo.Project)]; ok {
			byProject[project] = append(byProject[project], todo)
		}
	}
	for _, project := range open {
		hostClient.Cache.Put(info.Hostname, "", cacheTodos, project, byProject[project])
	}
}

// openProjects is the projects on info's host among the open repositories,
// by lower-cased path, and the refreshed repository's own.
func openProjects(info HostInfo) map[string]string {
	projects := map[string]string{strings.ToLower(info.Project): info.Project}
	for _, path := range configOrDefault().OpenTabs {
		if _, other, ok, err := primaryHost(path); err == nil && ok && other.Hostname == info.Hostname {
			projects[strings.ToLower(other.Project)] = other.Project
		}
	}
	return projects
}

// todoRows adds the project's to-dos to the rows: one that says what a row
// already says, a review request or an assignment, joins that row so it can
// be marked done from it; the rest are rows of their own.
func todoRows(rows []InboxRow, info HostInfo, prefix string) ([]InboxRow, string) {
	notice, _, _ := hostClient.Cache.Get(info.Hostname, "", cacheTodosNotice, "")
	text, _ := notice.(string)
	cached, at, ok := hostClient.Cache.Get(info.Hostname, "", cacheTodos, info.Project)
	if !ok {
		return rows, text
	}
	fetched := at
	for _, todo := range cached.([]hostapi.Todo) {
		if i := rowSaying(rows, todo); i >= 0 {
			rows[i].TodoID = todo.ID
			// A mention on an issue you're assigned is that issue's row too,
			// with why the host listed it beside what the row already says.
			if rows[i].Kind == "assigned" && todo.Reason != hostapi.TodoAssigned {
				rows[i].Detail = strings.TrimPrefix(rows[i].Detail+", "+todo.Reason, ", ")
			}
			continue
		}
		row := InboxRow{ID: "todo:" + todo.ID, Kind: "todo", TodoID: todo.ID, Title: todo.Title, Detail: todo.Reason, WebURL: todo.WebURL, FetchedAt: &fetched}
		switch todo.Target {
		case "change":
			row.Number, row.Prefix = todo.Number, prefix
		case "issue":
			row.Key = strconv.Itoa(todo.Number)
		}
		rows = append(rows, row)
	}
	return rows, text
}

func rowSaying(rows []InboxRow, todo hostapi.Todo) int {
	for i, row := range rows {
		switch {
		case todo.Reason == hostapi.TodoReviewRequested && todo.Target == "change" && row.Kind == "review" && row.Number == todo.Number:
			return i
		case todo.Target == "issue" && row.Kind == "assigned" && row.Key == strconv.Itoa(todo.Number):
			return i
		}
	}
	return -1
}

// markTodoDone tells the host, then drops the to-do from the cache so the
// row goes without a refresh.
func markTodoDone(repoPath, todoID string) error {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil {
		return err
	}
	if !ok || !info.Enabled {
		return errors.New("integration is off for this repository's host")
	}
	list, ok := providerFor(info).(hostapi.TodoList)
	if !ok {
		return errors.New("this host keeps no to-do list")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := list.MarkTodoDone(ctx, todoID); err != nil {
		return err
	}
	if cached, _, ok := hostClient.Cache.Get(info.Hostname, "", cacheTodos, info.Project); ok {
		var kept []hostapi.Todo
		for _, todo := range cached.([]hostapi.Todo) {
			if todo.ID != todoID {
				kept = append(kept, todo)
			}
		}
		hostClient.Cache.Put(info.Hostname, "", cacheTodos, info.Project, kept)
	}
	return nil
}
