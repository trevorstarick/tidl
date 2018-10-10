package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"
)

const baseurl = "https://api.tidalhifi.com/v1/"
const clientVersion = "1.9.1"
const token = "kgsOOmYk3zShYrNP"

const (
	AQ_LOSSLESS int = iota
	AQ_HI_RES
)

var username, password string
var cookieJar, _ = cookiejar.New(nil)
var c = &http.Client{
	Jar: cookieJar,
}

// Tidal api struct
type Tidal struct {
	albumMap    map[string]Album
	SessionID   string      `json:"sessionID"`
	CountryCode string      `json:"countryCode"`
	UserID      json.Number `json:"userId"`
}

// Artist struct
type Artist struct {
	ID         json.Number `json:"id"`
	Name       string      `json:"name"`
	Type       string      `json:"type"`
	Popularity int         `json:"popularity,omitempty"`
}

// Album struct
type Album struct {
	Artists              []Artist    `json:"artists,omitempty"`
	Title                string      `json:"title"`
	ID                   json.Number `json:"id"`
	NumberOfTracks       json.Number `json:"numberOfTracks"`
	Explicit             bool        `json:"explicit,omitempty"`
	Copyright            string      `json:"copyright,omitempty"`
	AudioQuality         string      `json:"audioQuality"`
	ReleaseDate          string      `json:"releaseDate"`
	Duration             float64     `json:"duration"`
	PremiumStreamingOnly bool        `json:"premiumStreamingOnly"`
	Popularity           float64     `json:"popularity,omitempty"`
	Artist               Artist      `json:"artist"`
	Cover                string      `json:"cover"`
}

// Track struct
type Track struct {
	Artists      []Artist    `json:"artists"`
	Artist       Artist      `json:"artist"`
	Album        Album       `json:"album"`
	Title        string      `json:"title"`
	ID           json.Number `json:"id"`
	Explicit     bool        `json:"explicit"`
	Copyright    string      `json:"copyright"`
	Popularity   int         `json:"popularity"`
	TrackNumber  json.Number `json:"trackNumber"`
	Duration     json.Number `json:"duration"`
	AudioQuality string      `json:"audioQuality"`
}

// Search struct
type Search struct {
	Items  []Album `json:"items"`
	Albums struct {
		Items []Album `json:"items"`
	} `json:"albums"`
	Artists struct {
		Items []Artist `json:"items"`
	} `json:"artists"`
	Tracks struct {
		Items []Track `json:"items"`
	} `json:"tracks"`
}

func (t *Tidal) get(dest string, query *url.Values, s interface{}) error {
	// log.Println(baseurl + dest)
	req, err := http.NewRequest("GET", baseurl+dest, nil)
	if err != nil {
		return err
	}
	req.Header.Add("X-Tidal-SessionID", t.SessionID)
	query.Add("countryCode", t.CountryCode)
	req.URL.RawQuery = query.Encode()
	res, err := c.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	return json.NewDecoder(res.Body).Decode(&s)
}

func (t *Tidal) CheckSession() (bool, error) {
	//if self.user is None or not self.user.id or not self.session_id:
	//return False
	var out interface{}
	err := t.get(fmt.Sprintf("users/%s/subscription", t.UserID), nil, &out)
	// fmt.Println(out)
	return true, err
}

// GetStreamURL func
func (t *Tidal) GetStreamURL(id, q string) (string, error) {
	var s struct {
		URL string `json:"url"`
	}
	err := t.get("tracks/"+id+"/streamUrl", &url.Values{
		"soundQuality": {q},
	}, &s)
	return s.URL, err
}

func (t *Tidal) GetAlbum(id string) (Album, error) {
	var s Album

	if album, ok := t.albumMap[id]; ok {
		return album, nil
	}

	err := t.get("albums/"+id, &url.Values{}, &s)
	t.albumMap[id] = s

	return s, err
}

