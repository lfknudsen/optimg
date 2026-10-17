package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/hymkor/trash-go"
)

var rxNonPng = regexp.MustCompile(`(?i).*\.(?:jpg|jpeg)`)
var rxPNG = regexp.MustCompile(`(?i).*\.png`)

func debug(msg ...string) {
	if *DEBUG {
		fmt.Println(msg)
	}
}

func handle(dir string, wg *sync.WaitGroup) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Panic("Error reading directory:", err)
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			fmt.Println("Error opening file info:", err)
			continue
		}
		if info.Mode().IsRegular() {
			abs, err := filepath.Abs(filepath.Join(dir, info.Name()))
			if err != nil {
				fmt.Println("Error getting absolute filepath:", err)
				continue
			}
			// Start by converting to .png
			if rxNonPng.MatchString(info.Name()) {
				convert := fmt.Sprintf("/convert=%s",
					strings.TrimSuffix(abs, filepath.Ext(abs))+".png")
				cmd := exec.Command("i_view64", abs, convert)
				cmd.Dir = dir
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				debug("Conversion command:", cmd.String())
				err = cmd.Run()
				if err != nil {
					log.Println(err)
				} else {
					_ = trash.Throw(filepath.Join(dir, info.Name()))
				}
			}

			// Resize the image if it's 6k or larger
			f, err := os.OpenFile(abs, os.O_RDONLY, 0666)
			if err != nil {
				fmt.Println("Error opening image file for resizing:", err)
				continue
			}
			debug("Decoding image file:", abs)
			img, _, err := image.DecodeConfig(f)
			if err != nil {
				fmt.Println("Error decoding image info header:", err)
				_ = f.Close()
				continue
			}
			_ = f.Close()
			if img.Width > 6000 && img.Height > 4000 || img.Height > 6000 && img.Width > 4000 {
				abs, err := filepath.Abs(filepath.Join(dir, info.Name()))
				if err != nil {
					fmt.Println(err)
				} else {
					resizeArg := fmt.Sprintf("resize=(%d,%d)", img.Width/2, img.Height/2)
					cmd := exec.Command("i_view64", "/file="+abs, "/makecopy", resizeArg, "/resample", "/aspectratio")
					cmd.Dir = dir
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					debug("Resizing command:", cmd.String())
					err = cmd.Run()
					if err != nil {
						log.Println(err)
					}
				}
			}

			// Optimise the image
			cmd := exec.Command("optipng", "-preserve", "-o", strconv.Itoa(*level), abs)
			cmd.Dir = dir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			debug("OptiPNG command:", cmd.String())
			err = cmd.Run()
			if err != nil {
				log.Println(err)
			}
		} else if info.Mode().IsDir() {
			if wg != nil {
				wg.Go(func() { handle(filepath.Join(dir, info.Name()), wg) })
			} else {
				handle(filepath.Join(dir, info.Name()), wg)
			}
		}
	}
}

func printHelp() {
	flag.PrintDefaults()
}

var level = flag.Int("o", 2, "OptiPNG optimisation level (0-7)")
var DEBUG = flag.Bool("d", false, "Debug mode")

func main() {
	flag.Parse()
	if *level < 0 || *level > 7 {
		printHelp()
		os.Exit(1)
	}

	handle(".", nil)
	//wg := new(sync.WaitGroup)
	//wg.Go(func() { handle(".", level, wg) })
	//wg.Wait()
}
