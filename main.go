package main

import (
	"fmt"
	"io"
	"log"
	"net/http"			
	"encoding/json"
	"mime/multipart"
)

const MAX_UPLOAD_SIZE = 1024 * 1024 * 100 // 100MB, note : this code only work to upload file below 150MB

const DROPBOX_DEFAULT_PATH = "/file-upload"		// path to save the file in dropbox -> Path will be : Dropbox/Application/project_name/file-upload

const DROPBOX_ACCESS_TOKEN = "<get access token>"

const DROPBOX_UPLOAD_URL = "https://content.dropboxapi.com/2/files/upload"

const DROPBOX_GET_FILE_URL = "https://api.dropboxapi.com/2/sharing/create_shared_link_with_settings"

// Progress is used to track the progress of a file upload.
// It implements the io.Writer interface so it can be passed
// to an io.TeeReader()
type Progress struct {
	TotalSize int64
	BytesRead int64
}

// Config is used to set the configuration in order to upload to dropbox
type Config struct {	
	Autorename     bool   `json:"autorename"`
	Mode           string `json:"mode"`
	Mute           bool   `json:"mute"`
	Path           string `json:"path"`
	StrictConflict bool   `json:"strict_conflict"`
}


// Write is used to satisfy the io.Writer interface.
// Instead of writing somewhere, it simply aggregates
// the total bytes on each read
func (pr *Progress) Write(p []byte) (n int, err error) {
	n, err = len(p), nil
	pr.BytesRead += int64(n)
	pr.Print()
	return
}

// Print displays the current progress of the file upload
func (pr *Progress) Print() {
	if pr.BytesRead == pr.TotalSize {
		fmt.Println("DONE!")
		return
	}

	fmt.Printf("File upload in progress: %d\n", pr.BytesRead)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	http.ServeFile(w, r, "index.html")
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 32 MB is the default used by FormFile
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// get a reference to the fileHeaders
	files := r.MultipartForm.File["file"]

	for _, fileHeader := range files {
		if fileHeader.Size > MAX_UPLOAD_SIZE {
			http.Error(w, fmt.Sprintf("The uploaded image is too big: %s. Please use an image less than 100MB in size", fileHeader.Filename), http.StatusBadRequest)
			return
		}		
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		defer file.Close()

		buff := make([]byte, 512)
		_, err = file.Read(buff)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		filetype := http.DetectContentType(buff)
		if filetype != "image/jpeg" && filetype != "image/png" {
			http.Error(w, "The provided file format is not allowed. Please upload a JPEG or PNG image", http.StatusBadRequest)
			return
		}

		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Println("Upload to Dropbox")
		uploadDropbox(file,fileHeader.Filename)
		
	}

	fmt.Fprintf(w, "Upload successful")
}

// upload file to dropbox using http request
func uploadDropbox(file multipart.File, fileName string){
	req, err := http.NewRequest("POST", DROPBOX_UPLOAD_URL, file)
	if err != nil {
		panic(err)		
	}	
	config := Config{false, "add", false, fmt.Sprintf("%s/%s", DROPBOX_DEFAULT_PATH, fileName), false}
	jsonConfig, err := json.Marshal(config)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s",DROPBOX_ACCESS_TOKEN))
	req.Header.Set("Dropbox-Api-Arg", string(jsonConfig))
	req.Header.Set("Content-Type", "application/octet-stream")	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}	
	defer resp.Body.Close()
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", IndexHandler)
	mux.HandleFunc("/upload", uploadHandler)

	if err := http.ListenAndServe(":4500", mux); err != nil {
		log.Fatal(err)
	}
}
