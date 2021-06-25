package main

import (
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
		studentByName[strings.ToLower(strings.TrimSpace(s.Name))] = s
	}
	fmt.Printf("%v\n", studentList.Students[0:5])
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
			attendanceByDate[a.Date[0:10]] = a
		}
	}
	fmt.Printf("%v\n", attendanceList.Attendances[0])
}

func importAttendanceFromCsv() {
	fmt.Println("Reading attendance data form csv...")
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
}

func importAttendanceRecord(rec []string) {
	studName := rec[0]
	stud, ok := studentByName[strings.ToLower(strings.TrimSpace(studName))]
	if !ok {
		log.Fatalf("Can't find student %s", studName)
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

	if stud.Gender == "m" || stud.Gender == "M" {
		att.NumMales++
	}
	if stud.Gender == "f" || stud.Gender == "F" {
		att.NumFemales++
	}

	studPresent := true
	if rec[2] == "assente" {
		studPresent = false
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
