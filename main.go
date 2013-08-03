package main

import (
	"bytes"
	"log"
	"net/url"
	"os"
	"regexp"

	"github.com/gorilla/http"
)

const workers = 16

var match = regexp.MustCompile(`(https?:\/\/)?([\da-z\.-]+)\.([a-z\.]{2,6})([\/\w \.-]*)*\/?`)

func worker(in <-chan string, out chan<- string) {
	for {
		var b bytes.Buffer
		u := <-in
		if _, err := url.ParseRequestURI(u); err != nil {
			// dud url, ignore
			continue
		}
		if _, err := http.Get(&b, u); err != nil {
			log.Printf("error: %s: %v", u, err)
			continue
		}
		for _, u := range match.FindAll(b.Bytes(), -1) {
			out <- string(u)
		}
	}
}

func main() {
	in, out := make(chan string), make(chan string)
	for i := 0; i < workers; i++ {
		go worker(out, in)
	}
	var urls []string
	tmp := os.Args[1]
	o := out
	seen := map[string]struct{}{}
	for {
		select {
		case in := <-in:
			if _, ok := seen[in]; ok {
				continue
			}
			seen[in] = struct{}{}
			urls = append(urls, in)
			o = out // restore o
		case o <- tmp:
			if len(urls) == 0 {
				o = nil // block
				log.Println("urls empty, blocking send")
				continue
			}
			tmp = urls[0]
			urls[0], urls = urls[len(urls)-1], urls[:len(urls)-1]
		}
	}
}
