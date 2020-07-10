package tidl

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"time"
	"strconv"
	"strings"

	// TODO(ts): look at replacing bitio

	"github.com/icza/bitio"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"

	"github.com/h2non/filetype"
	"github.com/h2non/filetype/types"

	// "github.com/davecgh/go-spew/spew"
	"context"
	"golang.org/x/time/rate"
)

const baseurl = "https://api.tidalhifi.com/v1/"
const clientVersion = "1.9.1"

const (
	AQ_LOSSLESS int = iota
	AQ_HI_RES
)

var limiter *rate.Limiter

func init() {
	limiter = rate.NewLimiter(3, 10)
}

type cTime time.Time

func (ct *cTime) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	
	t, err := time.Parse("2006-01-02T15:04:05-0700", strings.Replace(string(b), `"`, "", -1))
	*ct = cTime(t)
	return err
}

var cookieJar, _ = cookiejar.New(nil)
var c = &http.Client{
	Jar: cookieJar,
}

type TidalError struct {
	Status      int
	SubStatus   int
	UserMessage string
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
	ID   json.Number `json:"id"`
	Name string      `json:"name"`
	Type string      `json:"type"`
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
	artBody              []byte
}

type Playlist struct {
	Creator struct {
		ID int
	}

	ID string

	Description string
	Created cTime
	URL string
	SquareImage string
	LastItemAddedAt cTime
	Image string
	Popularity float64
	LastUpdated cTime
	NumberOfTracks int
	Duration int
	Type string
	PublicPlaylist bool
	Title string

	Tracks []Track `json:"-"`

	artBody []byte
}

func (p *Playlist) GetArt() ([]byte, error) {
	u := "https://resources.tidal.com/images/" + strings.Replace(p.SquareImage, "-", "/", -1) + "/320x320.jpg"
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

// Track struct
type Track struct {
	Artists      []Artist    `json:"artists"`
	Artist       Artist      `json:"artist"`
	Album        Album       `json:"album"`
	Playlist 	 Playlist	 `json:"-"`
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
	err := limiter.Wait(context.Background())
	// fmt.Println(baseurl+dest+"?"+query.Encode(), t.SessionID)
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

	// if res.StatusCode := 200 {
	// 	fmt.Println(res.StatusCode)
	// }

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

func (t *Tidal) GetFavoriteAlbums() ([]string, error) {
	var out struct {
		Limit              int `json:"limit"`
		Offset             int `json:"offset"`
		TotalNumberOfItems int `json:"totalNumberOfItems"`
		Items              []struct {
			Created string `json:"created"`
			Item    struct {
				ID                   int         `json:"id"`
				Title                string      `json:"title"`
				Duration             int         `json:"duration"`
				StreamReady          bool        `json:"streamReady"`
				StreamStartDate      string      `json:"streamStartDate"`
				AllowStreaming       bool        `json:"allowStreaming"`
				PremiumStreamingOnly bool        `json:"premiumStreamingOnly"`
				NumberOfTracks       int         `json:"numberOfTracks"`
				NumberOfVideos       int         `json:"numberOfVideos"`
				NumberOfVolumes      int         `json:"numberOfVolumes"`
				ReleaseDate          string      `json:"releaseDate"`
				Copyright            string      `json:"copyright"`
				Type                 string      `json:"type"`
				Version              interface{} `json:"version"`
				URL                  string      `json:"url"`
				Cover                string      `json:"cover"`
				VideoCover           interface{} `json:"videoCover"`
				Explicit             bool        `json:"explicit"`
				Upc                  string      `json:"upc"`
				Popularity           int         `json:"popularity"`
				AudioQuality         string      `json:"audioQuality"`
				SurroundTypes        interface{} `json:"surroundTypes"`
				Artist               struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
					Type string `json:"type"`
				} `json:"artist"`
				Artists []struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
					Type string `json:"type"`
				} `json:"artists"`
			} `json:"item"`
		} `json:"items"`
	}

	err := t.get(fmt.Sprintf("users/%s/favorites/albums", t.UserID), &url.Values{
		"limit": {"500"},
	}, &out)
	var ids []string

	for _, id := range out.Items {
		ids = append(ids, strconv.Itoa(id.Item.ID))
	}

	return ids, err
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
	_, err = s.GetArt()

	t.albumMap[id] = s

	if s.Duration == 0 {
		return s, errors.New("album unavailable")
	}

	return s, err
}

// GetAlbumTracks func
func (t *Tidal) GetAlbumTracks(id string) ([]Track, error) {
	var s struct {
		Items []Track `json:"items"`
	}
	return s.Items, t.get("albums/"+id+"/tracks", &url.Values{}, &s)
}

