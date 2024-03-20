package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"golang.org/x/net/html"
)

type RequestData struct {
    Parent Parent `json:"parent"`
    Properties map[string]interface{} `json:"properties"`
}

type Parent struct {
    Type string `json:"type"`
    DatabaseID string `json:"database_id"`
}

func main() {
    var save bool
    var manual bool

    app := &cli.App{
        Name: "apptrack",
        UseShortOptionHandling: true,
        Flags: []cli.Flag{
            &cli.BoolFlag{
                Name:        "save",
                Aliases:     []string{"s"},
                Value:       false,
                Usage:       "save job post to apply later",
                Destination: &save,
            },
            &cli.BoolFlag{
                Name:       "manual",
                Aliases:    []string{"m"},
                Value:      false,
                Usage:      "fill in attributes manually",
                Destination:&manual,
            },
        },
        Action: func(cCtx *cli.Context) error {
            requestData := RequestData{}
            requestData.Properties = make(map[string]interface{})
            if !save {
                requestData.Properties["Date Applied"] = map[string]interface{}{
                    "type": "date",
                    "date": map[string]string {
                        "start": time.Now().Format("2006-01-02"), 
                    },
                }
                requestData.Properties["Status"] = map[string]interface{}{
                    "type": "status",
                    "status": map[string]string {
                        "name": "Applied",
                    },
                }
            } else {
                requestData.Properties["Status"] = map[string]interface{}{
                    "type": "status",
                    "status": map[string]string {
                        "name": "Ready to apply",
                    },
                }

            }

            if manual {
                manualInput(&requestData) 
            } else {
                scrapeLink(&requestData)
            } 

            notionRequest(requestData)

            return nil
        },
    }

    if err := app.Run(os.Args); err != nil {
        log.Fatal(err)
    }
}

func getInput(property string) string {
    var input string
    reader := bufio.NewReader(os.Stdin)
    for {
        fmt.Print("Enter " + property + ": ")
        input, _ = reader.ReadString('\n')
        if input != "\n" {
            break
        }
        fmt.Println("Input can't be empty!")
    }
    return input
}

func manualInput(requestData *RequestData) {    
    if _, ok := requestData.Properties["Company"]; !ok {
        input := getInput("company")
        requestData.Properties["Company"] = map[string]interface{}{
            "type": "title",
            "title": []map[string]interface{}{
                {
                    "type": "text",
                    "text": map[string]string{
                        "content": strings.TrimSpace(input),
                    },
                },
            },
        }
    }

    if _, ok := requestData.Properties["Position"]; !ok {
        input := getInput("position")
        requestData.Properties["Position"] = map[string]interface{}{
            "rich_text": []map[string]interface{}{
                {
                    "type": "text",
                    "text": map[string]string{
                        "content": strings.TrimSpace(input),
                    },
                },
            },
        }
    }

    if _, ok := requestData.Properties["Location"]; !ok {
        input := getInput("location")
        requestData.Properties["Location"] = map[string]interface{}{
            "rich_text": []map[string]interface{}{
                {
                    "type": "text",
                    "text": map[string]string{
                        "content": strings.TrimSpace(input),
                    },
                },
            },
        }
    }

    if _, ok := requestData.Properties["Link"]; !ok {
        input := getInput("link")
        requestData.Properties["Link"] = map[string]interface{}{
            "type": "url",
            "url": strings.TrimSpace(input),
        }
    }    
}

func scrapeLink(requestData *RequestData) {
    link := getInput("link")
    jobId := getJobId(link)

    jobLink := "https://www.linkedin.com/jobs-guest/jobs/api/jobPosting/" + jobId
    requestData.Properties["Link"] = map[string]interface{}{
        "type": "url",
        "url": "https://www.linkedin.com/jobs/view/" + jobId,
    }


    attempts, missing := 0, true
    for attempts < 3 && missing {
        attempts++
        reader, err := getContent(jobLink)
        if err != nil {
            log.Println("Error getting page contents", err)
            continue
        }
        missing = false
        err = getAttributes(reader, requestData)
        if err != nil {
            log.Println("Error getting attributes", err)
        }
        if _, ok := requestData.Properties["Company"]; !ok {
            missing = true
        } else if _, ok := requestData.Properties["Position"]; !ok {
            missing = true
        } else if _, ok := requestData.Properties["Location"]; !ok {
            missing = true
        }
    }

    if missing {
        manualInput(requestData)
    }
}

