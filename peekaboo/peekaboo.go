package main

import (
    "fmt"
    "os"
    "os/exec"
    "regexp"
    "bytes"
    "log"
    "image"
	"image/jpeg"
    "strconv"
    "time"
    "path/filepath"
    "net/http"
    "github.com/gorilla/mux" 
    "encoding/json"
    "github.com/disintegration/imaging"
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
            log.Println("starting in dev mode")
            os.Setenv("goenv", "dev")
        }
    }
    router := mux.NewRouter().StrictSlash(true)
    router.HandleFunc("/{baby}/peekaboo", allPeekaboos).Methods("GET")
    router.HandleFunc("/peekaboo", addPeekaboo).Methods("POST")
    router.HandleFunc("/peekaboo", editPeekaboo).Methods("PUT", "OPTIONS")
    router.HandleFunc("/{baby}/peekaboo/{token}", deletePeekaboo).Methods("OPTIONS", "DELETE")
    router.HandleFunc("/peekaboo/{token}/thumb", createThumb).Methods("POST")
    log.Println("starting peekaboo server")
    log.Fatal(http.ListenAndServe(":6060", router))
}

func getAccessControlString() string {
    goEnv := os.Getenv("goenv")
    if goEnv == "dev" {
        return "http://localhost:3000"
    }
    return "http://photos.remarkabelle.us"
}

func getBucket() string {
    goEnv := os.Getenv("goenv")
    if goEnv == "dev" {
        return "peekaboos1"
    }
    return "peekaboos"
}

func allPeekaboos(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", getAccessControlString())
    w.Header().Set("Access-Control-Allow-Credentials", "true")

    log.Println("Getting all peeks..")
    dataStore := peekaboodata.NewDataStore()
    defer dataStore.CloseSession()

    vars := mux.Vars(r)
    baby := vars["baby"]
    
	allPeeks, err := dataStore.AllPeeks(baby)
    if err != nil {
		allPeeks = []peekaboodata.Peekaboo {} 
	}
	value, err := json.Marshal(allPeeks)	
    w.Write(value)
}

func deletePeekaboo(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", getAccessControlString())
    w.Header().Set("Access-Control-Allow-Credentials", "true")
    w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
    
    if r.Method == "OPTIONS" {
        return
    }

    vars := mux.Vars(r)
    baby := vars["baby"]
    
    var peek peekaboodata.Peekaboo
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&peek)
    if err != nil {
        log.Println(err.Error())
        w.WriteHeader(400)
        return
    }
    log.Println(peek.Name)
    log.Println(peek.Token)

    
    log.Println("Deleting peek..")
    // delete actual
    peekabooKey := fmt.Sprintf("%s/%s%s", baby, peek.Token, filepath.Ext(peek.Name))
    err = peekaboos3.DeleteFile(getBucket(), peekabooKey)
        if err != nil {
        log.Println(err.Error())
        w.WriteHeader(400)
        return
    }

    // // delete thumb
    peekabooThumbKey := fmt.Sprintf("%s/thumbs/%s.jpg", baby, peek.Token)
    err = peekaboos3.DeleteFile(getBucket(), peekabooThumbKey)
    if err != nil {
        log.Println(err.Error())
        w.WriteHeader(400)
        return
    }

    // delete mongo
    dataStore := peekaboodata.NewDataStore()
    defer dataStore.CloseSession()
    err = dataStore.DeletePeek(&peek)
    if err != nil {
        w.WriteHeader(500)
        return
    }
}

