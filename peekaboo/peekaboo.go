package main

import (
    "os"
    "os/exec"
    "bytes"
    "log"
    "strconv"
    "path/filepath"
    "image/jpeg"
    "net/http"
    "github.com/gorilla/mux" 
    "encoding/json"
    "github.com/nfnt/resize"
    "github.com/mukk88/peekaboo-server/peekaboodata"
    "github.com/mukk88/peekaboo-server/peekaboos3"
)

type tokenObj struct {
    Token string
}

func main() {
    argsWithoutProg := os.Args[1:]
    if len(argsWithoutProg) >= 1 {
        isDev, err := strconv.ParseBool(argsWithoutProg[0])
        if err != nil {
            log.Println("failed to parse command line arguments, exiting.")
            return
        }
        if isDev {
            os.Setenv("goenv", "dev")
        }
    }
    router := mux.NewRouter().StrictSlash(true)
    router.HandleFunc("/peekaboo", allPeekaboos).Methods("GET")
    router.HandleFunc("/peekaboo", addPeekaboo).Methods("POST")
    router.HandleFunc("/peekaboo/{token}/thumb", createThumb).Methods("POST")
    log.Println("starting peekaboo server")
    log.Fatal(http.ListenAndServe(":6060", router))
}

func getAccessControlString() string {
    goEnv := os.Getenv("goenv")
    if goEnv == "dev" {
        return "http://localhost:3000"
    }
    return "http://liv.remarkabelle.us"
}

func allPeekaboos(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", getAccessControlString())
    w.Header().Set("Access-Control-Allow-Credentials", "true")

    log.Println("Getting all peeks..")
    dataStore := peekaboodata.NewDataStore()
    defer dataStore.CloseSession()
    
	allPeeks, err := dataStore.AllPeeks()
    if err != nil {
		allPeeks = []peekaboodata.Peekaboo {} 
	}
	value, err := json.Marshal(allPeeks)	
    w.Write(value)
}

func addPeekaboo(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", getAccessControlString())
    w.Header().Set("Access-Control-Allow-Credentials", "true")

    var peek peekaboodata.Peekaboo
    log.Println("parsing peek body..")
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&peek)
    if err != nil {
        log.Println(err.Error())
        w.WriteHeader(400)
        return
    }
    log.Println("Creating peek..")
	dataStore := peekaboodata.NewDataStore()
	defer dataStore.CloseSession()
    token := peekaboodata.GenerateToken(9)
    peek.Token = token
    err = dataStore.InsertPeek(&peek)
	if err != nil {
        log.Println(err)
        w.WriteHeader(409)
		return
    }
    tokenAsObj := tokenObj{token}
    body, _ := json.Marshal(&tokenAsObj)
    w.Write(body)
}

func generateThumbNail(inputPath string, outputPath string, isVideo bool) error {
    if isVideo {
        cmd := exec.Command("ffmpeg", "-i", inputPath, "-vframes", "1", outputPath, "-y")
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        return cmd.Run()
    }
    file, err := os.Open(inputPath)
    defer file.Close()
    if err != nil {
        return err
    }

    img, err := jpeg.Decode(file)
    if err != nil {
        return err
    }
    m := resize.Thumbnail(425, 425, img, resize.Lanczos3)
    out, err := os.Create(outputPath)
    if err != nil {
        return err
    }
    defer out.Close()
    jpeg.Encode(out, m, nil)
    return nil
}

func createThumb(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", getAccessControlString())
    w.Header().Set("Access-Control-Allow-Credentials", "true")
    var peek peekaboodata.Peekaboo
    log.Println("parsing peek body..")
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&peek)
    if err != nil {
        log.Println(err.Error())
        w.WriteHeader(400)
        return
    }
    log.Println("Creating thumb..")
    vars := mux.Vars(r)
    token := vars["token"]
    log.Println(token)
    var downloadKeyBuffer bytes.Buffer
    downloadKeyBuffer.WriteString(peek.Baby)
    downloadKeyBuffer.WriteString("/")
    downloadKeyBuffer.WriteString(token)
    downloadKeyBuffer.WriteString(filepath.Ext(peek.Name))
    log.Println("Downloading file..")
    log.Println(downloadKeyBuffer.String())
    err = peekaboos3.DownloadFile("peekaboos", downloadKeyBuffer.String(), "/tmp/tmps3object")
    if err != nil {
        w.WriteHeader(500)
        return
    }
    log.Println("Generating thumbnail..")
    err = generateThumbNail("/tmp/tmps3object", "/tmp/tmps3thumb.jpg", peek.IsVideo)
    if err != nil {
        w.WriteHeader(500)
        return
    }
    var uploadKeyBuffer bytes.Buffer
    uploadKeyBuffer.WriteString(peek.Baby)
    uploadKeyBuffer.WriteString("/thumbs/")
    uploadKeyBuffer.WriteString(token)
    uploadKeyBuffer.WriteString(".jpg")
    log.Println("Uploading thumbnail..")
    err = peekaboos3.UploadFile("peekaboos", uploadKeyBuffer.String(), "/tmp/tmps3thumb.jpg")
    if err != nil {
        w.WriteHeader(500)
        return
    }
    peek.ThumbCreated = true
    peek.Token = token
    dataStore := peekaboodata.NewDataStore()
	defer dataStore.CloseSession()
    err = dataStore.UpdatePeek(&peek)
    if err != nil {
        w.WriteHeader(500)
        return
    }
}