func getJobId(link string) string {
    re := regexp.MustCompile(`\d{9,}`)
    return string(re.Find([]byte(link)))
}

func getContent(link string) (io.Reader, error) {
    res, err := http.Get(link)
    if err != nil {
        log.Println("Error getting website contents", err)
        return nil, err
    }

    defer res.Body.Close()
    body, err := io.ReadAll(res.Body)
    if err != nil {
        log.Println("Error reading body", err)
        return nil, err
    }
    return bytes.NewReader(body), nil
}

func getAttributes(reader io.Reader, requestData *RequestData) error {
    nodeTree, err := html.Parse(reader)
    if err != nil {
        log.Println("Error parsing html file", err)
        return err
    }

    var parse func(*html.Node, *RequestData)
    parse = func(n *html.Node, requestData *RequestData) {
        if n.Type == html.ElementNode {
            for _, a := range n.Attr {
                if a.Key == "class" {
                    if strings.Contains(a.Val, "topcard__org-name-link") {
                        requestData.Properties["Company"] = map[string]interface{}{
                            "type": "title",
                            "title": []map[string]interface{}{
                                {
                                    "type": "text",
                                    "text": map[string]string{
                                        "content": strings.TrimSpace(n.FirstChild.Data),
                                    },
                                },
                            },
                        }


                    } else if strings.Contains(a.Val, "topcard__title") {
                        requestData.Properties["Position"] = map[string]interface{}{
                            "rich_text": []map[string]interface{}{
                                {
                                    "type": "text",
                                    "text": map[string]string{
                                        "content": strings.TrimSpace(n.FirstChild.Data),
                                    },
                                },
                            },
                        }
                    } else if strings.Contains(a.Val, "topcard__flavor topcard__flavor--bullet") {
                        requestData.Properties["Location"] = map[string]interface{}{
                            "rich_text": []map[string]interface{}{
                                {
                                    "type": "text",
                                    "text": map[string]string{
                                        "content": strings.TrimSpace(n.FirstChild.Data),
                                    },
                                },
                            },
                        }
                    }
                }
            }
        }
        for child := n.FirstChild; child != nil; child = child.NextSibling {
            parse(child, requestData)
        }
    }

    parse(nodeTree, requestData)

    return nil
}

func notionRequest(requestData RequestData) {
    apiKey := os.Getenv("APPTRACK_NOTION_API_KEY")
    databaseID := os.Getenv("APPTRACK_NOTION_DATABASE_ID")

    apiUrl := "https://api.notion.com/v1/pages"

    requestData.Parent = Parent {
        Type:       "database_id",
        DatabaseID: databaseID,
    }

    jsonData, err := json.Marshal(requestData)
    if err != nil {
        fmt.Println("Error marshalling JSON", err)
        return
    }

    req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonData))
    if err != nil {
        fmt.Println("Error creating HTTP request:", err)
        return
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+apiKey)
    req.Header.Set("Notion-Version", "2022-06-28")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        fmt.Println("Error sending HTTP request:", err)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        fmt.Println("Error:", resp.Status)
        body, err := io.ReadAll(resp.Body)
        if err != nil {
            fmt.Println("Error reading response body:", err)
            return
        }

        var responseBody map[string]interface{}
        if err := json.Unmarshal(body, &responseBody); err != nil {
            fmt.Println("Error parsing JSON:", err)
            return
        }

        if code, ok := responseBody["code"].(string); ok {
            fmt.Println("Code property:", code)
        } else {
            fmt.Println("Code property not found in response body")
        }

        if message, ok := responseBody["message"].(string); ok {
            fmt.Println("Message property:", message)
        } else {
            fmt.Println("Message property not found in response body")
        }
        return
    }

    fmt.Println("Successfully added to database.")
}
