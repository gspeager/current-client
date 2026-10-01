package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/identity"
	"github.com/gspeager/current-client/internal/config"
)

type IdentityService struct{}

type IdentityRef struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type IdentityInfo struct {
	Initials string `json:"initials"`
	Color    string `json:"color"`
}

func (s *IdentityService) GetIdentities(refs []IdentityRef) []IdentityInfo {
	cfg := configOrDefault()
	return mapSlice(refs, func(r IdentityRef) IdentityInfo {
		return IdentityInfo{
			Initials: identity.Initials(r.Name, r.Email),
			Color:    identity.ResolveColor(cfg.IdentityColors, r.Name, r.Email),
		}
	})
}

type CurrentUserInfo struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Initials string `json:"initials"`
	Color    string `json:"color"`
}

func (s *IdentityService) GetCurrentUser(repoPath string) CurrentUserInfo {
	user := git.CurrentUser(context.Background(), repoPath)
	return CurrentUserInfo{
		Name:     user.Name,
		Email:    user.Email,
		Initials: identity.Initials(user.Name, user.Email),
		Color:    identity.ResolveColor(configOrDefault().IdentityColors, user.Name, user.Email),
	}
}

func (s *IdentityService) SetIdentityColor(name, email, color string) error {
	cfg, path, err := loadCurrentConfig()
	if err != nil {
		return err
	}
	cfg.IdentityColors, err = identity.SetColor(cfg.IdentityColors, name, email, color)
	if err != nil {
		return err
	}
	return config.Save(path, cfg)
}

type GlobalUserInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (s *IdentityService) GetGlobalUser() GlobalUserInfo {
	return GlobalUserInfo(git.GlobalUser(context.Background()))
}

func (s *IdentityService) SetGlobalUser(name, email string) error {
	return git.SetGlobalUser(context.Background(), name, email)
}

func (s *IdentityService) SetRepoUser(repoPath, name, email string) error {
	return git.SetRepoUser(context.Background(), repoPath, name, email)
}
