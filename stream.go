package rbxdhist

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/robloxapi/rbxver"
)

type Stream []Token

func (s Stream) MarshalJSON() (b []byte, err error) {
	b = append(b, []byte(`[`)...)
	for i, token := range s {
		if i > 0 {
			b = append(b, ',')
		}
		bsub, err := token.MarshalJSON()
		if err != nil {
			return nil, err
		}
		b = append(b, bsub...)
	}
	b = append(b, ']')
	return b, nil
}

func (s *Stream) UnmarshalJSON(b []byte) error {
	type jStream []struct {
		Type    string
		Action  string
		Build   string
		GUID    string
		Time    time.Time
		Version rbxver.Version
		Value   string
	}
	var stream jStream
	if err := json.Unmarshal(b, &stream); err != nil {
		return err
	}
	for _, token := range stream {
		switch token.Type {
		case "Job":
			*s = append(*s, &Job{
				Action:  token.Action,
				Build:   token.Build,
				GUID:    token.GUID,
				Time:    token.Time,
				Version: token.Version,
			})
		case "Status":
			t := Status(token.Value)
			*s = append(*s, &t)
		case "Raw":
			t := Raw(token.Value)
			*s = append(*s, &t)
		}
	}
	return nil
}

type Token interface {
	token()
	json.Marshaler
	json.Unmarshaler
}

type Job struct {
	Action  string
	Build   string
	GUID    string
	Time    time.Time
	Version rbxver.Version
	GitHash string
}

func (Job) token() {}

func (j *Job) MarshalJSON() (b []byte, err error) {
	var buf bytes.Buffer
	var c []byte
	buf.WriteString(`{"Type":"Job","Action":`)
	c, _ = json.Marshal(j.Action)
	buf.Write(c)
	buf.WriteString(`,"Build":`)
	c, _ = json.Marshal(j.Build)
	buf.Write(c)
	buf.WriteString(`,"GUID":`)
	c, _ = json.Marshal(j.GUID)
	buf.Write(c)
	buf.WriteString(`,"Time":`)
	c, _ = j.Time.MarshalJSON()
	buf.Write(c)
	if j.Version != (rbxver.Version{}) {
		buf.WriteString(`,"Version":`)
		c, _ = j.Version.MarshalJSON()
		buf.Write(c)
	}
	if j.GitHash != "" {
		buf.WriteString(`,"GitHash":`)
		c, _ = json.Marshal(j.GitHash)
		buf.Write(c)
	}
	buf.WriteString(`}`)
	return buf.Bytes(), nil
}

func (j *Job) UnmarshalJSON(b []byte) error {
	type jJob struct {
		Type    string
		Action  string
		Build   string
		GUID    string
		Time    time.Time
		Version rbxver.Version
		GitHash string
	}
	job := jJob{}
	if err := json.Unmarshal(b, &job); err != nil {
		return err
	}
	if job.Type != "Job" {
		return nil
	}
	j.Action = job.Action
	j.Build = job.Build
	j.GUID = job.GUID
	j.Time = job.Time
	j.Version = job.Version
	j.GitHash = job.GitHash
	return nil
}

type Status string

func (Status) token() {}

func (s Status) MarshalJSON() (b []byte, err error) {
	var buf bytes.Buffer
	var c []byte
	buf.WriteString(`{"Type":"Status","Value":`)
	c, _ = json.Marshal(string(s))
	buf.Write(c)
	buf.WriteString(`}`)
	return buf.Bytes(), nil
}

func (s *Status) UnmarshalJSON(b []byte) error {
	type jStatus struct {
		Type  string
		Value string
	}
	status := jStatus{}
	if err := json.Unmarshal(b, &status); err != nil {
		return err
	}
	if status.Type != "Status" {
		return nil
	}
	*s = Status(status.Value)
	return nil
}

type Raw string

func (Raw) token() {}

func (r Raw) MarshalJSON() (b []byte, err error) {
	var buf bytes.Buffer
	var c []byte
	buf.WriteString(`{"Type":"Raw","Value":`)
	c, _ = json.Marshal(string(r))
	buf.Write(c)
	buf.WriteString(`}`)
	return buf.Bytes(), nil
}

func (r *Raw) UnmarshalJSON(b []byte) error {
	type jRaw struct {
		Type  string
		Value string
	}
	raw := jRaw{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if raw.Type != "Raw" {
		return nil
	}
	*r = Raw(raw.Value)
	return nil
}
