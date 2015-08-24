package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/bbhasiapacific/digitaldigest/session"
)

const (
	MultipartMaxMemory = 4 * 1024 * 1024        // 4MB
	totalDiskSpace     = 1 * 1024 * 1024 * 1024 // 1GB
)

func edit(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		getContent(w, r)
	case "POST":
		postContent(w, r)
	case "DELETE":
		deleteContent(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func getContent(w http.ResponseWriter, r *http.Request) {
	data := struct {
		UsedDiskSpacePercentage string
		Files                   []File
		IsLoggedIn              bool
	}{
		UsedDiskSpacePercentage: fmt.Sprintf("%.1f", float64(usedDiskSpace())/float64(totalDiskSpace)*100),
		Files: listFiles(),
		IsLoggedIn: session.GetEmail(r) != "",
	}

	templates.ExecuteTemplate(w, "edit", data)
}

func postContent(w http.ResponseWriter, r *http.Request) {
	if session.GetEmail(r) == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := r.ParseMultipartForm(MultipartMaxMemory); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	headers := r.MultipartForm.File["file"]
	if len(headers) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("no files were uploaded")
		return
	}

	file, err := headers[0].Open()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer file.Close()

	filepath := "content/" + headers[0].Filename
	output, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		os.Remove(filepath)
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer output.Close()

	if _, err = io.Copy(output, file); err != nil {
		os.Remove(filepath)
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	http.Redirect(w, r, "/edit", http.StatusFound)
}

func deleteContent(w http.ResponseWriter, r *http.Request) {
	if session.GetEmail(r) == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	file := r.FormValue("file")

	if err := os.Remove(file); err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("File " + file + " was not found!"))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func usedDiskSpace() int64 {
	fileinfos, err := ioutil.ReadDir("content")
	if err != nil {
		log.Fatal(err)
		return -1
	}

	var total int64
	for _, fileinfo := range fileinfos {
		total += fileinfo.Size()
	}

	return total
}

