package main

import (
	"flag"
	"fmt"
	"github.com/odwrtw/transmission"
	"github.com/radovskyb/watcher"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path"
	"regexp"
	"syscall"
	"time"
)

var directory = flag.String("directory", ".", "Directory to watch, will watch the current directory if not provided")
var fileNamePattern = flag.String("fileNamePattern", ".*\\.magnet$", "The pattern to match file names, by default it look for files with extension \".magnet\"")
var processFilesOnStart = flag.Bool("process-files-on-start", true, "Process the files when the application starts")
var help = flag.Bool("help", false, "Print the help message")
var debug = flag.Bool("debug", false, "Log more information for debug purposes")

// transmission client configuration
var transmissionUrl = flag.String("transmission-url", "http://localhost:9091", "The URL of transmission")
var transmissionApiPath = flag.String("transmission-api-path", "/transmission/rpc", "The path of the transmission API")
var userName = flag.String("transmission-username", "", "The username to authenticate in transmission API")
var password = flag.String("transmission-password", "", "The password to authenticate in transmission API")

func logDebug(format string, v ...interface{}) {
	if *debug {
		log.Printf(format, v)
	}
}

func main() {
	flag.Parse()

	if *help == true {
		flag.Usage()
		os.Exit(0)
	}

	var dirToWatch string

	if *directory == "." || *directory == "" {
		dir, err := os.Getwd()

		if err != nil {
			log.Fatalln("Failed to get current path", err)
		}

		dirToWatch = dir
	} else {
		dirToWatch = path.Clean(*directory)
	}

	fileNameRegex := regexp.MustCompile(*fileNamePattern)

	log.Printf("Watching for magnet links at directory %s", dirToWatch)
	log.Printf("Matching files with pattern: %s", *fileNamePattern)

	url := fmt.Sprintf("%s%s", *transmissionUrl, *transmissionApiPath)

	conf := transmission.Config{
		Address:  url,
		User:     *userName,
		Password: *password,
	}

	transmissionClient, err := transmission.New(conf)
	if err != nil {
		log.Fatal(err)
	}

	// verify the connection to transmission is working
	_, err = transmissionClient.GetTorrents()
	if err != nil {
		log.Fatalln(err)
	}

	filesToProcess := make(chan string, 1024)

	go func() {
		fileProcessor(filesToProcess, transmissionClient)
	}()

	w := watcher.New()
	w.AddFilterHook(watcher.RegexFilterHook(fileNameRegex, false))

	if err := w.AddRecursive(dirToWatch); err != nil {
		log.Fatalln(err)
	}

	// monitor the folders
	go func() {
		for {
			select {
			case event := <-w.Event:
				// created a new file
				if event.Op == watcher.Write || event.Op == watcher.Create {
					logDebug("Found file %s", event.Path)

					// queue the message to be processed
					filesToProcess <- event.Path
				} else {
					logDebug("Event: %v", event)
				}
			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	if *processFilesOnStart {
		// process the files that are already in the folder
		go func() {
			// wait for the monitor to start
			w.Wait()

			log.Printf("Current files in the directory")

			for watchedFilePath := range w.WatchedFiles() {
				log.Printf(watchedFilePath)

				filesToProcess <- watchedFilePath
			}
		}()
	}

	signalChan := make(chan os.Signal, 2)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// listen for OS signals
	go func() {
		for {
			select {
			case s := <-signalChan:
				switch s {
				case syscall.SIGINT, syscall.SIGTERM:
					log.Printf("Got SIGINT/SIGTERM, exiting...")
					w.Close()
				}
			}
		}
	}()

	// start and block on the folder monitoring
	if err := w.Start(time.Millisecond * 1000); err != nil {
		log.Fatalln(err)
	}
}

// Process the files found by the watcher
func fileProcessor(files <-chan string, transmissionClient *transmission.Client) {
	// internal queue, will receive the file paths with some delay to allow for any I/O flush
	// prior to actually reading and sending it to transmission
	queue := make(chan string, 1024)

	for {
		select {
		case filePath := <-files:
			// wait a bit before actually processing the file, because
			// we could be notified while it's being written and we could have incomplete
			// contents there
			time.AfterFunc(time.Second*3, func() {
				queue <- filePath
			})

		case file := <-queue:
			log.Printf("Processing file: %s", file)

			bytes, err := ioutil.ReadFile(file)

			if err != nil {
				log.Printf("Failed to read file %s: %v", file, err)
			} else {
				go func() {
					sendMagnet(file, transmissionClient, bytes, queue)
				}()
			}
		}
	}
}

func sendMagnet(file string, transmissionClient *transmission.Client, bytes []byte, queue chan string) {
	magnet := path.Base(file)
	log.Printf("Adding torrent from magnet %s", magnet)

	torrent, err := transmissionClient.Add(string(bytes))

	if err == nil {
		log.Printf("Magnet %s added successfully: %s", magnet, torrent.TorrentFile)

		renameCompletedFile(file)
		return
	}

	if err == transmission.ErrDuplicateTorrent {
		log.Printf("The torrent was already added")
	} else {
		log.Printf("Failed to add magnet %s: %v", magnet, err)

		// add in the queue after some time to retry
		time.AfterFunc(time.Second*10, func() {
			queue <- file
		})
	}
}

func renameCompletedFile(file string) {
	if e := os.Rename(file, fmt.Sprintf("%s.added", file)); e != nil {
		log.Printf("Failed to move file %s to %s.added", file, file)
	} else {
		log.Printf("Renamed file %s to %s.added", file, file)
	}
}
