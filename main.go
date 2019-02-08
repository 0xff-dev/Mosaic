package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"image"
	"image/draw"
	"image/jpeg"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type DB struct {
	mutex *sync.Mutex
	store map[string][3]float64
}

func main() {
	mux := http.NewServeMux()
	files := http.FileServer(http.Dir("public"))
	mux.Handle("/static/", http.StripPrefix("/static/", files))

	mux.HandleFunc("/upload", upload)
	mux.HandleFunc("/mosaic", mosaic)

	server := http.Server{
        Addr: ":8080",
		Handler: mux,
	}
	TILESDB = tilesDB()
	fmt.Println("MOSAIC server listen...")
	server.ListenAndServe()
}

func upload(writer http.ResponseWriter, req *http.Request) {
	t, _ := template.ParseFiles("upload.html")
	t.Execute(writer, nil)
}

func cut(original image.Image, db *DB, tileSize, x1, y1, x2, y2 int ) <- chan image.Image{
	c := make(chan image.Image)
	sp := image.Point{0, 0}
	go func() {
		newImage := image.NewNRGBA(image.Rect(x1, y1, x2, y2))
		for y := y1; y < y2; y = y + tileSize{
			for x := x1; x < x2; x = x + tileSize {
				r, g, b, _ := original.At(x, y).RGBA()
				color := [3]float64{float64(r), float64(g), float64(b)}
				filename := db.nearest(color)
				file, err := os.Open(filename)
				if err == nil {
					img, _, err := image.Decode(file)
					if err == nil {
						t := resize(img, tileSize)
						tile := t.SubImage(t.Bounds())
						tileBounds := image.Rect(x, y, x+tileSize, y+tileSize)
						draw.Draw(newImage, tileBounds, tile, sp, draw.Src)
					} else {
						fmt.Println("Error ", err)
					}
				} else {
					fmt.Println("Error ", err)
				}
				file.Close()
			}
		}
		c <- newImage.SubImage(newImage.Rect)
	}()
	return c
}

func combine(r image.Rectangle, c1, c2, c3, c4 <- chan image.Image) <- chan string{
	c := make(chan string)
	go func() {
		var wg sync.WaitGroup
		img := image.NewNRGBA(r)
		copy := func(dst draw.Image, r image.Rectangle, src image.Image, sp image.Point) {
			draw.Draw(dst, r, src, sp, draw.Src)
			wg.Done()
		}
		wg.Add(4)
		var s1, s2, s3, s4 image.Image
		var ok1, ok2, ok3, ok4 bool
		for {
			select {
			case s1, ok1 = <-c1:
				go copy(img, s1.Bounds(), s1, image.Point{r.Min.X, r.Min.Y})
			case s2, ok2 = <-c2:
				go copy(img, s2.Bounds(), s2, image.Point{r.Max.X / 2, r.Min.Y})
			case s3, ok3 = <-c3:
				go copy(img, s3.Bounds(), s3, image.Point{r.Min.X, r.Max.Y / 2})
			case s4, ok4 = <-c4:
				go copy(img, s4.Bounds(), s4, image.Point{r.Max.X / 2, r.Max.Y / 2})
			}
			if (ok1 && ok2 && ok3 && ok4) {
				break
			}
		}
		wg.Wait()
		buf2 := new(bytes.Buffer)
		jpeg.Encode(buf2, img, nil)
		c <- base64.StdEncoding.EncodeToString(buf2.Bytes())
	}()
	return c
}

func mosaic(writer http.ResponseWriter, req *http.Request) {
	tStart := time.Now()
	req.ParseMultipartForm(10485760)
	file, _, _ := req.FormFile("image")
	defer file.Close()
	tileSize, _ := strconv.Atoi(req.FormValue("tile_size"))
	original, _, _ := image.Decode(file)
	bounds := original.Bounds()
	db := cloneTilesDB()

	c1 := cut(original, &db, tileSize, bounds.Min.X, bounds.Min.Y,
		bounds.Max.X/2, bounds.Max.Y/2)
	c2 := cut(original, &db, tileSize, bounds.Max.X/2, bounds.Min.Y,
		bounds.Max.X, bounds.Max.Y/2)
	c3 := cut(original, &db, tileSize, bounds.Min.X, bounds.Max.Y/2,
		bounds.Max.X/2, bounds.Max.Y)
	c4 := cut(original, &db, tileSize, bounds.Max.X/2, bounds.Max.Y/2,
		bounds.Max.X, bounds.Max.Y)
	c := combine(bounds, c1, c2, c3, c4)

	// 马赛克成功后，两张图都返回, 用base64编码
	buf1 := new(bytes.Buffer)
	jpeg.Encode(buf1, original, nil)
	originalStr := base64.StdEncoding.EncodeToString(buf1.Bytes())

	tEnd := time.Now()
	images := map[string]string{
		"original": originalStr,
		"mosaic": <-c,
		"duration": fmt.Sprintf("%v ", tEnd.Sub(tStart)),
	}
	t, _ := template.ParseFiles("results.html")
	t.Execute(writer, images)
}
