package main

import (
	"strings"

	"github.com/hashicorp/hcl/v2/hclsimple"
)

type Config struct {
	Vault   VaultConfig    `hcl:"vault,block"`
	Spotify *SpotifyConfig `hcl:"spotify,block"`
	Plex    []PlexConfig   `hcl:"plex,block"`
	Output  *OutputConfig  `hcl:"output,block"`
}

type VaultConfig struct {
	Address   string `hcl:"address,attr"`
	MountPath string `hcl:"mount_path,attr"`
}

func (v VaultConfig) getMountPath() string {
	return strings.TrimPrefix(strings.TrimSuffix(v.MountPath, "/"), "/")
}

type SpotifyConfig struct {
	VaultPath string `hcl:"vault_path,optional"`
}

type PlexConfig struct {
	Server    string   `hcl:"server,attr"`
	VaultPath string   `hcl:"vault_path,optional"`
	Users     []string `hcl:"users,optional"`
}

type OutputConfig struct {
	Path string `hcl:"path,attr"`
}

func parseConfig(file string) (Config, error) {
	var config Config
	err := hclsimple.DecodeFile(file, nil, &config)
	return config, err
}
