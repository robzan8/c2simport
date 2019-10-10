package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	user, schema int
	token        string

	client http.Client
)

const postUrl = "https://fptt.dewco.io/api/reports/form_data/"

func main() {
	flag.IntVar(&user, "user", 1, "user id")
	flag.IntVar(&schema, "schema", 1, "schema id")
	flag.StringVar(&token, "token", "", "auth token")
	flag.Parse()
	args := flag.Args()
	log.SetFlags(0)

	if len(args) == 0 {
		log.Fatal(`No input files provided.
Usage: dewcoadd -user=1 -schema=1 table.csv`)
	}

	for _, fileName := range args {
		dewcoAddCsv(fileName)
	}
}

func dewcoAddCsv(fileName string) {
	f, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	head, err := r.Read()
	if err != nil {
		log.Fatal(err)
	}
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		dewcoAddRecord(head, rec)
	}
}

type Line struct {
	User      int               `json:"user"`
	Schema    int               `json:"schema"`
	Timelevel int               `json:"timelevel"`
	DateStart string            `json:"date_start"`
	DateEnd   string            `json:"date_end"`
	Submitted bool              `json:"submitted"`
	Data      map[string]string `json:"data"`
}

func dewcoAddRecord(head, rec []string) {
	line := Line{
		User:      user,
		Schema:    schema,
		Timelevel: 4,
		Submitted: false,
	}
	year := strings.TrimSpace(rec[1])
	month := strings.TrimSpace(rec[0])
	line.DateStart = strings.Join([]string{year, month, "1"}, "-")
	line.DateEnd = strings.Join([]string{year, month, lastDay[month]}, "-")
	line.Data = make(map[string]string)
	for i := 2; i < len(rec); i++ {
		if rec[i] != "" {
			line.Data[head[i]] = rec[i]
		}
	}
	body, err := json.Marshal(&line)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", postUrl, bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/json")
	if token != "" {
		req.Header.Add("Authorization", "Token "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		log.Fatalf("Unexpected response with code %d:\n%s", resp.StatusCode, body)
	}
}

var lastDay = map[string]string{ // last day of each month
	"1":  "31",
	"2":  "28",
	"3":  "31",
	"4":  "30",
	"5":  "31",
	"6":  "30",
	"7":  "31",
	"8":  "31",
	"9":  "30",
	"10": "31",
	"11": "30",
	"12": "31",
}
