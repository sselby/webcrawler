package main

import "fmt"
import "net/http"
import "net/url"
import "golang.org/x/net/html"
import "os"
import "strings"
import "time"
import "strconv"

const bufferSize int = 1000
const maxDop int = 8

type PageLink struct {
	Parent string
	Child  string
}

func main() {
	startLink, urlErr := url.Parse(os.Args[1])
	if urlErr != nil || startLink.Scheme == "" {
		fmt.Println("First argument must be a valid URL")
		os.Exit(2)
	}
	numberOfLinks, argerr := strconv.Atoi(os.Args[2])
	if argerr != nil {
		fmt.Println("Second argument must be an integer")
		os.Exit(2)
	}
	output := make(chan *PageLink)
	linkBuffer := make(chan string, bufferSize)
	overflow := make(chan *PageLink)
	go handleOverflow(overflow, linkBuffer)

	for i := 0; i < maxDop; i++ {
		go scrape(output, linkBuffer, overflow)
	}
	linkBuffer <- startLink.String()
	for i := 0; i < numberOfLinks; i++ {
		select {
		case o := <-output:
			fmt.Printf("%s -> %s\n", o.Parent, o.Child)
		case <-time.After(time.Second * 10):
			fmt.Println("No more links.")
			os.Exit(0)
		}
	}
}

func scrape(output chan *PageLink, linkBuffer chan string, overflow chan *PageLink) {
	defer fmt.Println("!!!scrapeexit!")
	for {
		select {
		case location := <-linkBuffer:
			resp, err := http.Get(location)
			if err != nil {
				fmt.Printf("Error retrieving %s", location)
				fmt.Println("")
				fmt.Println(err.Error())
				continue
			}
			links := findLinks(resp, location)
			for _, link := range links {
				select {
				case linkBuffer <- link.Child:
				default:
					overflow <- link
				}
				output <- link
			}
		case <-time.After(time.Second * 2):
			fmt.Println("Could not read from buffer")
		}
	}
}

func handleOverflow(incoming chan *PageLink, linkBuffer chan string) {
	var overflow []string
	defer fmt.Println("!!Exit!! error")
	for {
		select {
		case failed := <-incoming:
			if len(overflow) > bufferSize*2 {
				//Write to disk for later run
				continue
			}
			overflow = append(overflow, failed.Child)
		default:
			if len(linkBuffer) == 0 && len(overflow) > 0 {
				link := overflow[0]
				if len(overflow) == 1 {
					overflow = make([]string, 0)
				} else {
					overflow = overflow[1:]
				}
				select {
				case linkBuffer <- link:
				default:
					if len(overflow) < bufferSize*2 {
						overflow = append(overflow, link)
					} else {
						// Write to disk for later run
					}
				}
			}
		}
	}
}

func findLinks(resp *http.Response, location string) []*PageLink {
	tokenizer := html.NewTokenizer(resp.Body)
	defer resp.Body.Close()
	links := make([]*PageLink, 0)
	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return links
		case html.StartTagToken:
			tn, _ := tokenizer.TagName()
			if len(tn) == 1 && tn[0] == 'a' {
				for {
					tag, value, moretags := tokenizer.TagAttr()
					if string(tag) == "href" {
						formattedUrl := formatUrl(string(value), location)
						if formattedUrl != "" {
							newLink := new(PageLink)
							newLink.Parent = location
							newLink.Child = formattedUrl
							links = append(links, newLink)
						}
						break
					}
					if !moretags {
						break
					}
				}
			}
		}
	}
	return links
}

func formatUrl(location string, parent string) string {
	if len(location) <= 0 {
		return ""
	}
	if location[0] == '/' {
		parsedParent, err := url.Parse(parent)
		if err != nil {
			return ""
		}
		return fmt.Sprintf("%s://%s%s", parsedParent.Scheme, parsedParent.Host, location)
	}
	if strings.Index(location, "http") == 0 {
		return location
	}
	if location[len(location)-1] == '/' {
		return fmt.Sprintf("%s%s", parent, location)
	}
	if location[0] == '#' {
		return ""
	}
	//Ideally this would get relative links,
	//but it catches a lot of other bad stuff too (like javascript:void(0))
	//We'll ignore those for now
	//return fmt.Sprintf("%s/%s", parent, location)
	return ""
}
