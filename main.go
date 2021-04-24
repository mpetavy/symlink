package main

import (
	"flag"
	"fmt"
	"github.com/mpetavy/common"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	SYMLINK = ".symlink"
)

var (
	backup          *string
	restore         *string
	output          *string
	createDirectory *bool
)

func init() {
	common.Init(false, "1.0.6", "", "", "2018", "backup tool for file symbolic links", "mpetavy", fmt.Sprintf("https://github.com/mpetavy/%s", common.Title()), common.APACHE, nil, nil, nil, run, 0)

	backup = flag.String("backup", "", "Directory to backup all symbolic links to '*.symlink' files")
	restore = flag.String("restore", "", "Directory to restore content of '*.symlink' files to symbolic links")
	output = flag.String("o", "", "Output directory symbolic links/files")
	createDirectory = flag.Bool("d", false, "Create missing symlink target as directory")
}

func backupSymbolicLink(symlinkFilename string) error {
	symlinkTarget, err := os.Readlink(symlinkFilename)
	if common.Error(err) {
		return err
	}

	common.Debug("target: %s", symlinkTarget)

	var filename string

	if *output != "" {
		filename = filepath.Join(*output, filepath.Base(symlinkFilename))
	} else {
		filename = symlinkFilename
	}

	if !common.FileExists(filepath.Dir(filename)) {
		common.Debug("create directory: %s", filepath.Dir(filename))

		err = os.MkdirAll(filepath.Dir(filename), common.DefaultDirMode)
		if common.Error(err) {
			return err
		}
	}

	filename += SYMLINK

	common.Debug("symlink: %s -> file: %s", symlinkFilename, filename)

	err = os.WriteFile(filename, []byte(symlinkTarget), common.DefaultFileMode)
	if common.Error(err) {
		return err
	}

	common.Info("backupSymbolicLink: symlink: %s; target: %s; file: %s", symlinkFilename, symlinkTarget, filename)

	return nil
}

func EvalString(b bool, st string, sf string) string {
	if b {
		return st
	} else {
		return sf
	}
}

func restoreSymbolicLink(filename string) error {
	bytes, err := os.ReadFile(filename)
	if common.Error(err) {
		return err
	}

	symlinkTarget := string(bytes)
	symlinkFilename := filename[:len(filename)-len(SYMLINK)]

	common.Debug("target: %s", symlinkTarget)

	if *output != "" {
		symlinkFilename = filepath.Join(*output, filepath.Base(symlinkFilename))
	}

	common.Debug("file: %s -> symlink: %s", filename, symlinkFilename)

	if !common.FileExists(filepath.Dir(symlinkFilename)) {
		common.Debug("create directory: %s", filepath.Dir(symlinkFilename))

		err = os.MkdirAll(filepath.Dir(symlinkFilename), common.DefaultDirMode)
		if common.Error(err) {
			return err
		}
	} else {
		if common.FileExists(symlinkFilename) {
			if common.IsSymbolicLink(symlinkFilename) {
				common.Debug("delete existing symlink: %s", symlinkFilename)

				err = os.Remove(symlinkFilename)
				if common.Error(err) {
					return err
				}
			} else {
				return fmt.Errorf("symlink file location already occupied: %s", symlinkFilename)
			}
		}
	}

	absPath := symlinkTarget

	if !filepath.IsAbs(absPath) {
		if *output != "" {
			absPath = filepath.Join(*output, filepath.Base(absPath))
		} else {
			cwd, err := os.Getwd()
			if common.Error(err) {
				return err
			}

			absPath = filepath.Join(*output, cwd)
		}
	}

	common.Debug("absolute path of target %s: %s", symlinkTarget, absPath)

	var isDirectory = false

	if common.FileExists(absPath) {
		isDirectory = common.IsDirectory(absPath)

		common.Debug("detect target %s as directory: %v", absPath, isDirectory)
	} else {
		if *createDirectory {
			common.Debug("create target as directory: %s", absPath)

			err = os.MkdirAll(absPath, common.DefaultDirMode)
			if common.Error(err) {
				return err
			}

			isDirectory = true
		} else {
			common.Debug("create target as file: %s", absPath)

			f, err := os.OpenFile(absPath, os.O_RDONLY|os.O_CREATE, common.DefaultFileMode)
			if common.Error(err) {
				return err
			}
			err = f.Close()
			if common.Error(err) {
				return err
			}
		}
	}

	common.Debug("link as %s", EvalString(isDirectory, "directory", "file"))

	if common.IsWindowsOS() && isDirectory {
		common.Debug("use Windows 'mklink'")

		cmd := exec.Command("cmd.exe", "/c", "mklink", "/d", symlinkFilename, symlinkTarget)
		cmd.Stderr = os.Stderr

		common.Debug("exec: %s", common.CmdToString(cmd))

		err := cmd.Run()
		if common.Error(err) {
			return err
		}
	} else {
		common.Debug("use GO symlink")

		err := os.Symlink(symlinkTarget, symlinkFilename)
		if common.Error(err) {
			return err
		}
	}

	if err == nil {
		common.Info("restoreSymbolicLink: file: %s; target: %s; symlink: %s", filename, symlinkTarget, symlinkFilename)
	}

	return err
}

func run() error {
	var path string

	if *backup != "" {
		path = *backup
	} else {
		path = *restore
	}

	if !filepath.IsAbs(path) {
		p, err := filepath.Abs(path)
		if common.Error(err) {
			return err
		}
		path = p
	}

	if !common.FileExists(path) {
		return &common.ErrFileNotFound{path}
	}

	isDirectory := common.IsDirectory(path)
	isSymbolicLink := common.IsSymbolicLink(path)

	if isDirectory && !isSymbolicLink {
		files, err := os.ReadDir(path)
		if common.Error(err) {
			return err
		}

		if len(files) == 0 {
			common.Warn("no symbolic links found in directory: %s", path)

			return nil
		}

		for _, file := range files {
			filename := filepath.Join(path, file.Name())

			if *backup != "" {
				isSymbolicLink := common.IsSymbolicLink(filename)

				if isSymbolicLink {
					err := backupSymbolicLink(filename)
					if common.Error(err) {
						return err
					}
				}
			} else {
				isFile := common.IsFile(filename)

				if isFile && strings.HasSuffix(filename, ".symlink") {
					err := restoreSymbolicLink(filename)
					if common.Error(err) {
						return err
					}
				}
			}
		}
	} else {
		if *backup != "" {
			isSymbolicLink := common.IsSymbolicLink(path)

			if isSymbolicLink {
				err := backupSymbolicLink(path)
				if common.Error(err) {
					return err
				}
			} else {
				return fmt.Errorf("not a symbolic symlink: %s", path)
			}
		} else {
			isFile := common.IsFile(path)

			if isFile && strings.HasSuffix(path, ".symlink") {
				err := restoreSymbolicLink(path)
				if common.Error(err) {
					return err
				}
			} else {
				return fmt.Errorf("not a symbolic symlink: %s", path)
			}
		}
	}

	return nil
}

func main() {
	defer common.Done()

	common.Run(nil)
}
