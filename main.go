package main

import (
	"fmt"
	"github.com/gocolly/colly"
	"log"
	"strconv"
	"strings"
	//tea "github.com/charmbracelet/bubbletea"
)

//const url = "https://charm.sh/"

//type model struct {
//	status int
//	err    error
//}

type card struct {
	name  string
	count int
	price float64
}
type deck struct {
	cards  []card
	name   string
	format string
}
type archetype struct {
	decks  []deck
	name   string
	format string
}

//func checkServer() tea.Msg {
//	c := &http.Client{Timeout: time.Second * 10}
//	res, err := c.Get(url)
//	if err != nil {
//		return errMsg{err}
//	}
//	return statusMsg(res.StatusCode)
//}

//type statusMsg int
//type errMsg struct {
//	err error
//}

//func (e errMsg) Error() string { return e.err.Error() }

//func (m model) Init() tea.Cmd {
//	return checkServer
//}
//
//func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
//	switch msg := msg.(type) {
//	case statusMsg:
//		m.status = int(msg)
//		return m, tea.Quit
//
//	case errMsg:
//		m.err = msg
//		return m, tea.Quit
//
//	case tea.KeyMsg:
//		if msg.Type == tea.KeyCtrlC {
//			return m, tea.Quit
//		}
//	}
//	return m, nil
//}

//func (m model) View() string {
//	if m.err != nil {
//		return fmt.Sprintf("\nWe had some trouble: %v\n\n", m.err)
//	}
//
//	s := fmt.Sprintf("Checking %s ...", url)
//
//	if m.status > 0 {
//		s += fmt.Sprintf("%d %s!", m.status, http.StatusText(m.status))
//	}
//	return "\n" + s + "\n\n"
//}

func main() {
	//if _, err := tea.NewProgram(model{}).Run(); err != nil {
	//	fmt.Println(err)
	//	os.Exit(1)
	//}

	// Instantiate default collector
	url := "https://mtgdecks.net/Pioneer/rakdos-demons"

	// Instantiate default collector
	c := colly.NewCollector()

	// Set a timeout for requests
	//c.SetRequestTimeout(15 * time.Second) // 15 seconds should be plenty

	var cards []card
	var decks []deck
	var archetypes []archetype

	deckName := ""
	archetypeSelector := "a.text-uppercase"
	ctx := colly.NewContext()
	// Get archetypes
	c.OnHTML(archetypeSelector, func(e *colly.HTMLElement) {
		link := e.Attr("href")
		fmt.Printf("Link: %s\n", link)
		archetypeName := strings.ReplaceAll(e.Text, " ", "-")
		fmt.Printf("Name: %s\n", e.Text)
		if archetypeName != "" && strings.Contains(link, url) {
			fmt.Printf("Visiting %s\n", link)
			err := e.Request.Visit(link)
			if err != nil {
				log.Printf("Error visiting link %s: %v\n", link, err)
			}
			archetype := archetype{
				decks:  decks,
				name:   e.Text,
				format: "Pioneer",
			}
			archetypes = append(archetypes, archetype)
		}
	})
	tableLinkSelector := "table.clickable.table.table-striped.hidden-xs a[href]"
	//Get decks from archetype page
	c.OnHTML(tableLinkSelector, func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if link != "" && strings.Contains(link, "-decklist-") {
			deckName = e.Text
			ctx.Put("deckName", deckName)
			fmt.Printf("Visiting %s\n", link)
			err := e.Request.Visit(link)
			if err != nil {
				log.Printf("Error visiting link %s: %v\n", link, err)
			}
			deck := deck{
				cards:  cards,
				name:   e.Text,
				format: "Pioneer",
			}

			decks = append(decks, deck)
		}
	})
	// Get cards from a deck
	c.OnHTML("tr.cardItem", func(e *colly.HTMLElement) {

		name := e.Attr("data-card-id")
		req := e.Attr("data-required")
		tcg := e.Attr("tcgplayer")
		count, err := strconv.Atoi(req)
		if err != nil {
			fmt.Println("error parsing count")
			return
		}
		price, err := strconv.ParseFloat(tcg, 32)
		if err != nil {
			fmt.Println("error parsing price")
		}
		card := card{
			name:  name,
			count: count,
			price: price,
		}
		cards = append(cards, card)
	})

	// Set a User-Agent to mimic a browser
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36")
	})

	// Handle any errors
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %s failed with response code: %d\nError: %v", r.Request.URL, r.StatusCode, err)
	})

	// After scraping is done
	c.OnScraped(func(r *colly.Response) {
		for _, deck := range decks {
			fmt.Printf("%s\n", deck.name)
			for _, card := range deck.cards {
				fmt.Printf("%s\n", card.name)
			}

		}
		fmt.Println("\nFinished scraping", r.Request.URL)
	})

	// Start scraping
	err := c.Visit(url)
	if err != nil {
		log.Fatal("Fatal error visiting URL:", err)
	}
}
