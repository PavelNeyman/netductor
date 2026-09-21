package gha

import "testing"

func TestParseAndSkipUses(t *testing.T) {
	yml := []byte(`
name: ci
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: unit
        run: echo hello
`)
	w, err := Parse(yml)
	if err != nil {
		t.Fatal(err)
	}
	if w.Name != "ci" || len(w.Jobs["test"].Steps) != 2 {
		t.Fatalf("%+v", w)
	}
}
