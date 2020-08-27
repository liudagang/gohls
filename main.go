package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"toolkit"
)

var wd string
var saveDir string
var ch chan string
var baseurl string

// @title parseContent
// @description parse m3u8 file, then send the segement to the channel
func parseContent(content string) {
	buf := bytes.NewBufferString(content)
	reg, err := regexp.Compile(`\.ts$`)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	for {
		line, err := buf.ReadString('\n')
		if err == io.EOF {
			break
		}
		line = strings.TrimSpace(line)
		if reg.MatchString(line) {
			// fmt.Println(line)
			ch <- line
		}
	}

	for i := 0; i < 20; i++ {
		ch <- ""
	}
}

// @title saveSegment
// @description download segments
func saveSegment() {
	for {
		seg := <-ch
		if len(seg) < 1 {
			fmt.Println("goroutine finish...")
			break
		}
		url := baseurl + seg

		// save file
		savefilename := saveDir + "/" + path.Base(url)
		if toolkit.FileExists(savefilename) {
			fmt.Println("skip " + url)
			continue
		}

		for {
			fmt.Println("download", url)
			reader, err := toolkit.GetRemoteFileReader(url)
			if err != nil {
				// retry it..
				fmt.Println("retry", url)
				continue
			}

			// save content to file
			f, err := os.Create(savefilename)
			if err != nil {
				fmt.Println(err)
				continue
			}

			_, err = io.Copy(f, reader)
			if err != nil {
				fmt.Println(err)
				f.Close()
				continue
			}

			f.Close()
			break
		}
	}
}

func main() {
	flag.Parse()
	if flag.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "Usage: gohls media-playlist-url output-directory\n")
		os.Exit(0)
	}

	// check playlist url
	url := flag.Arg(0)
	reg, err := regexp.Compile(`^http[\w\W]+?\.m3u8$`)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	if !reg.MatchString(url) {
		fmt.Fprintln(os.Stderr, "invalid media-playlist-url")
		os.Exit(0)
	}

	// get base url
	baseurl = toolkit.PregReplace(url, `index\.m3u8$`, "")
	if len(baseurl) < 1 {
		fmt.Println("the media playlist url is not end with index.m3u8")
		os.Exit(0)
	}

	// create output directory if it do not exists
	wd, _ = os.Getwd()
	saveDir = wd + "/" + flag.Arg(1)

	if !toolkit.FileExists(saveDir) {
		err := os.Mkdir(saveDir, os.ModePerm)
		if err != nil {
			fmt.Println(err)
		}
	}

	// parse playlist file
	content, err := toolkit.GetRemoteFile(url)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	// save playlist file
	m3u8File := saveDir + "/index.m3u8"
	if !toolkit.FileExists(m3u8File) {
		err := ioutil.WriteFile(m3u8File, []byte(content), os.ModePerm)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
	}

	// parse & download
	ch = make(chan string, 20)
	go parseContent(content)
	for i := 0; i < 10; i++ {
		go saveSegment()
	}

	// wait
	waitCh := make(chan int)
	<-waitCh

	// open a webserver to the savedir
	// http.Handle("/", http.FileServer(http.Dir(saveDir)))
	// http.ListenAndServe(":80", nil)
}
