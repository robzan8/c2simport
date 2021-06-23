package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var (
	auth   string
	client http.Client
)

const baseUrl = "https://cheese2school.gnucoop.io/api"

func main() {
	flag.StringVar(&auth, "auth", "", "Authorization header")
	flag.Parse()
	log.SetFlags(0)

	readStudentList()
}

type Student struct {
	Id      int    `json:"id"`
	Name    string `json:"identifier"`
	ClassId int    `json:"student_class_id"`
	Gender  string `json:"gender"`
}

var studentList struct {
	Students []Student `json:"results"`
}

func readStudentList() {
	fmt.Println("Reading student list...")
	req, err := http.NewRequest("GET", baseUrl+"/student", nil)
	if err != nil {
		log.Fatal(err)
	}
	if auth != "" {
		req.Header.Add("Authorization", auth)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected response with code %d:\n%s", resp.StatusCode, body)
	}
	err = json.Unmarshal(body, &studentList)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v\n", studentList.Students[0:5])
}
