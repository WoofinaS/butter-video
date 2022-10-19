package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"sync"

	butter "github.com/WoofinaS/butteraugli-go"
	flag "github.com/spf13/pflag"
)

var (
	source, dist                string
	width, height               int
	pnorm, luma                 float32
	api                         butter.API = butter.ApiCreate()
	pixfmt                      butter.PixelFormat
	threads                     int
	feeders, workers, consumers sync.WaitGroup
	totalScore                  float32
)

type job struct {
	src, dis *[]byte
}

func init() {
	flag.CommandLine.SortFlags = false
	flag.StringVarP(&source, "source", "s", "", "specify the source video")
	flag.StringVarP(&dist, "dist", "d", "", "specify the distored video")
	flag.Float32VarP(&pnorm, "pnorm", "p", 3.0, "specify the pnorm")
	flag.Float32VarP(&luma, "intensity-target", "l", 250, "specify the target display brightness in nits")
	flag.IntVarP(&threads, "threads", "t", runtime.NumCPU()/2, "specify the number of frame threads")
	flag.Parse()

	// Checks if files exist / are valid
	if _, err := os.Stat(source); err != nil {
		log.Fatal("Source file does not eixst")
	}
	if _, err := os.Stat(dist); err != nil {
		log.Fatal("dist file does not eixst")
	}

	// Checks if the pnrom value is valid
	if pnorm <= 0 || pnorm > 100 {
		log.Fatal("invalid pnorm value")
	}

	// Gets resolutions and checks if they are the same
	srcWidth, srcHeight, err := getVideoResolution(source)
	if err != nil {
		log.Fatal(err)
	}
	disWidth, disHeight, err := getVideoResolution(dist)
	if err != nil {
		log.Fatal(err)
	}
	if srcWidth != disWidth || srcHeight != disHeight {
		log.Fatal("source and dist resolution must match")
	}
	width, height = srcWidth, srcHeight

	// Checks if the intensity-target is valid
	if luma <= 0 || luma > 1000 {
		log.Fatal("invalid intensity-target value")
	}
	api.SetIntensityTarget(luma)

	// Sets the value for the pixel format
	pixfmt = butter.PixelFormat{
		NumChannels: butter.RGB,
		DataType:    butter.TYPE_UINT16,
		Endianness:  butter.LITTLE_ENDIAN,
		Align:       0,
	}
}

func main() {
	jobs := make(chan job, 3)
	scores := make(chan float32, 3)

	for x := 0; x < 1; x++ {
		feeders.Add(1)
		go feeder(jobs)
	}

	feeders.Wait()
	close(jobs)

	for x := 0; x < threads; x++ {
		workers.Add(1)
		go worker(jobs, scores)
	}

	workers.Wait()
	close(scores)

	for x := 0; x < 1; x++ {
		consumers.Add(1)
		go consumer(scores)
	}

	consumers.Wait()
	fmt.Println(totalScore)

}

func feeder(jobs chan job) {
	v1, err := getVideo(source)
	if err != nil {
		log.Fatal(err)
	}
	v2, err := getVideo(dist)
	if err != nil {
		log.Fatal(err)
	}

	for {
		v1buf, err1 := v1.getFrame()
		v2buf, err2 := v2.getFrame()
		if err1 == io.EOF && err2 == io.EOF {
			break
		}
		if err1 != nil || err2 != nil {
			log.Print(err1)
			log.Fatal(err2)
		}
		jobs <- job{v1buf, v2buf}
	}
	feeders.Done()
}

func worker(jobs chan job, scores chan float32) {
	for j := range jobs {
		score := compare(j.src, j.dis)
		scores <- score
	}
	workers.Done()
}

func compare(srcBytes, disBytes *[]byte) float32 {
	task := butter.ComputeTask{
		Width:     uint32(width),
		Height:    uint32(height),
		RefBytes:  *srcBytes,
		DisBytes:  *disBytes,
		RefPixFmt: pixfmt,
		DisPixFmt: pixfmt,
	}
	result, err := api.Compute_new(task)
	if err != nil {
		log.Fatal(err)
	}
	score := result.GetDistance(pnorm)
	result.Destroy()

	return score
}

func consumer(scores chan float32) {
	for c := range scores {
		totalScore += c
	}
	consumers.Done()
}
