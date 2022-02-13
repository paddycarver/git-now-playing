package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/segmentio/ksuid"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

type SpotifyPlayer struct {
	client *spotify.Client
}

func (s SpotifyPlayer) GetTrackInfo(ctx context.Context) ([]TrackInfo, error) {
	track, err := getSpotifyNowPlaying(ctx, s.client)
	if err != nil {
		return nil, err
	}
	if track == nil {
		return nil, nil
	}
	return []TrackInfo{
		trackInfoFromSpotify(track),
	}, nil
}

func getSpotifyNowPlaying(ctx context.Context, client *spotify.Client) (*spotify.FullTrack, error) {
	cp, err := client.PlayerCurrentlyPlaying(ctx)
	if err != nil {
		return nil, err
	}
	if !cp.Playing {
		return nil, nil
	}
	return cp.Item, nil
}

func doSpotifyAuth(ctx context.Context, auth *spotifyauth.Authenticator) chan *oauth2.Token {
	state := ksuid.New().String()
	url := auth.AuthURL(state)
	resp := make(chan *oauth2.Token)
	serverCtx, serverCancel := context.WithCancel(context.Background())
	srv := &http.Server{
		Addr: ":8765",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := auth.Token(ctx, state, r)
			if err != nil {
				log.Println(err)
				http.Error(w, "Couldn't get token", http.StatusInternalServerError)
				return
			}
			w.Write([]byte("got token, you can close this window"))
			resp <- token
			serverCancel()
		}),
	}
	go runSpotifyServer(serverCtx, srv)
	go func(ctx context.Context, cancel func()) {
		<-ctx.Done()
		cancel()
	}(ctx, serverCancel)
	fmt.Println("Authorize git-now-playing at", url)
	return resp
}

func runSpotifyServer(ctx context.Context, srv *http.Server) {
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Println(err)
	}
}

func trackInfoFromSpotify(spotifyNowPlaying *spotify.FullTrack) TrackInfo {
	res := TrackInfo{
		Name:      spotifyNowPlaying.SimpleTrack.Name,
		Album:     spotifyNowPlaying.Album.Name,
		SpotifyID: spotifyNowPlaying.SimpleTrack.ID.String(),
	}
	for _, artist := range spotifyNowPlaying.SimpleTrack.Artists {
		res.Artists = append(res.Artists, artist.Name)
	}
	if v, ok := spotifyNowPlaying.ExternalIDs["isrc"]; ok {
		res.ISRC = v
	}
	return res
}
