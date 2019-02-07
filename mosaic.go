package main

import (
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"math"
	"os"
	"sync"
)

var TILESDB map[string][3]float64

// 去平均色的函数
func averageColor(img image.Image) [3]float64 {
	bounds := img.Bounds()
	r, g, b := 0.0, 0.0, 0.0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x ++ {
			R, G, B, _ := img.At(x, y).RGBA()
			r, g, b = r+float64(R), g+float64(G), b+float64(B)
		}
	}
	totalPixels := float64(bounds.Max.X * bounds.Max.Y)
	return [3]float64{r/totalPixels, g/totalPixels, b/totalPixels}
}

func resize(img image.Image, newWidth int) image.NRGBA {
	bounds := img.Bounds()
	ratio := bounds.Dx() / newWidth
	out := image.NewNRGBA(image.Rect(bounds.Min.X/ratio, bounds.Min.X/ratio,
		bounds.Max.X/ratio, bounds.Max.Y/ratio))
	for y, j := bounds.Min.Y, bounds.Min.Y; y < bounds.Max.Y; y, j =y+ratio, j+1 {
		for x, i := bounds.Min.X, bounds.Min.X; x < bounds.Max.X; x, i = x+ratio, i+1 {
			r, g, b, a := img.At(x, y).RGBA()
			out.Set(i, j, color.NRGBA{uint8(r>>8), uint8(g>>8), uint8(b>>8),
				uint8(a>>8)})
		}
	}
	return *out
}

func tilesDB() map[string][3]float64{
	// 用准备好的文件?
	fmt.Println("Start tiles db ...")
	db := make(map[string][3]float64)
	files, _ := ioutil.ReadDir("tiles")
	for _, file := range files {
		name := "./tiles/" + file.Name()
		file, err := os.Open(name)
		if err == nil {
			img, _, err := image.Decode(file)
			if err == nil {
				db[name] = averageColor(img)
			} else {
				fmt.Println("Error in TILES: ", err, name)
			}
		} else {
			fmt.Println("Error can not open file ", name, err)
		}
		file.Close()
	}
	fmt.Println("Finish tiles db")
	return db
}

// 考虑从文件读取新建db量费时间，每次从主库clone, 这是一个方法
func cloneTilesDB() DB{
	db := make(map[string][3]float64)
	for k, v := range TILESDB {
		db[k] = v
	}
	return DB{
		mutex: &sync.Mutex{},
		store: db,
	}
}


func sq(n float64) float64 {
	return n * n
}

func distance(p1 [3]float64, p2 [3]float64) float64 {
	return math.Sqrt(sq(p1[0]-p2[0])+sq(p1[1]-p2[1])+sq(p1[2]-p2[2]))
}

func (db *DB) nearest(target [3]float64) string {
	var filename string = "./tiles/default.jpg"
	db.mutex.Lock()
	smallest := 1000000.0
	for k, v := range db.store {
		dist := distance(target, v)
		if dist < smallest {
			smallest = dist
			filename = k
		}
	}
	delete(db.store, filename)
	db.mutex.Unlock()
	return filename
}
