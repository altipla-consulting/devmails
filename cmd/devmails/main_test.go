package main

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	require.NoError(t, run())
	compareFile(t, "hello-world")
	compareFile(t, "template-with-no-data")
}

func compareFile(t *testing.T, name string) {
	content, err := ioutil.ReadFile(filepath.Join(*outputFolder, name+".html"))
	require.NoError(t, err)

	compare, err := ioutil.ReadFile("/workspace/testdata/compare/" + name + ".html")
	require.NoError(t, err)

	require.Equal(t, content, compare)
}
