package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/trevorstarick/tidl"
)

var username, password string

var onlyAlbums = flag.Bool("albums", false, "only download albums")
var onlyEPs = flag.Bool("eps", false, "only download eps and singles")

var altUsername = flag.String("username", "", "optional username when not set in build process")
var altPassword = flag.String("password", "", "optional password when not set in build process")

func main() {
	var err error

	flag.Parse()

	// TODO(TS): look into input prompt
	if username == "" {
		username = *altUsername
		password = *altPassword
	} else if password == "" {
		password = *altPassword
	}

	if username == "" {
		fmt.Println("missing username")
		os.Exit(1)
	}

	if password == "" {
		fmt.Println("missing password")
		os.Exit(1)
	}

	t, err := tidl.New(username, password)
	if err != nil {
		fmt.Println("can't login to tidl right now")
		os.Exit(4)
	}

	var ids []string

	// TODO(ts): handle output better
	// TODO(ts): handle no input
	if len(flag.Args()) == 0 {
		ids, _ = t.GetFavoriteAlbums()
		for _, id := range ids {
			fmt.Println(id)
		}
		os.Exit(2)
	}

	if _, err = os.Stat(flag.Args()[0]); !os.IsNotExist(err) {
		f, err := os.Open(flag.Args()[0])
		if err != nil {
			fmt.Println("can't open file")
			os.Exit(3)
		}

		buffer := bufio.NewScanner(f)
		for buffer.Scan() {
			ids = append(ids, buffer.Text())
		}
	} else {
		ids = flag.Args()
	}

	for _, id := range ids {
		var albums []tidl.Album

		if id[0] == 'h' {
			id = strings.Split(id, "album/")[1]
		}

		// TODO(ts): support fetching of EP/Singles as well as flags to disable
		// TODO(ts): support fetching of artist info
		artist, err := t.GetArtist(id)
		if err != nil {
			fmt.Println("can't get artist info")
			os.Exit(5)
		}

		if artist.ID.String() != "" {
			fmt.Printf("Downloading %v (%v)...\n", artist.Name, artist.ID)

			if *onlyAlbums == true {
				fmt.Println("Only fetching Albums")
				lbums, err := t.GetArtistAlbums(id, 0)
				if err != nil {
					fmt.Println("can't get artist albums")
					os.Exit(5)
				}

				albums = append(albums, lbums...)
			} else if *onlyEPs {
				fmt.Println("Only fetching EPs & Singles")
				lbums, err := t.GetArtistEP(id, 0)
				if err != nil {
					fmt.Println("can't get artist eps")
					os.Exit(5)
				}

				albums = append(albums, lbums...)
			} else {
				fmt.Println("Fetching Albums, EPs & Singles")
				lbums, err := t.GetArtistAlbums(id, 0)
				if err != nil {
					fmt.Println("can't get artist albums")
					os.Exit(5)
				}

				albums = append(albums, lbums...)

				lbums, err = t.GetArtistEP(id, 0)
				if err != nil {
					fmt.Println("can't get artist eps")
					os.Exit(5)
				}

				albums = append(albums, lbums...)
			}
		} else {
			album, err := t.GetAlbum(id)
			if err != nil {
				fmt.Println("can't get album info: " + id)
				os.Exit(6)
			}

			albums = []tidl.Album{album}
		}

		albumMap := make(map[string]tidl.Album)
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

		albums = make([]tidl.Album, 0, len(albumMap))
		for _, album := range albumMap {
			if album.Duration > 0 {
				albums = append(albums, album)
			}
		}

		for _, album := range albums {
			fmt.Printf("[%v] %v - %v\n", album.ID.String(), album.Artist.Name, album.Title)
			if err := t.DownloadAlbum(album); err != nil {
				fmt.Println("can't download album")
				os.Exit(8)
			}
			// 	tracks, err := t.GetAlbumTracks(album.ID.String())
			// 	if err != nil {
			// 		panic(err)
			// 	}

			// 	processQueue := make(chan Track, len(tracks))
			// 	for _, track := range tracks {
			// 		processQueue <- track
			// 	}
			// 	close(processQueue)

			// 	wg := sync.WaitGroup{}

			// 	for i := 0; i < 8; i++ {
			// 		wg.Add(1)
			// 		go func() {
			// 			for track := range processQueue {
			// 				t.DownloadTrack(track)
			// 				fmt.Printf("\t%v\n", track.Title)
			// 			}
			// 			wg.Done()
			// 		}()
			// 	}

			// 	wg.Wait()
		}
	}
}
