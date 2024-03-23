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

type Parser interface {
    Parse(*html.Node)
    GetLink() string
    GetData() *RequestData
}

type LinkedIn struct {
    Link string
    Data *RequestData
}

func (l LinkedIn) GetLink() string {
    return l.Link
}

func (l LinkedIn) GetData() *RequestData {
    return l.Data
}

type Greenhouse struct {
    Data *RequestData
}

func (g Greenhouse) GetLink() string {
    property := g.Data.Properties["Link"].(map[string]interface{})
    return property["url"].(string)
}

func (g Greenhouse) GetData() *RequestData {
    return g.Data
}

type Lever struct {
    Data *RequestData
}

func (l Lever) GetLink() string {
    property := l.Data.Properties["Link"].(map[string]interface{})
    return property["url"].(string)
}

func (l Lever) GetData() *RequestData {
    return l.Data
}

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

            var link string
            if cCtx.Args().Len() > 0 {
                link = cCtx.Args().Get(0)
            } else {
                link = getInput("link")
            }

            parser := getParser(link, &requestData)
            if manual || parser == nil {
                manualInput(&requestData)
                notionRequest(requestData)
            } else {
                scrapeLink(parser)
                notionRequest(*parser.GetData())
            }

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

func getParser(link string, data *RequestData) Parser {
    if strings.Contains(link, "linkedin.com") {
        re := regexp.MustCompile(`\d{9,}`)
        jobId := string(re.Find([]byte(link)))
        jobLink := "https://www.linkedin.com/jobs-guest/jobs/api/jobPosting/" + jobId
        data.Properties["Link"] = map[string]interface{}{
            "type": "url",
            "url": "https://www.linkedin.com/jobs/view/" + jobId,
        }
        return LinkedIn{Link: jobLink, Data: data}
    } else if strings.Contains(link, "boards.greenhouse.io") {
        data.Properties["Link"] = map[string]interface{}{
            "type": "url",
            "url": strings.TrimSpace(link),
        }
        return Greenhouse{Data: data}
    } else if strings.Contains(link, "jobs.lever.co") {
        data.Properties["Link"] = map[string]interface{}{
            "type": "url",
            "url": strings.TrimSpace(link),
        }
        re := regexp.MustCompile(`jobs.lever.co/(.*)/`)
        companyName := re.FindStringSubmatch(link)[1]
        data.Properties["Company"] = map[string]interface{}{
            "type": "title",
            "title": []map[string]interface{}{
                {
                    "type": "text",
                    "text": map[string]string{
                        "content": companyName,
                    },
                },
            },
        }
        return Lever{Data: data}
    }
    data.Properties["Link"] = map[string]interface{}{
        "type": "url",
        "url": strings.TrimSpace(link),
    }
    return nil
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
}

func scrapeLink(parser Parser) {
    var data *RequestData
    attempts, missing := 0, true
    for attempts < 3 && missing {
        attempts++
        reader, err := getContent(parser.GetLink())
        if err != nil {
            log.Println("Error getting page contents", err)
            continue
        }

        nodeTree, err := html.Parse(reader)
        if err != nil {
            log.Println("Error parsing html file", err)
            continue
        }

        parser.Parse(nodeTree)
        data = parser.GetData()

        missing = false
        if _, ok := data.Properties["Company"]; !ok {
            missing = true
        } else if _, ok := data.Properties["Position"]; !ok {
            missing = true
        } else if _, ok := data.Properties["Location"]; !ok {
            missing = true
        }
    }

    if missing {
        manualInput(data)
    }
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

func (l LinkedIn) Parse(n *html.Node) {
    if n.Type == html.ElementNode {
        for _, a := range n.Attr {
            if a.Key == "class" {
                if strings.Contains(a.Val, "topcard__org-name-link") {
                    l.Data.Properties["Company"] = map[string]interface{}{
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
                    l.Data.Properties["Position"] = map[string]interface{}{
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
                    l.Data.Properties["Location"] = map[string]interface{}{
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
        l.Parse(child)
    }
}

func (g Greenhouse) Parse(n *html.Node) {
    if n.Type == html.ElementNode {
        for _, a := range n.Attr {
            if a.Key == "class" {
                if strings.Contains(a.Val, "company-name") {
                    re := regexp.MustCompile(`at (.+)`)
                    textContent := re.FindStringSubmatch(strings.TrimSpace(n.FirstChild.Data))[1]
                    g.Data.Properties["Company"] = map[string]interface{}{
                        "type": "title",
                        "title": []map[string]interface{}{
                            {
                                "type": "text",
                                "text": map[string]string{
                                    "content": textContent,
                                },
                            },
                        },
                    }
                } else if strings.Contains(a.Val, "app-title") {
                    g.Data.Properties["Position"] = map[string]interface{}{
                        "rich_text": []map[string]interface{}{
                            {
                                "type": "text",
                                "text": map[string]string{
                                    "content": strings.TrimSpace(n.FirstChild.Data),
                                },
                            },
                        },
                    }
                } else if strings.Contains(a.Val, "location") {
                    g.Data.Properties["Location"] = map[string]interface{}{
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
        g.Parse(child)
    }
}

func (l Lever) Parse(n *html.Node) {
    if n.Type == html.ElementNode {
        for _, a := range n.Attr {
            if a.Key == "class" {
                if strings.Contains(a.Val, "posting-headline") {
                    l.Data.Properties["Position"] = map[string]interface{}{
                        "rich_text": []map[string]interface{}{
                            {
                                "type": "text",
                                "text": map[string]string{
                                    "content": strings.TrimSpace(n.FirstChild.FirstChild.Data),
                                },
                            },
                        },
                    }
                } else if strings.Contains(a.Val, "location") {
                    l.Data.Properties["Location"] = map[string]interface{}{
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
        l.Parse(child)
    }
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
