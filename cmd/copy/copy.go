// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	usage = `This program copies source file into the destination.
Usage:
      cp [source file] [destination]`
)

func main() {

	switch len(os.Args) {
	case 1:
		printAndExitWithValue("Missing source and destination files arguments", 1)
	case 2:
		printAndExitWithValue("Missing destination path", 2)
	case 3:
		if err := copyFile(os.Args[1], os.Args[2]); err != nil {
			fmt.Println(err.Error())
			os.Exit(3)
		}

		return
	default:
		printAndExitWithValue("CP require only two arguments. Found more!", 4)
	}
}

func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)
	if err != nil {
		return err
	}

	if sourceFileStat.IsDir() {
		return copyDir(src, dst)
	}

	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, input, sourceFileStat.Mode())
}

func copyDir(src, dst string) error {
	files, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := createDirectory(dst); err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() {
			if err := copyDir(filepath.Join(src, f.Name()), filepath.Join(dst, f.Name())); err != nil {
				return err
			}
		}
		if err := copyFile(filepath.Join(src, f.Name()), filepath.Join(dst, f.Name())); err != nil {
			return err
		}
	}

	return nil
}

func printAndExitWithValue(errMsg string, exitValue int) {
	if errMsg != "" {
		fmt.Println(errMsg)
	}
	fmt.Println(usage)
	os.Exit(exitValue)
}

func createDirectory(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// The dir does not exist so create it.
			return os.MkdirAll(path, fs.FileMode(0750))
		}

		return err
	}

	if !stat.IsDir() {
		return fmt.Errorf("%s already exists and it is not a directory", path)
	}

	return nil
}