// GetAlbumTracks func
func (t *Tidal) GetAlbumTracks(id string) ([]Track, error) {
	var s struct {
		Items []Track `json:"items"`
	}
	return s.Items, t.get("albums/"+id+"/tracks", &url.Values{}, &s)
}

// GetPlaylistTracks func
func (t *Tidal) GetPlaylistTracks(id string) ([]Track, error) {
	var s struct {
		Items []Track `json:"items"`
	}
	return s.Items, t.get("playlists/"+id+"/tracks", &url.Values{}, &s)
}

// SearchTracks func
func (t *Tidal) SearchTracks(d string, l int) ([]Track, error) {
	var s Search
	var limit string

	if l > 0 {
		limit = strconv.Itoa(l)
	}

	return s.Tracks.Items, t.get("search", &url.Values{
		"query": {d},
		"types": {"TRACKS"},
		"limit": {limit},
	}, &s)
}

// SearchAlbums func
func (t *Tidal) SearchAlbums(d string, l int) ([]Album, error) {
	var s Search
	var limit string

	if l > 0 {
		limit = strconv.Itoa(l)
	}

	err := t.get("search", &url.Values{
		"query": {d},
		"types": {"ALBUMS"},
		"limit": {limit},
	}, &s)

	if err != nil {
		return s.Albums.Items, err
	}

	for _, album := range s.Albums.Items {
		t.albumMap[album.ID.String()] = album
	}

	return s.Albums.Items, nil
}

// SearchArtists func
func (t *Tidal) SearchArtists(d string, l int) ([]Artist, error) {
	var s Search
	var limit string

	if l > 0 {
		limit = strconv.Itoa(l)
	}

	return s.Artists.Items, t.get("search", &url.Values{
		"query": {d},
		"types": {"ARTISTS"},
		"limit": {limit},
	}, &s)
}

// GetArtistAlbums func
func (t *Tidal) GetArtistAlbums(artist string, l int) ([]Album, error) {
	var s Search
	var limit string

	if l > 0 {
		limit = strconv.Itoa(l)
	}

	err := t.get(fmt.Sprintf("artists/%s/albums", artist), &url.Values{
		"limit": {limit},
	}, &s)

	if err != nil {
		return s.Items, err
	}

	for _, album := range s.Items {
		t.albumMap[album.ID.String()] = album
	}

	return s.Items, nil
}

func (al *Album) GetArt() ([]byte, error) {
	u := "https://resources.tidal.com/images/" + strings.Replace(al.Cover, "-", "/", -1) + "/1280x1280.jpg"
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

func (t *Tidal) DownloadAlbum(al Album) {
	tracks, err := t.GetAlbumTracks(al.ID.String())
	if err != nil {
		panic(err)
	}

	dirs := clean(al.Artists[0].Name) + "/" + clean(al.Title)
	os.MkdirAll(dirs, os.ModePerm)

	for _, track := range tracks {
	meta, err := json.MarshalIndent(al, "", "\t")
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(dirs+"/meta.json", meta, 0777)
	if err != nil {
		panic(err)
	}

	body, err := al.GetArt()
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(dirs+"/album.jpg", body, 0777)
	if err != nil {
		panic(err)
	}
		t.DownloadTrack(track)
	}
}

func (t *Tidal) DownloadTrack(tr Track) {
	dirs := clean(tr.Artists[0].Name) + "/" + clean(tr.Album.Title)
	path := dirs + "/" + clean(tr.Artists[0].Name) + " - " + clean(tr.Title)
	al := t.albumMap[tr.Album.ID.String()]

	os.MkdirAll(dirs, os.ModePerm)
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}

	u, err := t.GetStreamURL(tr.ID.String(), "LOSSLESS")
	if err != nil {
		log.Fatal(err)
	}
	res, err := http.Get(u)
	if err != nil {
		log.Fatal(err)
	}

	io.Copy(f, res.Body)
	res.Body.Close()
	f.Close()

	err = enc(dirs, tr)
	if err != nil {
		panic(err)
	}
	os.Remove(path)
}

