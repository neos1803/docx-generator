package main

import (
	"bufio"
	"doc-generator/m/templating"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Map English weekdays to Indonesian
var weekdays = map[string]string{
	"Sunday":    "Minggu",
	"Monday":    "Senin",
	"Tuesday":   "Selasa",
	"Wednesday": "Rabu",
	"Thursday":  "Kamis",
	"Friday":    "Jumat",
	"Saturday":  "Sabtu",
}

// Map English months to Indonesian
var months = map[string]string{
	"January":   "Januari",
	"February":  "Februari",
	"March":     "Maret",
	"April":     "April",
	"May":       "Mei",
	"June":      "Juni",
	"July":      "Juli",
	"August":    "Agustus",
	"September": "September",
	"October":   "Oktober",
	"November":  "November",
	"December":  "Desember",
}

type Author struct {
	Name      string `json:"name"`
	Username  string `json:"username"`
	ID        int    `json:"id"`
	State     string `json:"state"`
	AvatarURL string `json:"avatar_url"`
	WebURL    string `json:"web_url"`
}

type PushData struct {
	CommitCount int    `json:"commit_count"`
	CommitTo    string `json:"commit_to"`
	CommitTitle string `json:"commit_title"`
	Ref         string `json:"ref"`
}

type Issue struct {
	ID             int      `json:"id"`
	Title          string   `json:"title"`
	ProjectID      int      `json:"project_id"`
	ActionName     string   `json:"action_name"`
	TargetID       int      `json:"target_id"`
	TargetIID      int      `json:"target_iid"`
	TargetType     string   `json:"target_type"`
	AuthorID       int      `json:"author_id"`
	TargetTitle    string   `json:"target_title"`
	CreatedAt      string   `json:"created_at"`
	Author         Author   `json:"author"`
	AuthorUsername string   `json:"author_username"`
	Imported       bool     `json:"imported"`
	ImportedFrom   string   `json:"imported_from"`
	PushData       PushData `json:"push_data"`
}

type Gitlab struct {
	Key  string
	Host string
}

func main() {
	var start, end time.Time
	var branch, commitTo, title string
	var name, sort string

	// Read all provided data

	file, err := os.Open("init.txt")

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	inits := make(map[string]string)

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		data := strings.SplitN(line, "=", 2)

		if len(data) == 2 {
			key := data[0]
			value := data[1]
			inits[key] = value
		}
	}

	// Check for scanning errors
	if err := scanner.Err(); err != nil {
		log.Fatalf("error reading file: %s", err)
	}

	name = inits["name"]
	after := inits["after"]
	before := inits["before"]
	format := "2006-01-02"

	// Parse the user id
	id, err := strconv.Atoi(inits["userId"])

	if err != nil {
		log.Fatal(err)
	}

	userId := id

	if inits["sort"] == "" {
		sort = "asc"
	} else {
		sort = inits["sort"]
	}

	gitlab := Gitlab{Key: inits["key"], Host: inits["host"]}

	start, err = time.Parse(format, after)
	if err != nil {
		log.Fatal(err)
	}

	end, err = time.Parse(format, before)
	if err != nil {
		log.Fatal(err)
	}

	current := start.AddDate(0, 0, -1)

	// Split the date an process one by one
	for !current.After(end) {
		var paragraphs []templating.Paragraph
		interval := current.AddDate(0, 0, 2)
		target := current.AddDate(0, 0, 1)
		issues, err := gitlab.getAllGitContributions(userId, interval.Format(format), current.Format(format), sort)

		if err != nil {
			fmt.Println(err)
		}

		if len(issues) != 0 {
			// Define the dynamic title, text, and paragraphs
			for _, issue := range issues {
				if (PushData{}) != issue.PushData {
					branch = issue.PushData.Ref
					commitTo = issue.PushData.CommitTo
					if issue.PushData.CommitTitle == "" {
						if issue.TargetTitle != "" {
							title = issue.TargetTitle
						} else {
							title = "Tanpa Judul"
						}
					} else {
						title = issue.PushData.CommitTitle
					}
				} else {
					branch = "Tanpa Branch"
					commitTo = "Bukan Commit"
					title = "Tanpa Judul"
				}
				paragraphs = append(paragraphs, templating.Paragraph{Action: issue.ActionName, Branch: branch, Hash: commitTo, Job: title})
			}

			template := templating.Template{Folder: fmt.Sprintf("./%s", months[target.Month().String()]), FileName: fmt.Sprintf("Laporan-%s-%s.docx", name, target.Format(format)), Name: name, Date: fmt.Sprintf("%s, %02d %s %d", weekdays[target.Weekday().String()], target.Day(), months[target.Month().String()], target.Year()), Title: "Form Laporan Kerja", FontSize: 28, Paragraphs: paragraphs}

			err = template.GenerateDocx()

			if err != nil {
				log.Fatal(err)
			}
		}

		current = current.AddDate(0, 0, 1)
	}
}

func (g Gitlab) getAllGitContributions(id int, before, after, sort string) ([]Issue, error) {

	var issues []Issue
	page := 1

	for page != 0 {
		data, next, err := g.getUserEvents(id, before, after, sort, page)

		if err != nil {
			log.Fatal(err)
		}

		issues = append(issues, data...)

		page = next
	}

	return issues, nil
}

func (g Gitlab) getUserEvents(id int, before, after, sort string, page int) ([]Issue, int, error) {

	var issues []Issue
	var nextPage int

	// URL to send the HTTP request to
	url := fmt.Sprintf("%s/api/v4/users/%d/events?before=%s&after=%s&sort=%s&per_page=10&page=%d", g.Host, id, before, after, sort, page)

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, -1, err
	}

	// Add the header with the PRIVATE-TOKEN
	req.Header.Set("PRIVATE-TOKEN", g.Key)

	// Create a new HTTP client
	client := &http.Client{}

	// Send the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		return nil, -1, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, -1, err
	}

	err = json.Unmarshal(body, &issues)
	if err != nil {
		return nil, -1, err
	}

	nextPageHeader := resp.Header.Get("X-Next-Page")

	if nextPageHeader != "" {
		nextPage, err = strconv.Atoi(resp.Header.Get("X-Next-Page"))

		if err != nil {
			log.Fatal(err)
		}
	} else {
		nextPage = 0
	}

	return issues, nextPage, err
}