// GetPlaylist func 
func (t *Tidal) GetPlaylist(id string) (Playlist, error) {
	var s Playlist

	s.ID = id

	return s, t.get("playlists/"+id, &url.Values{}, &s)
}

// GetPlaylistTracks func
func (t *Tidal) GetPlaylistTracks(id string) ([]Track, error) {
	var s struct {
		Items []Track
	}

	values := &url.Values{}
	values.Add("limit", "500")

	return s.Items, t.get("playlists/"+id+"/tracks", values, &s) 
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

func (t *Tidal) GetArtist(artist string) (Artist, error) {
	var s Artist
	err := t.get(fmt.Sprintf("artists/%s", artist), &url.Values{}, &s)
	return s, err
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

func (t *Tidal) GetArtistEP(artist string, l int) ([]Album, error) {
	var s Search
	var limit string

	if l > 0 {
		limit = strconv.Itoa(l)
	}

	err := t.get(fmt.Sprintf("artists/%s/albums", artist), &url.Values{
		"limit":  {limit},
		"filter": {"EPSANDSINGLES"},
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
	al.artBody, err = ioutil.ReadAll(res.Body)
	return al.artBody, err
}

func (t *Tidal) DownloadAlbum(al Album) error {
	tracks, err := t.GetAlbumTracks(al.ID.String())
	if err != nil {
		return err
	}

	if al.Duration == 0.0 {
		return errors.New("album unavailable")
	}

	dirs := clean(al.Artists[0].Name) + "/" + clean(al.Title)
	os.MkdirAll(dirs, os.ModePerm)

	metadata, err := json.MarshalIndent(al, "", "\t")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dirs+"/meta.json", metadata, 0777)
	if err != nil {
		return err
	}

	body, err := al.GetArt()
	if err != nil {
		return err
	}

	al.artBody = body
	t.albumMap[al.ID.String()] = al

	err = ioutil.WriteFile(dirs+"/album.jpg", body, 0777)
	if err != nil {
		return err
	}

	for i, track := range tracks {
		fmt.Printf("\t [%v/%v] %v\n", i+1, len(tracks), track.Title)
		if err := t.DownloadTrack(dirs, track); err != nil {
			return err
		}
	}

	return nil
}

func (t *Tidal) DownloadPlaylist(p Playlist) error {
	if p.Duration == 0 {
		return errors.New("playlist unavailable")
	}

	if len(p.Tracks) == 0 {
		var err error
		p.Tracks, err = t.GetPlaylistTracks(p.ID)
		if err != nil {
			return err
		}
	}

	root := "Playlists/" + clean(p.Title)
	os.MkdirAll(root, os.ModePerm)

	metadata, err := json.MarshalIndent(p, "", "\t")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(root+"/meta.json", metadata, 0777)
	if err != nil {
		return err
	}

	body, err := p.GetArt()
	if err != nil {
		return err
	}

	p.artBody = body

	err = ioutil.WriteFile(root+"/album.jpg", body, 0777)
	if err != nil {
		return err
	}

	for i, tr := range p.Tracks {
		fmt.Printf("\t [%v/%v] %v - %v\n", i+1, len(p.Tracks), tr.Artist.Name, tr.Title)
		// TODO(ts): improve ID3

		tr.Playlist = p
		tr.TrackNumber = json.Number(strconv.Itoa(i+1))

		if tr.DoExists(root) {
			continue
		}

		t.GetAlbum(string(tr.Album.ID))
		t.DownloadTrack(root, tr)
	}

	return nil
}

func (tr Track) GetPath(root string) string {
	return root + "/" + clean(tr.Artist.Name) + " - " + clean(tr.Title)
}

func (tr Track) DoExists(root string) bool {
	path := tr.GetPath(root)
	matches, err := filepath.Glob("./" + path + ".*")
	if err != nil {
		return false
	}

	return (len(matches) > 0)
}

func (t Tidal) DownloadTrack(root string, tr Track) error {
	if tr.DoExists(root) {
		return nil
	}

	// TODO(ts): improve ID3
	al := t.albumMap[tr.Album.ID.String()]
	tr.Album = al

	os.MkdirAll(root, os.ModePerm)

	path := tr.GetPath(root)
	
	u, err := t.GetStreamURL(tr.ID.String(), "LOSSLESS")
	if err != nil {
		panic(err)
	}

	if u == "" {
		return nil
	}

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}

	res, err := http.Get(u)
	if err != nil {
		panic(err)
	}

	buf, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()
	
	f.Write(buf)
	f.Close()

	kind, _ := filetype.Match(buf)
	err = enc(root, tr, kind)
	if err != nil {
		panic(err)
	}
	os.Remove(path)

	return nil
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
	}

	req, err := http.NewRequest("POST", baseurl + "login/username", strings.NewReader(query.Encode()))
	
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("X-Tidal-Token", "wc8j_yBJd20zOmx0")

	client := &http.Client{}
	
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		msg, _ := ioutil.ReadAll(res.Body)
		fmt.Println(string(msg))
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

func encFlac(src string, tr Track) error {
	// https://wiki.hydrogenaud.io/index.php?title=Tag_Mapping#Titles
	// Decode FLAC file.
	path := src + "/" + clean(tr.Artist.Name) + " - " + clean(tr.Title)
	stream, err := flac.ParseFile(path)
	if err != nil {
		// isn't a FLAC file
		return err
	}

	// https://xiph.org/flac/format.html#metadata_block_picture
	MIMETYPE := "image/jpeg"
	pictureData := &bytes.Buffer{}
	w := bitio.NewWriter(pictureData)
	w.WriteBits(uint64(3), 32)                     // picture type (3)
	w.WriteBits(uint64(len(MIMETYPE)), 32)         // length of "image/jpeg"
	w.Write([]byte(MIMETYPE))                      // "image/jpeg"
	w.WriteBits(uint64(0), 32)                     // description length (0)
	w.Write([]byte{})                              // description
	w.WriteBits(uint64(1280), 32)                  // width (1280)
	w.WriteBits(uint64(1280), 32)                  // height (1280)
	w.WriteBits(uint64(24), 32)                    // colour depth (24)
	w.WriteBits(uint64(0), 32)                     // is pal? (0)
	w.WriteBits(uint64(len(tr.Album.artBody)), 32) // length of content
	w.Write(tr.Album.artBody)                      // actual content
	w.Close()

	encodedPictureData := base64.StdEncoding.EncodeToString(pictureData.Bytes())
	foundComments := false
	extraComments := ""

	albumName := tr.Album.Title
	trackTotal := tr.Album.NumberOfTracks.String()

	if tr.Playlist.ID != "" {
		extraComments += fmt.Sprintf("Original Album Title: %v\n", albumName)
		albumName = tr.Playlist.Title
		trackTotal = strconv.Itoa(tr.Playlist.NumberOfTracks)
	}

	comments := [][2]string{}
	comments = append(comments, [2]string{"TITLE", tr.Title})
	comments = append(comments, [2]string{"ALBUM", albumName})
	comments = append(comments, [2]string{"TRACKNUMBER", tr.TrackNumber.String()})
	comments = append(comments, [2]string{"TRACKTOTAL", trackTotal})

	comments = append(comments, [2]string{"ARTIST", tr.Artist.Name})
	comments = append(comments, [2]string{"ALBUMARTIST", tr.Album.Artist.Name})
	
	comments = append(comments, [2]string{"COPYRIGHT", tr.Copyright})
	comments = append(comments, [2]string{"METADATA_BLOCK_PICTURE", encodedPictureData})
	comments = append(comments, [2]string{"COMMENTS", extraComments})

	// Add custom vorbis comment.
	for _, block := range stream.Blocks {
		if comment, ok := block.Body.(*meta.VorbisComment); ok {
			foundComments = true
			comment.Tags = append(comment.Tags, comments...)
		}
	}

	if foundComments == false {
		block := new(meta.Block)
		block.IsLast = true
		block.Type = meta.Type(4)
		block.Length = 0

		comment := new(meta.VorbisComment)
		block.Body = comment
		comment.Vendor = "Lavf57.71.100"
		comment.Tags = append(comment.Tags, comments...)

		stream.Blocks = append(stream.Blocks, block)
	}

	// Encode FLAC file.
	f, err := os.Create(path + ".flac")
	if err != nil {
		return err
	}
	err = flac.Encode(f, stream)
	f.Close()
	stream.Close()

	return nil
}

func encMp4(src string, tr Track) error {
	path := src + "/" + clean(tr.Artist.Name) + " - " + clean(tr.Title)
	return os.Rename(path, path + ".mp4")

	return nil
}

func enc(src string, tr Track, kind types.Type) error {
	switch kind.MIME.Value {
	case "audio/x-flac":
		return encFlac(src, tr)
	case "audio/mp4", "video/mp4":
		return encMp4(src, tr)
	default:
		fmt.Println(kind.MIME.Value)
		return nil
	}
}
