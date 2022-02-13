package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type PlexPlayer struct {
	server string
	token  string
	users  []string
}

func (p PlexPlayer) GetTrackInfo(ctx context.Context) ([]TrackInfo, error) {
	pmc, err := getPlexNowPlaying(ctx, p.server, p.token)
	if err != nil {
		return nil, err
	}
	return trackInfoFromPlex(pmc, p.users), nil
}

type PlexMediaContainer struct {
	Tracks []PlexTrack `xml:"Track"`
}

type PlexTrack struct {
	Title            string          `xml:"title,attr"`
	GrandparentTitle string          `xml:"grandparentTitle,attr"`
	ParentTitle      string          `xml:"parentTitle,attr"`
	OriginalTitle    string          `xml:"originalTitle,attr"`
	Type             string          `xml:"type,attr"`
	Player           PlexTrackPlayer `xml:"Player"`
	User             PlexTrackUser   `xml:"User"`
}

type PlexTrackPlayer struct {
	State string `xml:"state,attr"`
}

type PlexTrackUser struct {
	Name string `xml:"title,attr"`
}

func getPlexNowPlaying(ctx context.Context, plexServer, plexToken string) (PlexMediaContainer, error) {
	resp, err := http.Get(strings.TrimSuffix(plexServer, "/") + "/status/sessions?X-Plex-Token=" + plexToken)
	if err != nil {
		return PlexMediaContainer{}, fmt.Errorf("error getting plex streaming status: %w", err)
	}
	body, err := readHTTPResponseBody(resp.Body)
	if err != nil {
		return PlexMediaContainer{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return PlexMediaContainer{}, fmt.Errorf("bad plex response:\nStatus: %s\nBody: %s", resp.Status, body)
	}
	var pmc PlexMediaContainer
	err = xml.Unmarshal([]byte(body), &pmc)
	if err != nil {
		return pmc, fmt.Errorf("error parsing plex response: %w\n\nresponse: %s", err, body)
	}
	return pmc, nil
}

func readHTTPResponseBody(body io.ReadCloser) (string, error) {
	defer body.Close()
	b, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("error reading plex response: %w", err)
	}
	return string(b), nil
}

func trackInfoFromPlex(pmc PlexMediaContainer, users []string) []TrackInfo {
	var resp []TrackInfo
	for _, track := range pmc.Tracks {
		if track.Type != "track" {
			continue
		}
		if track.Player.State != "playing" {
			continue
		}
		if len(users) > 0 {
			var match bool
			for _, user := range users {
				if track.User.Name == user {
					match = true
					break
				}
			}
			if !match {
				log.Printf("User %q doesn't match any of configured users %s, ignoring", track.User.Name, users)
				continue
			}
		}
		ti := TrackInfo{
			Name:    track.Title,
			Artists: []string{track.GrandparentTitle},
			Album:   track.ParentTitle,
		}
		if track.OriginalTitle != "" {
			ti.Artists = []string{track.OriginalTitle}
		}
		resp = append(resp, ti)
	}
	return resp
}
