package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strconv"
)

type Page struct {
	Title string
	Body []byte
}

func loadPage(title string) (*Page, error) {
	filename := title
	body, err := ioutil.ReadFile("../wasm/" + filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL)
	title := r.URL.Path[len("/"):]
	var p *Page
	p = &Page{Title: title}
	t, err := template.ParseFiles("../wasm/" + title)
	if err != nil {
		fmt.Println(err)
	}
	t.Execute(w, p)
}

func styleHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL)
	title := r.URL.Path[len("/"):]
	var p *Page
	p = &Page{Title: title}
	t, err := template.ParseFiles("/" + title)
	if err != nil {
		fmt.Println(err)
	}
	w.Header().Set("content-type", "text/css")
	t.Execute(w, p)
}

func scriptHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL)
	title := r.URL.Path[len("/"):]
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	w.Header().Set("content-type", "text/javascript")
	w.Header().Set("content-length", strconv.Itoa(len(p.Body)))
	n := len(p.Body)
	fmt.Fprint(w, string(p.Body[:n]))
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL)
	title := r.URL.Path[len("/"):]
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	w.Header().Set("content-type", "image/svg+xml")
	w.Header().Set("content-length", strconv.Itoa(len(p.Body)))
	n := len(p.Body)
	fmt.Fprint(w, string(p.Body[:n]))
}


func wasmHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL)
	title := r.URL.Path[len("/"):]
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	w.Header().Set("content-type", "application/wasm")
	w.Header().Set("content-length", strconv.Itoa(len(p.Body)))
	w.Header().Set("content-encoding", "gzip")
	n := len(p.Body)
	fmt.Fprint(w, string(p.Body[:n]))
}

func main() {
	http.HandleFunc("/wasm_exec.html", handler)
	http.HandleFunc("/wasm_exec.js", scriptHandler)
	http.HandleFunc("/test.wasm.gz", wasmHandler)
	http.HandleFunc("/logo.svg", imageHandler)

	http.ListenAndServe(":8080", nil)
}
