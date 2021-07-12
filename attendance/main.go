package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
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

	if classId == -1 {
		log.Fatal("Error: must provide the class id")
	}

	readStudentList()
	readAttendanceList()
	importAttendanceFromCsv()
	postAttendances()
}

type Student struct {
	Id      int    `json:"id"`
	Name    string `json:"identifier"`
	ClassId int    `json:"student_class_id"`
	Gender  string `json:"gender"`
}

var studentByName = make(map[string]*Student)

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
	for i := range studentList.Students {
		s := &studentList.Students[i]
		studentByName[canonicalizeName(s.Name)] = s
		if s.ClassId == classId {
			fmt.Println(s.Name)
		}
	}
}

type Presence struct {
	StudentId int  `json:"id"`
	Present   bool `json:"attendant"`
}

type Attendance struct {
	Id         int        `json:"id"`
	CreatedBy  int        `json:"created_by_id"`
	ClassId    int        `json:"student_class_id"`
	NumMales   int        `json:"male"`
	NumFemales int        `json:"female"`
	Date       string     `json:"attendance_date"`
	Register   []Presence `json:"register"`
}

var attendanceByDate = make(map[string]*Attendance)

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
	for i := range attendanceList.Attendances {
		a := &attendanceList.Attendances[i]
		if a.ClassId == classId {
			a.NumMales = 0
			a.NumFemales = 0
			attendanceByDate[a.Date[0:10]] = a
		}
	}
	fmt.Printf("%v\n", attendanceList.Attendances[0])
}

func importAttendanceFromCsv() {
	fmt.Println("Reading attendance data from csv...")
	f, err := os.Open("./data/" + strconv.Itoa(classId) + ".csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		importAttendanceRecord(rec)
	}

	fmt.Println("The following students couldn't be found:")
	for stud := range studentsNotFound {
		fmt.Println(stud)
	}
}

var studentsNotFound = make(map[string]bool)

func canonicalizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "  ", " ")
	name = strings.ReplaceAll(name, "  ", " ")
	return name
}

func importAttendanceRecord(rec []string) {
	studName := rec[0]
	stud, ok := studentByName[canonicalizeName(studName)]
	if !ok {
		studentsNotFound[studName] = true
		return
	}

	date := rec[1]
	att, attExists := attendanceByDate[date]
	if !attExists {
		att = &Attendance{
			CreatedBy: 1, // admin
			ClassId:   classId,
			Date:      date,
		}
		attendanceByDate[date] = att
	}

	studPresent := true
	if rec[2] == "assente" {
		studPresent = false
	}

	if studPresent && stud.Gender == "m" {
		att.NumMales++
	}
	if studPresent && stud.Gender == "f" {
		att.NumFemales++
	}

	for i := range att.Register {
		p := &att.Register[i]
		if p.StudentId == stud.Id {
			p.Present = studPresent
			return
		}
	}
	att.Register = append(att.Register, Presence{
		StudentId: stud.Id,
		Present:   studPresent,
	})
}

func postAttendances() {
	fmt.Println("Posting/patching attendances...")
	for _, att := range attendanceByDate {
		body, err := json.Marshal(att)
		if err != nil {
			log.Fatal(err)
		}

		method := "POST"
		url := baseUrl + "/attendance"
		if att.Id > 0 {
			method = "PATCH"
			url += "/" + strconv.Itoa(att.Id)
		}
		req, err := http.NewRequest(method, url, bytes.NewReader(body))
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Add("Content-Type", "application/json")
		if auth != "" {
			req.Header.Add("Authorization", auth)
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
		if method == "POST" && resp.StatusCode != http.StatusCreated ||
			method == "PATCH" && resp.StatusCode != http.StatusOK {
			log.Fatalf("Unexpected response with code %d:\n%s", resp.StatusCode, body)
		}
	}
}
