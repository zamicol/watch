package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/DisposaBoy/JsonConfigReader"
	"github.com/fsnotify/fsnotify"
)

var dircmd map[string]string
var cpath *string // Config file path.

func main() {
	run()
}

func run() {
	cpath = flag.String("config", "watch.json5", "path for the watch config.")
	flag.Parse()
	parseConfig(&dircmd)

	var expanded = make(map[string]string)
	for k, v := range dircmd {
		var err error
		// For windows slashes
		k, err = filepath.Abs(os.ExpandEnv(k))
		if err != nil {
			panic(err)
		}

		// For windows slashes
		v, err = filepath.Abs(os.ExpandEnv(v))
		if err != nil {
			panic(err)
		}

		expanded[k] = v
	}
	dircmd = expanded

	done := make(chan bool)
	for dir, cmd := range dircmd {
		go Watch(dir, cmd)
	}
	<-done
}

// Watch is for each dir/cmd to watch/run.
func Watch(dir, cmd string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Rename == fsnotify.Rename {
					log.Printf("running %q\n", cmd)
					start := time.Now()

					var ob, eb bytes.Buffer
					c := exec.Command(cmd)
					c.Stdout = &ob
					c.Stderr = &eb

					if err := c.Run(); err != nil {
						fmt.Println("Watch Error: ", err)
						fmt.Println(ob.String(), eb.String())
					}

					elapsed := time.Since(start)
					log.Printf("Done running %q in %s\n", cmd, elapsed)
				}
			case err := <-watcher.Errors:
				log.Fatal(err)
			}
		}
	}()

	err = watcher.Add(dir)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Done setting up watch for " + dir)
	<-done
}

func parseConfig(i interface{}) {
	expand := os.ExpandEnv(*cpath)

	// For windows slashes
	expand, err := filepath.Abs(expand)
	if err != nil {
		panic(err)
	}

	file, err := os.Open(expand)
	if err != nil {
		panic(err)
	}

	// wrap reader before passing it to the json decoder for comment stripping
	r := JsonConfigReader.New(file)

	decoder := json.NewDecoder(r)
	err = decoder.Decode(i)
	if err != nil {
		panic(err)
	}
}
