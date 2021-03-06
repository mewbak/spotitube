package spotify

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"

	api "github.com/zmb3/spotify"
)

// AuthURL : generate new authentication URL
func AuthURL() *SpotifyAuthURL {
	clientAuthenticator.SetAuthInfo(SpotifyClientID, SpotifyClientSecret)
	spotifyURL := clientAuthenticator.AuthURL(clientState)
	tinyURL := fmt.Sprintf("http://tinyurl.com/api-create.php?url=%s", spotifyURL)
	tinyResponse, tinyErr := http.Get(tinyURL)
	if tinyErr != nil {
		return &SpotifyAuthURL{Full: spotifyURL, Short: ""}
	}
	defer tinyResponse.Body.Close()
	tinyContent, tinyErr := ioutil.ReadAll(tinyResponse.Body)
	if tinyErr != nil {
		return &SpotifyAuthURL{Full: spotifyURL, Short: ""}

	}
	return &SpotifyAuthURL{Full: spotifyURL, Short: string(tinyContent)}

}

// NewClient : return a new Spotify instance
func NewClient() *Spotify {
	return &Spotify{}
}

// Auth : start local callback server to handle xdg-preferred browser authentication redirection
func (spotify *Spotify) Auth(url string) bool {
	http.HandleFunc("/favicon.ico", webHTTPFaviconHandler)
	http.HandleFunc("/callback", webHTTPCompleteAuthHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go http.ListenAndServe(":8080", nil)

	commandCmd := "xdg-open"
	commandArgs := []string{url}
	_, err := exec.Command(commandCmd, commandArgs...).Output()
	if err != nil {
		return false
	}

	spotify.Client = <-clientChannel

	return true
}

// LibraryTracks : return array of Spotify FullTrack of all authenticated user library songs
func (spotify *Spotify) LibraryTracks() ([]api.FullTrack, error) {
	var (
		tracks     []api.FullTrack
		iterations int
		options    = defaultOptions()
	)
	for true {
		*options.Offset = *options.Limit * iterations
		chunk, err := spotify.Client.CurrentUsersTracksOpt(&options)
		if err != nil {
			return []api.FullTrack{}, fmt.Errorf(fmt.Sprintf("Something gone wrong while reading %dth chunk of tracks: %s.", iterations, err.Error()))
		}
		for _, track := range chunk.Tracks {
			tracks = append(tracks, track.FullTrack)
		}
		if len(chunk.Tracks) < 50 {
			break
		}
		iterations++
	}
	return tracks, nil
}

// Playlist : return Spotify FullPlaylist from input string playlistURI
func (spotify *Spotify) Playlist(playlistURI string) (*api.FullPlaylist, error) {
	playlistOwner, playlistID, playlistErr := parsePlaylistURI(playlistURI)
	if playlistErr != nil {
		return &api.FullPlaylist{}, playlistErr
	}
	return spotify.Client.GetPlaylist(playlistOwner, playlistID)
}

// PlaylistTracks : return array of Spotify FullTrack of all input string playlistURI identified playlist
func (spotify *Spotify) PlaylistTracks(playlistURI string) ([]api.FullTrack, error) {
	var (
		tracks     []api.FullTrack
		iterations int
		options    = defaultOptions()
	)
	playlistOwner, playlistID, playlistErr := parsePlaylistURI(playlistURI)
	if playlistErr != nil {
		return tracks, playlistErr
	}
	for true {
		*options.Offset = *options.Limit * iterations
		chunk, err := spotify.Client.GetPlaylistTracksOpt(playlistOwner, playlistID, &options, "")
		if err != nil {
			return []api.FullTrack{}, fmt.Errorf(fmt.Sprintf("Something gone wrong while reading %dth chunk of tracks: %s.", iterations, err.Error()))
		}
		for _, track := range chunk.Tracks {
			tracks = append(tracks, track.Track)
		}
		if len(chunk.Tracks) < 50 {
			break
		}
		iterations++
	}
	return tracks, nil
}

// Albums : return array Spotify FullAlbum, specular to the array of Spotify ID
func (spotify *Spotify) Albums(ids []api.ID) ([]api.FullAlbum, error) {
	var (
		albums     []api.FullAlbum
		iterations int
		upperbound int
		lowerbound int
	)
	for true {
		lowerbound = iterations * 20
		if upperbound = lowerbound + 20; upperbound >= len(ids) {
			upperbound = lowerbound + (len(ids) - lowerbound)
		}
		chunk, err := spotify.Client.GetAlbums(ids[lowerbound:upperbound]...)
		if err != nil {
			var chunk []api.FullAlbum
			for _, albumID := range ids[lowerbound:upperbound] {
				album, err := spotify.Client.GetAlbum(albumID)
				if err == nil {
					chunk = append(chunk, *album)
				} else {
					chunk = append(chunk, api.FullAlbum{})
				}
			}
		}
		for _, album := range chunk {
			albums = append(albums, *album)
		}
		if len(chunk) < 20 {
			break
		}
		iterations++
	}
	return albums, nil
}
