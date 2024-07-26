package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"mp/doit/pkg/recipe"
	"os"
	"strings"
)

func Hello(input io.Reader) (io.Reader, error) {
	text, err := io.ReadAll(input)
	if err != err {
		return nil, err
	}
	var buf bytes.Buffer
	_, err = buf.WriteString("Hello, " + string(text) + "!")
	return &buf, err
}

func World(input io.Reader) (io.Reader, error) {
	text, err := io.ReadAll(input)
	if err != err {
		return nil, err
	}
	var buf bytes.Buffer
	_, err = buf.WriteString(`World says, "` + string(text) + `"`)
	return &buf, err
}

//go:embed hello_world.yaml
var helloWorldYaml string

func main() {
	stepFuncMap := map[string]recipe.StepFunc{
		"HelloFunc": Hello,
		"WorldFunc": World,
	}

	r, err := recipe.New(strings.NewReader(helloWorldYaml), stepFuncMap)
	check(err)

	output, err := r.Cook(strings.NewReader("Mike"))
	check(err)

	_, err = io.Copy(os.Stdout, output)
	check(err)

	fmt.Println("")
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
