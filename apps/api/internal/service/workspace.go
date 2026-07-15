package service

import (
	"github.com/riffpad/riffpad/apps/api/internal/repository"
)

type Services struct {
	Workspace *WorkspaceService
}

func New(repos *repository.Repositories) *Services {
	return &Services{
		Workspace: NewWorkspaceService(repos),
	}
}

type WorkspaceService struct {
	repos *repository.Repositories
}

func NewWorkspaceService(repos *repository.Repositories) *WorkspaceService {
	return &WorkspaceService{repos: repos}
}
