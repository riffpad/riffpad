package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type recipient struct {
	Email string
	Name  string
	Date  string
}

func loadRecipients(path string) ([]recipient, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	idx := map[string]int{}
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	emailCol, ok := idx["email"]
	if !ok {
		return nil, fmt.Errorf("CSV needs an \"email\" column, got %v", header)
	}
	nameCol, nameOK := idx["name"]
	dateCol, dateOK := idx["date"]

	seen := map[string]bool{}
	var out []recipient
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		email := normalizeEmail(row[emailCol])
		if email == "" || !validEmail(email) || seen[email] {
			continue
		}
		seen[email] = true
		rec := recipient{Email: email}
		if nameOK && len(row) > nameCol {
			rec.Name = strings.TrimSpace(row[nameCol])
		}
		if dateOK && len(row) > dateCol {
			rec.Date = strings.TrimSpace(row[dateCol])
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Email < out[j].Email })
	return out, nil
}

func writeRecipientsCSV(path string, recs []recipient) error {
	var w io.Writer = os.Stdout
	var f *os.File
	if path != "-" {
		var err error
		f, err = os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"email", "name", "date"}); err != nil {
		return err
	}
	for _, r := range recs {
		if err := cw.Write([]string{r.Email, r.Name, r.Date}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
