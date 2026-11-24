package bioconvert

import (
	"reflect"
	"strings"
	"testing"
)

func TestLiteralParser(t *testing.T) {
	src := "from typing import Mapping\n\ncatalog: dict[str, Mapping] = {\n  # comment\n  'a': [1, -2.5, ('B', 'IV'), True, None],\n  \"b\": 'it\\'s' ' joined',\n}\n"
	v, err := Assignment(src, "catalog")
	if err != nil {
		t.Fatal(err)
	}
	d := v.(*Dict)
	if !reflect.DeepEqual(d.Keys, []string{"a", "b"}) {
		t.Fatal(d.Keys)
	}
	if got := d.Values["a"]; !reflect.DeepEqual(got, []any{int64(1), -2.5, Tuple{"B", "IV"}, true, nil}) {
		t.Fatalf("%#v", got)
	}
	if d.Values["b"] != "it's joined" {
		t.Fatalf("%q", d.Values["b"])
	}
}

// Only literal data is accepted: any expression, call or name is refused,
// so nothing upstream can be executed or evaluated.
func TestLiteralParserRejectsCode(t *testing.T) {
	for _, src := range []string{
		"data = {'a': __import__('os').system('true')}",
		"data = [x for x in range(3)]",
		"data = {'a': other_dict}",
		"data = {'a': 1, 'a': 2}",
		"data = {'a': 1",
		"data = {'a': 1} | extra\n",
		"data = {'a': 1}\ndata.update({'b': 2})\n",
		"data = {'a': 1}\ndata = {}\n",
		"data = " + strings.Repeat("[", 100) + strings.Repeat("]", 100),
	} {
		if _, err := Assignment(src, "data"); err == nil {
			t.Errorf("accepted %q", src)
		}
	}
	if _, err := Assignment("x = 1\n", "data"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Error("missing assignment not reported")
	}
}
