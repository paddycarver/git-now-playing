package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

type Player interface {
	GetTrackInfo(context.Context) ([]TrackInfo, error)
}

func main() {
	ctx, signalStop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	if len(os.Args) < 2 {
		log.Println("usage: git-now-playing /path/to/config")
	}
	config, err := parseConfig(os.Args[1])
	if err != nil {
		log.Println("error parsing config:", err)
		os.Exit(1)
	}
	vaultClient, err := vault.NewClient(&vault.Config{
		Address: config.Vault.Address,
	})
	if err != nil {
		log.Println("Error creating vault client:", err)
		os.Exit(1)
	}
	vaultToken := os.Getenv("VAULT_TOKEN")
	if vaultToken == "" {
		log.Println("VAULT_TOKEN must be set")
		os.Exit(1)
	}
	vaultClient.SetToken(vaultToken)

	var players []Player

	if config.Spotify != nil {
		log.Println("spotify configured")
		path := config.Spotify.VaultPath
		if path == "" {
			path = "spotify"
		}
		key := config.Vault.getMountPath() + "/data/" + path
		spotifyInfo, err := vaultClient.Logical().Read(key)
		if err != nil {
			log.Printf("Error retrieving Spotify credentials from Vault %q:", key, err)
			return
		}
		spotifyClientID, err := getVaultString(spotifyInfo.Data, "client_id", key)
		if err != nil {
			log.Println(err)
			return
		}
		spotifyClientSecret, err := getVaultString(spotifyInfo.Data, "client_secret", key)
		if err != nil {
			log.Println(err)
			return
		}
		spotifyRedirectURL, err := getVaultString(spotifyInfo.Data, "redirect_url", key)
		if err != nil {
			log.Println(err)
			return
		}
		spotifyTokenString, err := getVaultString(spotifyInfo.Data, "token", key)
		if err != nil && !errors.Is(err, errVaultKeyNotFound{}) {
			log.Println(err)
			return
		}
		auth := spotifyauth.New(
			spotifyauth.WithClientID(spotifyClientID),
			spotifyauth.WithClientSecret(spotifyClientSecret),
			spotifyauth.WithRedirectURL(spotifyRedirectURL),
			spotifyauth.WithScopes(
				spotifyauth.ScopeUserReadCurrentlyPlaying,
				spotifyauth.ScopeUserReadPlaybackState,
			),
		)
		var spotifyToken *oauth2.Token
		if spotifyTokenString == "" {
			serverAddr := "0.0.0.0:8765"
			if config.Spotify.AuthCallbackAddr != "" {
				serverAddr = config.Spotify.AuthCallbackAddr
			}
			resp := doSpotifyAuth(ctx, auth, serverAddr)
			select {
			case <-ctx.Done():
				log.Println(ctx.Err())
				signalStop()
				return
			case token := <-resp:
				spotifyToken = token
				spotifyTokenBytes, err := json.Marshal(spotifyToken)
				if err != nil {
					log.Println("Couldn't encode spotify token as JSON:", err)
					return
				}
				_, err = vaultClient.Logical().JSONMergePatch(ctx, key, map[string]interface{}{
					"options": map[string]interface{}{
						"cas": 1,
					},
					"data": map[string]interface{}{
						"token": string(spotifyTokenBytes),
					},
				})
				if err != nil {
					log.Println("Couldn't write spotify token to vault:", err)
				}
			}
		} else {
			var st oauth2.Token
			err := json.Unmarshal([]byte(spotifyTokenString), &st)
			if err != nil {
				log.Println("Couldn't parse spotify token from vault:", err)
				return
			}
			spotifyToken = &st
		}
		spotifyClient := spotify.New(auth.Client(ctx, spotifyToken))
		players = append(players, SpotifyPlayer{
			client: spotifyClient,
		})
	}

	for _, plex := range config.Plex {
		log.Println("plex configured")
		path := plex.VaultPath
		if path == "" {
			path = "plex"
		}
		key := config.Vault.getMountPath() + "/data/" + path
		plexInfo, err := vaultClient.Logical().Read(key)
		if err != nil {
			log.Printf("Error retrieving Plex credentials from Vault for %q:", key, err)
			return
		}
		plexToken, err := getVaultString(plexInfo.Data, "token", key)
		players = append(players, PlexPlayer{
			server: plex.Server,
			users:  plex.Users,
			token:  plexToken,
		})
	}

	var commitFile string
	if config.Output == nil || config.Output.Path == "" {
		commitFile, err = os.UserHomeDir()
		if err != nil {
			log.Println("error getting home directory:", err)
			return
		}
		commitFile = filepath.Join(commitFile, ".config", "gitmessage")
	} else {
		commitFile = config.Output.Path
	}
	log.Println("writing to", commitFile)

	runLoop(ctx, players, commitFile)
}

func runLoop(ctx context.Context, players []Player, commitFile string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	var lastSong string
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		var results []TrackInfo

		for _, player := range players {
			tracks, err := player.GetTrackInfo(ctx)
			if err != nil {
				log.Println(err)
				continue
			}
			results = append(results, tracks...)
		}

		output := formatResults(results)

		if lastSong == output {
			continue
		}
		err := os.WriteFile(commitFile, []byte(output), 0600)
		if err != nil {
			log.Println("error writing now playing info to", commitFile+":", err)
			continue
		}
		lastSong = output
	}
}

func formatResults(in []TrackInfo) string {
	if len(in) < 1 {
		return ""
	}
	var res string
	for _, track := range in {
		res += "\n\n"
		res += track.String()
	}
	return res
}