func editPeekaboo(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", getAccessControlString())
    w.Header().Set("Access-Control-Allow-Credentials", "true")
    w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")
    
    if r.Method == "OPTIONS" {
        return
    }

    log.Println("editing")
    var peek peekaboodata.Peekaboo
    log.Println("parsing peek body..")
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&peek)
    if err != nil {
        log.Println(err.Error())
        w.WriteHeader(400)
        return
    }
    log.Println("Updating peek..")
    dataStore := peekaboodata.NewDataStore()
    defer dataStore.CloseSession()
    oldPeek := dataStore.GetPeek(peek.Token)
    if oldPeek.Baby != peek.Baby {
        err = peekaboos3.CopyFile(
            getBucket(),
            fmt.Sprintf("%s/%s%s", oldPeek.Baby, oldPeek.Token, filepath.Ext(oldPeek.Name)),
            fmt.Sprintf("%s/%s%s", peek.Baby, oldPeek.Token, filepath.Ext(oldPeek.Name)),
        )
        if err != nil {
            w.WriteHeader(500)
            return
        }
        err = peekaboos3.CopyFile(
            getBucket(),
            fmt.Sprintf("%s/thumbs/%s.jpg", oldPeek.Baby, oldPeek.Token),
            fmt.Sprintf("%s/thumbs/%s.jpg", peek.Baby, oldPeek.Token),
        )
        if err != nil {
            w.WriteHeader(500)
            return
        }
    }
    err = dataStore.UpdatePeek(&peek)
    if err != nil {
        w.WriteHeader(500)
        return
    }
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

func generateThumbNail(inputPath string, outputPath string, isVideo bool, orientation int) error {
    if isVideo {
        cmd := exec.Command("ffmpeg", "-i", inputPath, "-vframes", "1", outputPath, "-y")
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        return cmd.Run()
    }

    src, err := imaging.Open(inputPath)
	if err != nil {
		return err
	}
    src = imaging.Resize(src, 825, 0, imaging.Lanczos)
    if orientation == 8 {
        src = imaging.Rotate90(src)
    } else if orientation == 3 {
        src = imaging.Rotate180(src)
    } else if orientation == 6 {
        src = imaging.Rotate270(src)
    }
    err = imaging.Save(src, outputPath)
    if err != nil {
        return err
    }
    file, err := os.Open(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	outfile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outfile.Close()

	if err := jpeg.Encode(outfile, img, nil); err != nil {
		return err
	}
    return nil
}

func getDate(inputPath string) time.Time {
    cmd, _ := exec.Command("ffprobe", "-print_format", 
        "flat", "-show_format", inputPath).CombinedOutput()
    // creation_time   : 2017-09-21 13:31:27
    re := regexp.MustCompile("creation_time\\s*:\\s*([\\d-]*)")
    found := re.FindSubmatch(cmd)
    if len(found) < 2 {
        return time.Now()
    }
    dateFound, _ := time.Parse("2006-01-02", string(found[1]))
    return dateFound
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
    err = peekaboos3.DownloadFile(getBucket(), downloadKeyBuffer.String(), "/tmp/tmps3object")
    if err != nil {
        w.WriteHeader(500)
        return
    }
    log.Println("Generating thumbnail..")
    err = generateThumbNail("/tmp/tmps3object", "/tmp/tmps3thumb.jpg", peek.IsVideo, peek.Orientation)
    if err != nil {
        w.WriteHeader(500)
        return
    }
    log.Println("Finding date taken..")
    dateTaken := getDate("/tmp/tmps3object")
    log.Println(dateTaken.String())
    var uploadKeyBuffer bytes.Buffer
    uploadKeyBuffer.WriteString(peek.Baby)
    uploadKeyBuffer.WriteString("/thumbs/")
    uploadKeyBuffer.WriteString(token)
    uploadKeyBuffer.WriteString(".jpg")
    log.Println("Uploading thumbnail..")
    err = peekaboos3.UploadFile(getBucket(), uploadKeyBuffer.String(), "/tmp/tmps3thumb.jpg")
    if err != nil {
        w.WriteHeader(500)
        return
    }
    peek.ThumbCreated = true
    peek.Token = token
    if peek.IsVideo {
        peek.Date = dateTaken
    }
    dataStore := peekaboodata.NewDataStore()
	defer dataStore.CloseSession()
    err = dataStore.UpdatePeek(&peek)
    if err != nil {
        w.WriteHeader(500)
        return
    }
}