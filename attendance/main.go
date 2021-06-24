package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

var (
	classId int
	auth    string
	client  http.Client
)

const baseUrl = "https://cheese2school.gnucoop.io/api"

func main() {
	flag.IntVar(&classId, "class", -1, "Class ID")
	flag.StringVar(&auth, "auth", "", "Authorization header")
	flag.Parse()
	log.SetFlags(0)

	readStudentList()
	readAttendanceList()
}

type Student struct {
	Id      int    `json:"id"`
	Name    string `json:"identifier"`
	ClassId int    `json:"student_class_id"`
	Gender  string `json:"gender"`
}

var studentByName = make(map[string]Student)

func readStudentList() {
	fmt.Println("Reading student list...")
	var studentList struct {
		Students []Student `json:"results"`
	}

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
	for _, s := range studentList.Students {
		studentByName[strings.ToLower(strings.TrimSpace(s.Name))] = s
	}
	fmt.Printf("%v\n", studentList.Students[0:5])
}

type Presence struct {
	StudentId int  `json:"id"`
	Attendant bool `json:"attendant"`
}

type Attendance struct {
	Id         int        `json:"id"`
	ClassId    int        `json:"student_class_id"`
	NumMales   int        `json:"male"`
	NumFemales int        `json:"female"`
	Date       string     `json:"attendance_date"`
	Register   []Presence `json:"register"`
}

var attendanceByDate = make(map[string]Attendance)

func readAttendanceList() {
	fmt.Println("Reading attendance list...")
	var attendanceList struct {
		Attendances []Attendance `json:"results"`
	}

	req, err := http.NewRequest("GET", baseUrl+"/attendance", nil)
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
	err = json.Unmarshal(body, &attendanceList)
	if err != nil {
		log.Fatal(err)
	}
	for _, a := range attendanceList.Attendances {
		attendanceByDate[a.Date[0:10]] = a
	}
	fmt.Printf("%v\n", attendanceList.Attendances[0])
}
