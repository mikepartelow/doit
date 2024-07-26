package main

import (
	_ "embed"
	"fmt"
	"io"
	"mp/doit/pkg/recipe"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// MkdirAndSucceedIfItDidNotAlreadyExist makes a directory if it doesn't already exist, and returns error if it did already exist.
// The directory name is read from input.
func MkdirAndSucceedIfItDidNotAlreadyExist(input io.Reader) (io.Reader, error) {
	fmt.Println("+ MkdirAndSucceedIfItDidNotAlreadyExist")
	text, err := io.ReadAll(input)
	if err != err {
		return nil, err
	}

	directory := string(text)

	info, err := os.Stat(directory)
	if err != err {
		return nil, err
	}
	if info != nil && info.IsDir() {
		return nil, fmt.Errorf("ERROR: directory %q already exists", directory)
	}

	err = os.MkdirAll(directory, 0700)
	if err != err {
		return nil, err
	}

	return strings.NewReader(directory), err
}

// CreateFileInDirAndSucceedTheThirdTime creates a temp file in the directory named in input.
// If the number of files in the directory is less than 3, it returns an error, otherwise, it returns a CSV list of directory name and the three files.
func CreateFileInDirAndSucceedTheThirdTime(input io.Reader) (io.Reader, error) {
	fmt.Println("+ CreateFileInDirAndSucceedTheThirdTime")
	text, err := io.ReadAll(input)
	if err != err {
		return nil, err
	}

	directory := string(text)

	file, err := os.CreateTemp(directory, "spam")
	if err != err {
		return nil, err
	}
	file.Close()

	entries, err := os.ReadDir(directory)
	if err != err {
		return nil, err
	}

	if len(entries) < 3 {
		return nil, fmt.Errorf("Not succeeding until we've written 3 files. Currently %d.", len(entries))
	}
	info := directory + "," + entries[0].Name() + "," + entries[1].Name() + "," + entries[2].Name()
	return strings.NewReader(info), nil
}

// Tada reads 4 CSV values from input and writes a custom message to its output.
func Tada(input io.Reader) (io.Reader, error) {
	fmt.Println("+ Tada")
	text, err := io.ReadAll(input)
	if err != err {
		return nil, err
	}
	parts := strings.Split(string(text), ",")

	tada := fmt.Sprintf("Tada! Wrote [%q,%q,%q] to %q", parts[1], parts[2], parts[3], parts[0])

	return strings.NewReader(tada), err
}

//go:embed third_times_the_charm.yaml
var thirdTimesTheCharmYaml string

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage:", os.Args[0], "directory")
		os.Exit(1)
	}

	directory := os.Args[1]
	stateFilename := directory + ".doit-state"

	// map names found in the recipe YAML to functions in this file
	stepFuncMap := map[string]recipe.StepFunc{
		"MkdirAndSucceedIfItDidNotAlreadyExist": MkdirAndSucceedIfItDidNotAlreadyExist,
		"CreateFileInDirAndSucceedTheThirdTime": CreateFileInDirAndSucceedTheThirdTime,
		"Tada":                                  Tada,
	}

	// if state exists for this program, read it and pass it to recipe.New()
	var opts []recipe.Option
	_, err := os.Stat(stateFilename)
	if err == nil {
		state := mustReadState(stateFilename)
		opts = append(opts, recipe.WithState(state))
	}

	// construct a new Recipe
	r, err := recipe.New(strings.NewReader(thirdTimesTheCharmYaml), stepFuncMap, opts...)
	check(err)

	// pass the directory as input to the Recipe pipeline
	output, err := r.Cook(strings.NewReader(directory))
	if err != nil {
		fmt.Println("ERROR: ", err.Error())

		// if there was an error, save the state
		state := r.State()
		mustWriteState(state, stateFilename)
		os.Exit(1)
	}

	_, err = io.Copy(os.Stdout, output)
	check(err)

	fmt.Println("")
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func mustReadState(filename string) *recipe.State {
	var state recipe.State
	file, err := os.Open(filename)
	check(err)
	err = yaml.NewDecoder(file).Decode(&state)
	check(err)
	return &state
}

func mustWriteState(state *recipe.State, filename string) {
	file, err := os.Create(filename)
	check(err)
	err = yaml.NewEncoder(file).Encode(state)
	check(err)
}
