package os

import (
	goos "os"
	"runtime"
	"testing"

	"github.com/erickweyunga/sintax/object"
)

func s(v string) *object.StringObj { return &object.StringObj{Value: v} }
func n(v float64) *object.NumberObj { return &object.NumberObj{Value: v} }

func TestReadWrite(t *testing.T) {
	path := "/tmp/sintax_test_os.txt"
	defer goos.Remove(path)

	_, err := Write([]object.Object{s(path), s("hello")})
	if err != nil {
		t.Fatal(err)
	}

	r, err := Read([]object.Object{s(path)})
	if err != nil {
		t.Fatal(err)
	}
	if r.(*object.StringObj).Value != "hello" {
		t.Fatalf("expected 'hello', got %q", r.Inspect())
	}
}

func TestExists(t *testing.T) {
	r, _ := Exists([]object.Object{s("/tmp")})
	if !r.(*object.BoolObj).Value {
		t.Fatal("/tmp should exist")
	}

	r, _ = Exists([]object.Object{s("/does_not_exist")})
	if r.(*object.BoolObj).Value {
		t.Fatal("should not exist")
	}
}

func TestRemove(t *testing.T) {
	path := "/tmp/sintax_test_remove.txt"
	goos.WriteFile(path, []byte("test"), 0644)

	_, err := Remove([]object.Object{s(path)})
	if err != nil {
		t.Fatal(err)
	}

	r, _ := Exists([]object.Object{s(path)})
	if r.(*object.BoolObj).Value {
		t.Fatal("file should be deleted")
	}
}

func TestListdir(t *testing.T) {
	dir := "/tmp/sintax_test_dir"
	goos.MkdirAll(dir, 0755)
	goos.WriteFile(dir+"/a.txt", []byte("a"), 0644)
	goos.WriteFile(dir+"/b.txt", []byte("b"), 0644)
	defer goos.RemoveAll(dir)

	r, err := Listdir([]object.Object{s(dir)})
	if err != nil {
		t.Fatal(err)
	}
	list := r.(*object.ListObj)
	if len(list.Elements) != 2 {
		t.Fatalf("expected 2 files, got %d", len(list.Elements))
	}
}

func TestGetenv(t *testing.T) {
	goos.Setenv("SINTAX_TEST", "hello")
	defer goos.Unsetenv("SINTAX_TEST")

	r, _ := Getenv([]object.Object{s("SINTAX_TEST")})
	if r.(*object.StringObj).Value != "hello" {
		t.Fatalf("expected 'hello', got %q", r.Inspect())
	}
}

func TestExec(t *testing.T) {
	r, err := Exec([]object.Object{s("echo hello")})
	if err != nil {
		t.Fatal(err)
	}
	if r.(*object.StringObj).Value != "hello" {
		t.Fatalf("expected 'hello', got %q", r.Inspect())
	}
}

func TestPlatform(t *testing.T) {
	r, _ := Platform(nil)
	if r.(*object.StringObj).Value != runtime.GOOS {
		t.Fatalf("expected %s", runtime.GOOS)
	}
}

func TestCwd(t *testing.T) {
	r, err := Cwd(nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.(*object.StringObj).Value == "" {
		t.Fatal("cwd should not be empty")
	}
}
