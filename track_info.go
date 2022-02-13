package main

import "strings"

type TrackInfo struct {
	Name      string
	Artists   []string
	Album     string
	SpotifyID string
	ISRC      string
}

func (t TrackInfo) String() string {
	var res []string
	res = append(res, "🎵 Now Playing: "+t.Name)
	if len(t.Artists) > 0 {
		res = append(res, "🎵 Artist: "+formatArtists(t.Artists))
	}
	if t.Album != "" {
		res = append(res, "🎵 Album: "+t.Album)
	}
	if t.SpotifyID != "" {
		res = append(res, "🎵 SpotifyID: "+t.SpotifyID)
	}
	if t.ISRC != "" {
		res = append(res, "🎵 ISRC: "+t.ISRC)
	}
	return strings.Join(res, "\n")
}

func formatArtists(artists []string) string {
	if len(artists) == 0 {
		return ""
	} else if len(artists) == 1 {
		return artists[0]
	} else if len(artists) == 2 {
		return artists[0] + " and " + artists[1]
	}
	artists[len(artists)-1] = "and " + artists[len(artists)-1]
	return strings.Join(artists, ", ")
}
