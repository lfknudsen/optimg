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
var rxImage = regexp.MustCompile(`(?i).*\.(png|jpg|jpeg)`)

func DEBUG(msg ...string) {
	if *debug {
		fmt.Println(msg)
	}
}

const iniFile = "[PNG]\n" +
	"SavePNGTransp=0\n" +
	"SavePNGAlpha=0\n" +
	"PngWndColor=1\n" +
	"PngOptim=0\n" +
	"PngOptimLevel=2\n" +
	"CompressionLevel=7\n" +
	"[Resize]\n" +
	"ResampleFilter=5\n" +
	"FastResampleDown=0\n" +
	"ResampleGamma=0\n" +
	"ResizeAspectRatio=1\n" +
	"UseResample=1\n" +
	"Sharpen=0\n" +
	"Mode=1\n" +
	"MegaPix=4.00\n" +
	"ResizeAutoDPI=0\n" +
	"DPI=300\n" +
	"Unit=0"

func generateIniFile() {
	err := os.WriteFile(irfanIniFile, []byte(iniFile), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func ExecutableDirectory() string {
	e, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Dir(e)
}

var executableDirectory = ExecutableDirectory()
var irfanIniFile = filepath.Join(executableDirectory, "i_view64.ini")

func collectFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Panic("Error reading directory:", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			fmt.Println("Error opening file info:", err)
			continue
		}
		if info.Mode().IsRegular() && rxImage.MatchString(info.Name()) {
			abs, err := filepath.Abs(filepath.Join(dir, info.Name()))
			if err != nil {
				fmt.Println("Error getting absolute filepath:", err)
				continue
			}
			files = append(files, abs)
			continue
		}
		if info.IsDir() {
			files = append(files, collectFiles(filepath.Join(dir, info.Name()))...)
		}
	}
	return files
}

func handle(dir string) {
	files := collectFiles(dir)
	var err error
	total := fmt.Sprint(len(files))
	wg := sync.WaitGroup{}
	for i, file := range files {
		wg.Go(func() {
			current := fmt.Sprint(i)
			prefix := strings.Repeat(" ", len(total)-len(current))
			fmt.Printf("%s%s/%s: Processing %s\n", prefix, current, total, file)

			// Start by converting to .png
			if rxNonPng.MatchString(file) {
				file, err = convertToPNG(dir, file)
			}
			if err != nil {
				log.Panic("Error converting file:", err)
			}

			// Resize the image if it's 6k or larger
			err = resize(dir, file)
			if err != nil {
				log.Panic("Error resizing file:", err)
			}

			// Optimise the image
			cmd := exec.Command("optipng", "-preserve", "-o", strconv.Itoa(*level), file)
			cmd.Dir = dir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Start()
			if err != nil {
				log.Panic("Error optimising file:", err)
			}
			err = cmd.Wait()
			if err != nil {
				log.Panic("Error optimising file:", err)
			}
		})
	}
	wg.Wait()
}

func resize(dir string, abs string) error {
	f, err := os.OpenFile(abs, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	DEBUG("Decoding image file:", abs)
	img, _, err := image.DecodeConfig(f)
	_ = f.Close()
	if err != nil {
		return err
	}
	if img.Width > *maxDimension && img.Height > *minDimension ||
		img.Height > *maxDimension && img.Width > *minDimension {
		newPercentage := 50
		if img.Width/2 > *maxDimension && img.Height/2 > *minDimension ||
			img.Height/2 > *maxDimension && img.Width/2 > *minDimension {
			newPercentage /= 2
		}
		cmd := exec.Command("i_view64", "/file="+abs,
			fmt.Sprintf("/resize=(%dp,%dp)", newPercentage, newPercentage),
			"/resample", "/aspectratio", "/cmdexit", "/ini="+executableDirectory,
			"/convert="+abs)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		DEBUG("Resizing command:", cmd.String())
		err = cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func convertToPNG(dir string, abs string) (string, error) {
	newName := strings.TrimSuffix(abs, filepath.Ext(abs)) + ".png"
	convert := fmt.Sprintf("/convert=%s", newName)

	cmd := exec.Command("i_view64", abs, convert)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	DEBUG("Conversion command:", cmd.String())
	err := cmd.Run()
	if err != nil {
		return newName, err
	}
	return newName, trash.Throw(abs)
}

func printHelp() {
	fmt.Println("Usage: optimg [Options]\n\nOptions:")
	flag.PrintDefaults()
}

var help = flag.Bool("h", false, "Print this help message and exit.")
var level = flag.Int("o", 2, "OptiPNG optimisation level (0-7)")
var debug = flag.Bool("d", false, "Verbose debug mode")
var minDimension = flag.Int("min", 4000, "If an image has width/height below this number, it will not be resized.")
var maxDimension = flag.Int("max", 6000, "If an image has a width/height above this number, it will be resized (unless the other dimension is below min).")
var regenIniFile = flag.Bool("new-ini", false, "Generate new ini file.")

func main() {
	flag.Parse()
	if *help {
		printHelp()
		os.Exit(0)
	}
	if *regenIniFile {
		generateIniFile()
	} else {
		_, err := os.Stat(irfanIniFile)
		if err != nil {
			generateIniFile()
		}
	}

	if *level < 0 || *level > 7 {
		_, _ = fmt.Fprintf(os.Stderr, "Optimisation level must be between 0-7 inclusive.\n")
		printHelp()
		os.Exit(1)
	}
	if *minDimension < 0 || *maxDimension < 0 || *minDimension >= *maxDimension {
		_, _ = fmt.Fprintf(os.Stderr,
			"Minimum and maximum dimension breakpoints must be positive,\n"+
				" and min must be smaller than max.\n")
		printHelp()
		os.Exit(1)
	}

	handle(".")
}
