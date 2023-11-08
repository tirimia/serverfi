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

// TODO: optional logger for server
// TODO: more logs
// TODO: see if we can bypass the issue of
// TODO: clean this mess
func main() {
	var inputs TemplateInputs
	var binaryName string
	flag.StringVar(&binaryName, "name", "serverfi", "Name of the server binary")
	flag.IntVar(&inputs.ServerPort, "port", 8080, "Port on which to serve files")
	flag.Parse()

	path := flag.Arg(0)
	if path == "" {
		// TODO: print help and exit cleanly
		log.Panic("first argument (path) is mandatory")
	}
	info, err := os.Stat(path)
	if err != nil {
		panic(err)
	}
	if !info.IsDir() {
		log.Panicf("Path %s is supposed to be a directory", path)
	}

	inputs.Path = info.Name()

	tmpFolder, err := ioutil.TempDir("", "serverfi-go-*")
	if err != nil {
		log.Fatalf("Error creating temporary directory for build: %v", err)
	}
	defer func() {
		os.RemoveAll(tmpFolder)
		log.Println("Removed temporary directory " + tmpFolder)
	}()
	log.Println("Using temporary directory " + tmpFolder + " for building")

	serverFile, err := ioutil.TempFile(filepath.Dir(path), "serverfile-*.go")
	if err != nil {
		panic(err)
	}
	defer func() {
		serverFile.Close()
		os.Remove(serverFile.Name())
		log.Println("Removed temporary server file " + serverFile.Name())
	}()

	templateData, err := staticFiles.ReadFile("static/server.go.tmpl")
	if err != nil {
		panic(err)
	}

	tmpl := template.Must(template.New("main").Parse(string(templateData)))
	err = tmpl.Execute(serverFile, inputs)
	if err != nil {
		panic(err)
	}

	goZip, err := staticFiles.ReadFile("static/go.zip")
	if err != nil {
		log.Fatal("could not read bundled go archive")
	}

	err = Unzip(goZip, tmpFolder+"/")
	if err != nil {
		log.Fatal("could not extract " + err.Error())
	}

	// Execute the binary
	cmd := exec.Command(tmpFolder+"/go/bin/go", "build", "-o", binaryName, "-pkgdir", tmpFolder+"/go/pkg", serverFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// We need to set GOOS GOARCH and HOME, lest the compiler freak out
	cmd.Env = []string{
		"CGO_ENABLED=0",
		"GOOS=" + runtime.GOOS,
		"GOARCH=" + runtime.GOARCH,
		"GOROOT=" + tmpFolder + "/go",
		"HOME=" + os.Getenv("HOME"),
	}
	log.Print(cmd.Env)
	log.Print(cmd.String())

	err = cmd.Run()
	if err != nil {
		log.Fatal("Error running compiler: ", err)
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
