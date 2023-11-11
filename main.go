package main

import (
	"archive/zip"
	"bytes"
	"embed"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

//go:embed static/*
var staticFiles embed.FS

type TemplateInputs struct {
	Path       string
	ServerPort int
}

type Options struct {
	GOOS       string
	GOARCH     string
	BinaryName string
}

// TODO: optional logger for server
// TODO: more logs
// TODO: see if we can bypass the issue of
// TODO: clean this mess
func main() {
	w := flag.CommandLine.Output()
	flag.Usage = func() {
		fmt.Fprintln(w, "Usage: serverfi [flags] <directory_to_serve>")
		flag.PrintDefaults()
	}

	var options Options
	flag.StringVar(&options.GOOS, "goos", runtime.GOOS, "GOOS for which we compile the server binary")
	flag.StringVar(&options.GOARCH, "goarch", runtime.GOARCH, "GOARCH for which we compile the server binary")
	flag.StringVar(&options.BinaryName, "name", "serverfi", "Name of the server binary")

	var inputs TemplateInputs
	flag.IntVar(&inputs.ServerPort, "port", 8080, "Port on which to serve files")
	flag.Parse()

	path := flag.Arg(0)
	if path == "" {
		fmt.Fprintln(w, "Missing path of folder to serve")
		fmt.Fprintln(w, "")
		flag.Usage()
		os.Exit(1)
	}
	dirInfo, err := os.Stat(path)
	if err != nil {
		log.Fatal("Could not stat target directory: " + err.Error())
	}
	if !dirInfo.IsDir() {
		log.Fatalf("Path %s is supposed to be a directory", path)
	}

	inputs.Path = dirInfo.Name()

	tmpFolder, err := ioutil.TempDir("", "serverfi-go-*")
	if err != nil {
		log.Printf("Error creating temporary directory for build: %v", err)
		return
	}
	defer func() {
		os.RemoveAll(tmpFolder)
		log.Println("Removed temporary directory " + tmpFolder)
	}()
	log.Println("Using temporary directory " + tmpFolder + " for building")

	serverFile, err := ioutil.TempFile(filepath.Dir(path), "serverfile-*.go")
	if err != nil {
		log.Println("Could not create temporary server file: " + err.Error())
		return
	}
	defer func() {
		serverFile.Close()
		os.Remove(serverFile.Name())
		log.Println("Removed temporary server file " + serverFile.Name())
	}()

	templateData, err := staticFiles.ReadFile("static/server.go.tmpl")
	if err != nil {
		log.Println("Could not read server file template: " + err.Error())
		return
	}

	tmpl := template.Must(template.New("main").Parse(string(templateData)))
	err = tmpl.Execute(serverFile, inputs)
	if err != nil {
		log.Println("Could not write templated server file: " + err.Error())
		return
	}

	goZip, err := staticFiles.ReadFile("static/go.zip")
	if err != nil {
		log.Println("Could not read bundled go archive: " + err.Error())
		return
	}

	err = Unzip(goZip, tmpFolder+"/")
	if err != nil {
		log.Println("Could not extract bundled go zip: " + err.Error())
		return
	}

	// Execute the binary
	cmd := exec.Command(tmpFolder+"/go/bin/go", "build", "-o", options.BinaryName, "-pkgdir", tmpFolder+"/go/pkg", serverFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// We need to set GOOS GOARCH and HOME, lest the compiler freak out
	cmd.Env = []string{
		"CGO_ENABLED=0",
		"GOOS=" + options.GOOS,
		"GOARCH=" + options.GOARCH,
		"GOROOT=" + tmpFolder + "/go",
		"HOME=" + os.Getenv("HOME"),
	}
	log.Print(cmd.Env)
	log.Print(cmd.String())

	err = cmd.Run()
	if err != nil {
		log.Println("Could not compile server: " + err.Error())
		return
	}
	log.Print("Built")
}

func Unzip(zipBytes []byte, dest string) error {
	reader := bytes.NewReader(zipBytes)
	r, err := zip.NewReader(reader, int64(reader.Len()))
	if err != nil {
		return err
	}

	os.MkdirAll(dest, 0o755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Join(dest, f.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, 0o744)
		} else {
			err = os.MkdirAll(filepath.Dir(path), 0o744)
			if err != nil {
				return fmt.Errorf("could not make directory %s : %v", path, err)
			}

			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o744) // f.Mode())
			if err != nil {
				return fmt.Errorf("could not open file %s : %v", path, err)
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}
