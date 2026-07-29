package ui

import "github.com/kurabuchi-kentaro/git-snag/internal/domain"

// RepoFoundMsg streams one discovered-and-enriched repository into the UI.
type RepoFoundMsg struct {
	Repo domain.Repo
}

// ScanDoneMsg signals that the recursive scan finished.
type ScanDoneMsg struct{}
