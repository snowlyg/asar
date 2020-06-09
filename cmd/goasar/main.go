package main

import (
	"flag"
	"fmt"
	"github.com/snowlyg/goasar"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func openAsar(file *os.File) *asar.Entry {
	root, err := asar.Decode(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "asar: %s\n", err)
		os.Exit(1)
	}
	return root
}

func openFile(file string) *os.File {
	openedFile, err := os.Open(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "asar: %s\n", err)
		os.Exit(1)
	}
	return openedFile
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func getPath(str string) string {
	return strings.Replace(str, "\\", "/", -1)
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [options] [command]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  l|list <archive>\n")
		fmt.Fprintf(os.Stderr, "    list contents of asar archive\n")
		fmt.Fprintf(os.Stderr, "  x|extract <archive> <dir>\n")
		fmt.Fprintf(os.Stderr, "    extract contents of asar archive to directory\n")
		fmt.Fprintf(os.Stderr, "  p|pack <archive> <dir>\n")
		fmt.Fprintf(os.Stderr, "    create asar archive from directory\n")
		fmt.Fprintf(os.Stderr, "    -e to encrypt \n")
		fmt.Fprintf(os.Stderr, "\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 2 {
		flag.Usage()
		os.Exit(3)
	}

	switch command := flag.Arg(0); command {
	case "l", "list":
		path := getPath(flag.Arg(1))
		file := openFile(path)
		defer file.Close()
		root := openAsar(file)
		root.Walk(func(path string, _ os.FileInfo, _ error) error {
			fmt.Println("/" + path)
			return nil
		})

	case "x", "extract":
		file := openFile(getPath(flag.Arg(1)))
		defer file.Close()
		root := openAsar(file)
		if flag.NArg() < 3 {
			flag.Usage()
			os.Exit(1)
		}

		target := getPath(flag.Arg(2))

		err := root.Walk(func(path string, info os.FileInfo, _ error) error {
			entry := info.Sys().(*asar.Entry)

			realPath := filepath.Join(target, path)
			if entry.Flags&asar.FlagDir != 0 {
				return os.MkdirAll(realPath, 0755)
			}
			if entry.Flags&asar.FlagUnpacked != 0 {
				return nil
			}

			perm := os.FileMode(0644)
			if entry.Flags&asar.FlagExecutable != 0 {
				perm |= 0111
			}

			f, err := os.OpenFile(realPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
			if err != nil {
				return err
			}

			_, err = entry.WriteTo(f)
			if err != nil {
				f.Close()
				return err
			}

			if err := f.Close(); err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "asar: %s\n", err)
			os.Exit(1)
		}

	case "p", "pack":

		if flag.NArg() < 4 {
			flag.Usage()
			os.Exit(1)
		}

		asarFilename := getPath(flag.Arg(2))
		asarArchive, err := os.Create(asarFilename)
		check(err)
		defer asarArchive.Close()

		if flag.Arg(3) == "-e" {
			asar.Encrypt = true
		}

		dir := getPath(flag.Arg(1))
		entries := asar.Builder{}
		isRootFile := false
		isRootDir := true
		err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				if info.Name() != dir {
					if strings.Count(path, "\\") > 1 {
						isRootDir = false
					}
					entries.AddDir(info.Name(), asar.FlagDir, isRootDir)
				}
			} else {
				if strings.Count(path, "\\") == 1 {
					isRootFile = true
				}

				b, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}
				entries.AddString(info.Name(), string(b), asar.FlagNone, isRootFile)
			}

			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't read: %s\nError was %s\n", dir, err)
		}

		if _, err := entries.Root().EncodeTo(asarArchive); err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't make: %s\nError was %s\n", asarFilename, err)
			os.Exit(1)
		}

	default:
		flag.Usage()
		os.Exit(1)
	}

}