// helper function to generate a uuid
func uuid() string {
	b := make([]byte, 16)
	rand.Read(b[:])
	b[8] = (b[8] | 0x40) & 0x7F
	b[6] = (b[6] & 0xF) | (4 << 4)
	return fmt.Sprintf("%x", b)
}

// New func
func New(user, pass string) (*Tidal, error) {
	query := url.Values{
		"username":        {user},
		"password":        {pass},
		"token":           {token},
		"clientUniqueKey": {uuid()},
		"clientVersion":   {clientVersion},
	}
	res, err := http.PostForm(baseurl+"login/username", query)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected error code from tidal: %d", res.StatusCode)
	}

	defer res.Body.Close()
	var t Tidal
	t.albumMap = make(map[string]Album)
	return &t, json.NewDecoder(res.Body).Decode(&t)
}

func clean(s string) string {
	return strings.Replace(s, "/", "\u2215", -1)
}

func enc(src string, tr Track) error {
	// Decode FLAC file.
	path := src + "/" + clean(tr.Artist.Name) + " - " + clean(tr.Title)
	stream, err := flac.ParseFile(path)
	if err != nil {
		return err
	}

	// Add custom vorbis comment.
	for _, block := range stream.Blocks {
		if comment, ok := block.Body.(*meta.VorbisComment); ok {
			comment.Tags = append(comment.Tags, [2]string{"TITLE", tr.Title})
			comment.Tags = append(comment.Tags, [2]string{"ALBUM", tr.Album.Title})
			comment.Tags = append(comment.Tags, [2]string{"TRACKNUMBER", tr.TrackNumber.String()})
			comment.Tags = append(comment.Tags, [2]string{"ARTIST", tr.Artist.Name})
			comment.Tags = append(comment.Tags, [2]string{"COPYRIGHT", tr.Copyright})
		}
	}

	// Encode FLAC file.
	f, err := os.Create(path + ".flac")
	if err != nil {
		return err
	}
	err = flac.Encode(f, stream)
	f.Close()
	stream.Close()
	return err
}

func main() {
	// TODO(ts): handle output better
	// TODO(ts): handle no input
	if len(os.Args) == 1 {
		return
	}

	var ids []string

	if _, err := os.Stat(os.Args[1]); !os.IsNotExist(err) {
		f, err := os.Open(os.Args[1])
		if err != nil {
			panic(err)
		}

		buffer := bufio.NewScanner(f)
		for buffer.Scan() {
			ids = append(ids, buffer.Text())
		}
	} else {
		ids = os.Args[1:]
	}

	t, err := New(username, password)
	if err != nil {
		panic(err)
	}

	// spew.Dump(t)

	for _, id := range ids {
		// TODO(ts): support fetching of EP/Singles as well as flags to disable
		// TODO(ts): support fetching of artist info
		albums, err := t.GetArtistAlbums(ids[0], 1)
		if err != nil {
			panic(err)
		}

		if len(albums) == 0 {
			album, err := t.GetAlbum(id)
			if err != nil {
				panic(err)
			}

			albums = []Album{album}
		}

		albumMap := make(map[string]Album)
		for _, album := range albums {
			if _, ok := albumMap[album.Title]; !ok {
				albumMap[album.Title] = album
			} else {
				// TODO(ts): impove dedupe if statement

				if album.AudioQuality == "LOSSLESS" && albumMap[album.Title].AudioQuality != "LOSSLESS" {
					// if higher quality
					albumMap[album.Title] = album
				} else if album.Explicit && !albumMap[album.Title].Explicit {
					// if explicit
					albumMap[album.Title] = album
				} else if album.Popularity > albumMap[album.Title].Popularity {
					// if more popular
					albumMap[album.Title] = album
				}
			}
		}

		albums = make([]Album, 0, len(albumMap))
		for _, album := range albumMap {
			albums = append(albums, album)
		}

		for _, album := range albums {
			t.DownloadAlbum(album)
		}
	}
}
