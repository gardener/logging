// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
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
		if err := copy(os.Args[1], os.Args[2]); err != nil {
			fmt.Println(err.Error())
			os.Exit(3)
		}
		return
	default:
		printAndExitWithValue("CP require only two arguments. Found more!", 4)
	}
}

func copy(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if sourceFileStat.IsDir() {
		return copyDir(src, dst)
	}

	input, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(dst, input, sourceFileStat.Mode())
}

func copyDir(src, dst string) error {
	files, err := ioutil.ReadDir(src)
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
		if err := copy(filepath.Join(src, f.Name()), filepath.Join(dst, f.Name())); err != nil {
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
			return os.MkdirAll(path, os.ModePerm)
		}
		return err
	}

	if !stat.IsDir() {
		return fmt.Errorf("%s already exists and it is not a directory", path)
	}

	return nil
}
