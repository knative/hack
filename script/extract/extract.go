package extract

import (
	"io/fs"
	"os"
	"path"

	"knative.dev/hack"
)

const (
	DirPerm  = 0o750
	FilePerm = 0o640
)

// Printer is an interface for printing messages.
type Printer interface {
	Println(i ...interface{})
	Printf(format string, i ...interface{})
	PrintErr(i ...interface{})
	PrintErrf(format string, i ...interface{})
}

// Operation is the main extract object that can extract scripts.
type Operation struct {
	// ScriptName is the name of the script to extract.
	ScriptName string
	// Verbose will print more information.
	Verbose bool
}

// Extract will extract a script from the library to a temporary directory and
// provide the file path to it.
func (o Operation) Extract(prtr Printer) error {
	l := logger{o.Verbose, prtr}
	artifactsDir := os.Getenv("ARTIFACTS")
	if artifactsDir == "" {
		var err error
		if artifactsDir, err = os.MkdirTemp("", "knative.*"); err != nil {
			return wrapErr(err, ErrBug)
		}
	}
	hackRootDir := path.Join(artifactsDir, "hack-scripts")
	l.debugf("Extracting hack scripts to directory: %s", hackRootDir)
	libs, err := hack.Scripts.ReadDir(".")
	if err != nil {
		return wrapErr(err, ErrBug)
	}
	for _, lib := range libs {
		if err = copy(l, hack.Scripts, hackRootDir, lib.Name()); err != nil {
			return err
		}
	}
	return nil
}

func copy(l logger, files fs.ReadDirFS, destRoot, dir string) error {
	entries, err := files.ReadDir(dir)
	if err != nil {
		return wrapErr(err, ErrBug)
	}
	for _, entry := range entries {
		entryPath := path.Join(dir, entry.Name())
		if entry.IsDir() {
			if err = copy(l, files, destRoot, entryPath); err != nil {
				return err
			}
		} else {
			if err = copyFile(l, files, destRoot, entryPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(l logger, files fs.ReadDirFS, root string, filePath string) error {
	f, err := files.Open(filePath)
	if err != nil {
		return wrapErr(err, ErrBug)
	}
	defer func(f fs.File) {
		_ = f.Close()
	}(f)
	destPath := path.Join(root, filePath)
	destDir := path.Dir(destPath)
	if err = os.MkdirAll(destDir, DirPerm); err != nil {
		return wrapErr(err, ErrUnexpected)
	}
	
}
