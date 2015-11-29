package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"image"
	"image/color"
	"image/png"
)

const imageWidth = 2560
const imageHeight = 3360

const (
	UNKNOWN = iota
	NO_RAIN
	SPRINKLE
	RAIN
	DOWNPOUR
)
const SPRINKLE_MAX = 25
const DOWNPOUR_MIN = 53

type rainfallData struct {
	data       []byte
	created_at time.Time
}
type weatherBoolData struct {
	data       []bool
	created_at time.Time
}

func downloadData(t time.Time) []byte {
	var buffer bytes.Buffer
	var url string
	var file_name string
	t = t.UTC()
	t = t.Add(time.Duration(-(t.Minute() % 10)) * time.Minute) //10分おき に修正

	//Download
	buffer.WriteString("http://database.rish.kyoto-u.ac.jp/arch/jmadata/data/jma-radar/synthetic/original/")
	buffer.WriteString(t.Format("2006/01/02/"))
	buffer.WriteString("Z__C_RJTD_")
	buffer.WriteString(t.Format("20060102150400"))
	buffer.WriteString("_RDR_JMAGPV__grib2.tar")
	url = buffer.String()
	buffer.Reset()
	buffer.WriteString("Z__C_RJTD_")
	buffer.WriteString(t.Format("20060102150400"))
	buffer.WriteString("_RDR_JMAGPV_Ggis1km_Prr10lv_ANAL_grib2.bin")
	file_name = buffer.String()

	fmt.Println(url)
	res, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()
	reader := tar.NewReader(res.Body)

	var header *tar.Header
	var contents []byte = []byte("")
	for {
		header, err = reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalln(err)
		}
		buf := new(bytes.Buffer)
		if _, err = io.Copy(buf, reader); err != nil {
			log.Fatalln(err)
		}
		if header.Name != file_name {
			continue
		}
		contents = buf.Bytes()
		break
	}
	return contents
}
func decompress(compData *[]byte, maxV byte, res_size int) []byte {
	var data []byte = make([]byte, res_size)

	var p uint32 = 0
	var runLength uint32 = 0
	var runLength_digit uint8 = 0
	var LNGU byte = 255 - maxV
	for i := 0; i < len(*compData); i++ {
		val := (*compData)[i]
		//normal value
		if val <= maxV {
			runLength_digit = 0
			data[p] = val
			p++
			continue
		}
		//run length value
		val -= maxV + 1
		if runLength_digit == 0 {
			runLength = uint32(val)
		}
		runLength = uint32(val) * uint32(math.Pow(float64(LNGU), float64(runLength_digit)))
		val = data[p-1]

		lim := runLength + p
		for j := p; j < lim; j++ {
			data[j] = val
		}
		p = lim
		runLength_digit++
	}
	return data
}
func getData(t time.Time) rainfallData {

	res := downloadData(t)
	//Open
	var maxV byte = res[204]
	//var size uint32 = binary.BigEndian.Uint32(res[716:720])
	compData := res[721:(len(res) - 4)]

	data := rainfallData{data: decompress(&compData, maxV, imageWidth*imageHeight)}

	return data
}
func (rd rainfallData) createPng(filename string) {

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	m := image.NewGray(image.Rect(0, 0, imageWidth, imageHeight))
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			c := color.Gray{uint8(255 - rd.data[y*imageWidth+x])}
			m.SetGray(x, y, c)
		}
	}
	if err = png.Encode(f, m); err != nil {
		log.Fatalln(err)
	}
}
func (bd weatherBoolData) createPng(filename string) {

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	m := image.NewGray(image.Rect(0, 0, imageWidth, imageHeight))
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {

			c := color.Gray{255}
			if bd.data[y*imageWidth+x] == true {
				c = color.Gray{0}
			}
			m.SetGray(x, y, c)
		}
	}
	if err = png.Encode(f, m); err != nil {
		log.Fatalln(err)
	}
}
func (rd rainfallData) dump(filename string) {
	ioutil.WriteFile(filename, rd.data, os.ModePerm)
}
func (rd rainfallData) filterMatch(weather int) weatherBoolData {
	var data []bool = make([]bool, len(rd.data))
	for i, val := range rd.data {
		data[i] = false
		switch weather {
		case UNKNOWN:
			if val == 0 {
				data[i] = true
			}
		case NO_RAIN:
			if val == 1 {
				data[i] = true
			}
		case SPRINKLE:
			if val >= 2 && val < SPRINKLE_MAX {
				data[i] = true
			}
		case RAIN:
			if val >= 2 {
				data[i] = true
			}
		case DOWNPOUR:
			if val > DOWNPOUR_MIN {
				data[i] = true
			}
		}
	}
	return weatherBoolData{data: data}
}

func main() {
	data := getData(time.Now().Add(time.Duration(-60) * time.Minute))
	data.createPng("tmp/aaa.png")
	data.filterMatch(RAIN).createPng("tmp/rain.png")

}
