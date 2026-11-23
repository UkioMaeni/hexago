package assets_test

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"hexago/ui/assets"
)

func TestReadFromSetFS(t *testing.T) {
	old := assets.FS()
	defer assets.SetFS(old)
	assets.SetFS(fstest.MapFS{"data/hello.txt": {Data: []byte("привет")}})

	got, err := assets.ReadFile("data/hello.txt")
	if err != nil || string(got) != "привет" {
		t.Fatalf("ReadFile: %q, %v", got, err)
	}
	if _, err := assets.Open("нет.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("нет файла: %v, ожидали fs.ErrNotExist", err)
	}
}
