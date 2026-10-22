//go:build !production

package app

// Wails builds releases with the production tag; everything else is a dev run.
const devBuild = true
