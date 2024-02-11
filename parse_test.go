package rbxdhist

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

type Counter map[string]struct {
	a, b time.Time
	n    int
}

func TestParse(t *testing.T) {
	_ = json.Marshal
	resp, err := http.Get("https://setup.rbxcdn.com/DeployHistory.txt")
	if err != nil {
		t.Errorf("error fetching deploy history %s", err)
	}
	dh, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Errorf("error reading deploy history %s", err)
	}
	s0 := Lex(dh)
	b, _ := json.MarshalIndent(s0, "", "\t")
	var s1 Stream
	json.Unmarshal(b, &s1)
	if len(s1) != len(s0) {
		t.Errorf("mismatched slice length")
	}
	for i, a := range s0 {
		b := s1[i]
		switch a := a.(type) {
		case *Job:
			b := b.(*Job)
			switch {
			case
				a.Action != b.Action,
				a.Build != b.Build,
				a.GUID != b.GUID,
				!a.Time.Equal(b.Time),
				a.Version != b.Version:
				goto fail
			}
			break
		fail:
			t.Errorf("mismatched item %d:\n%#v\n%#v\n", i, a, b)
		case *Status:
			b := b.(*Status)
			if *a != *b {
				t.Errorf("mismatched item %d:\n%#v\n%#v\n", i, a, b)
			}
		case *Raw:
			b := b.(*Raw)
			if *a != *b {
				t.Errorf("mismatched item %d:\n%#v\n%#v\n", i, a, b)
			}
		default:
			if a != b {
				t.Errorf("mismatched item %d:\n%#v\n%#v\n", i, a, b)
			}
		}
	}

	count := Counter{}
	for _, item := range s0 {
		switch item := item.(type) {
		case *Job:
			c, ok := count[item.Build]
			if !ok {
				c.a = item.Time
			}
			c.b = item.Time
			c.n++
			count[item.Build] = c
		}
	}

	for k, v := range count {
		t.Logf("%s\t%d\t%s\t%s\n", k, v.n, v.a, v.b)
	}

	// var buf bytes.Buffer
	// e := json.NewEncoder(&buf)
	// e.SetIndent("", "\t")
	// e.Encode(s0)
	// ioutil.WriteFile("DeployHistory.json", buf.Bytes(), 0666)
}
