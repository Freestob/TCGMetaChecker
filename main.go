package main

import (
	"fmt"
	"github.com/gocolly/colly/v2"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sync"
	//tea "github.com/charmbracelet/bubbletea"
)

//const url = "https://charm.sh/"

//type model struct {
//	status int
//	err    error
//}

type Card struct {
	Name  string
	Count int
	Price float64
}
type Deck struct {
	Cards     []Card
	Name      string
	Format    string
	Archetype string
	URL       string
}
type Archetype struct {
	Decks  []*Deck
	Name   string
	Format string
	URL    string
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

//	func (m model) View() string {
//		if m.err != nil {
//			return fmt.Sprintf("\nWe had some trouble: %v\n\n", m.err)
//		}
//
//		s := fmt.Sprintf("Checking %s ...", url)
//
//		if m.status > 0 {
//			s += fmt.Sprintf("%d %s!", m.status, http.StatusText(m.status))
//		}
//		return "\n" + s + "\n\n"
//	}

func main() {
	//if _, err := tea.NewProgram(model{}).Run(); err != nil {
	//	fmt.Println(err)
	//	os.Exit(1)
	//}

	startURL := "https://mtgdecks.net/Pioneer/"
	parsedStartURL, err := url.Parse(startURL)
	if err != nil {
		log.Fatalf("Invalid start URL: %v", err)
	}

	allowedDomain := parsedStartURL.Hostname()

	// Instantiate default collector
	c := colly.NewCollector(
		colly.AllowedDomains(allowedDomain))

	results := make(map[string]*Archetype)
	var resultsMutex sync.Mutex

	format := "Pioneer"

	archetypeTable := "table#allArchetypes.clickable.table.sortable.table-striped"
	// Get archetypes
	c.OnHTML(archetypeTable, func(e *colly.HTMLElement) {
		e.ForEach("tr", func(_ int, el *colly.HTMLElement) {
			tierSpan := "td span.text-uppercase.label.label-default"
			tier := el.DOM.Find(tierSpan).Text()
			tier = strings.TrimSpace(tier)
			if /*tier == "tier-2" || */ tier == "tier-1" {
				archetypeSpan := "td.sort strong a.text-uppercase"
				archetypeName := el.DOM.Find(archetypeSpan).Text()

				fmt.Printf("archetypeName=%s\n", archetypeName)
				absoluteArchetypeURL := e.Request.AbsoluteURL(archetypeName)
				fmt.Printf("absoluteArchetypeURL=%s\n", absoluteArchetypeURL)
				if archetypeName != "" &&
					strings.HasPrefix(absoluteArchetypeURL, startURL) &&
					absoluteArchetypeURL != startURL &&
					!strings.Contains(archetypeName, "-decklist-") {

					fmt.Printf("Found Archetype: '%s' -> %s\n", archetypeName, absoluteArchetypeURL)

					ctx := colly.NewContext()
					ctx.Put("archetypeName", archetypeName)
					ctx.Put("archetypeURL", absoluteArchetypeURL)
					ctx.Put("format", format)

					resultsMutex.Lock()

					if _, exists := results[absoluteArchetypeURL]; !exists {
						results[absoluteArchetypeURL] = &Archetype{
							Name:   archetypeName,
							Format: format,
							Decks:  []*Deck{},
						}
					}
					resultsMutex.Unlock()
					err := c.Request("GET", absoluteArchetypeURL, nil, ctx, nil)
					if err != nil {
						log.Printf("Error visiting link %s: %v\n", absoluteArchetypeURL, err)
					}
				}
			}
		})
	})

	tableLinkSelector := "table.clickable.table.table-striped.hidden-xs a[href]"

	//Get decks from archetype page
	c.OnHTML(tableLinkSelector, func(e *colly.HTMLElement) {
		archetypeName := e.Request.Ctx.Get("archetypeName")
		archetypeUrl := e.Request.Ctx.Get("archetypeURL")
		format := e.Request.Ctx.Get("format")

		if archetypeUrl == "" {
			log.Printf("WARN: archetypeUrl missing from context on page %s. Skipping deck", e.Request.URL)
			return
		}

		link := e.Attr("href")

		absoluteDeckUrl := e.Request.AbsoluteURL(link)
		if link != "" && strings.Contains(link, "-decklist-") {
			currentDeckNameFromLink := strings.TrimSpace(e.Text)
			if currentDeckNameFromLink == "" {
				currentDeckNameFromLink = "Unknown Deck Name"
			}
			fmt.Printf("Found Deck: '%s' (for Archetype '%s') -> %s\n", currentDeckNameFromLink, archetypeName, archetypeUrl)

			deckPageCtx := colly.NewContext()
			deckPageCtx.Put("archetypeUrl", archetypeUrl)
			deckPageCtx.Put("deckName", currentDeckNameFromLink)
			deckPageCtx.Put("format", format)
			deckPageCtx.Put("deckUrl", absoluteDeckUrl)

			resultsMutex.Lock()

			parentArchetype, exists := results[archetypeUrl]
			if exists {
				deckExistsInArchetype := false
				for _, existingDeck := range parentArchetype.Decks {
					if existingDeck.URL == absoluteDeckUrl {
						deckExistsInArchetype = true
						break
					}
				}
				if !deckExistsInArchetype {
					parentArchetype.Decks = append(parentArchetype.Decks, &Deck{
						Name:   currentDeckNameFromLink,
						URL:    absoluteDeckUrl,
						Format: format,
						Cards:  []Card{},
					})
				}
			} else {
				log.Printf("WARN: Archetype (URL: %s) not found in results map when trying to add deck '%s'.", archetypeUrl, currentDeckNameFromLink)
			}
			resultsMutex.Unlock()
			err := c.Request("GET", absoluteDeckUrl, nil, deckPageCtx, nil)
			if err != nil {
				log.Printf("Error visiting link %s: %v\n", link, err)
			}
		}
	})

	// Get cards from a deck
	c.OnHTML("tr.cardItem", func(e *colly.HTMLElement) {
		archetypeUrl := e.Request.Ctx.Get("archetypeUrl")
		deckUrl := e.Request.Ctx.Get("deckUrl")

		if archetypeUrl == "" || deckUrl == "" {
			log.Printf("WARN: archetypeUrl or deckUrl missing from context for card on item on %s. Skipping card", e.Request.URL.String())
			return
		}
		deckURL := e.Request.Ctx.Get("deckUrl")
		name := e.Attr("data-card-id")
		req := e.Attr("data-required")
		tcg := e.Attr("tcgplayer")
		if name == "" || req == "" {
			log.Printf("WARN: Missing card name or required count on %s", e.Request.URL.String())
			return
		}
		count, err := strconv.Atoi(req)
		if err != nil {
			fmt.Printf("Error parsing count for card '%s':  %v", name, err)
			return
		}
		price, err := strconv.ParseFloat(tcg, 32)
		if err != nil {
			fmt.Printf("Error parsing  for card '%s', using 0.0: %v", name, err)
			price = 0.0
		}
		card := Card{
			Name:  name,
			Count: count,
			Price: price,
		}

		resultsMutex.Lock()
		defer resultsMutex.Unlock()
		if parentArchetype, exists := results[archetypeUrl]; exists {
			foundDeck := false

			for _, existingDeck := range parentArchetype.Decks {
				if existingDeck.URL == deckURL {
					existingDeck.Cards = append(existingDeck.Cards, card)
					foundDeck = true
					break
				}
			}
			if !foundDeck {
				log.Printf("WARN  Deck URL '%s' not found in results when adding deck '%s'.", deckURL, name)
			}
		}
	})

	// Set a User-Agent to mimic a browser
	c.OnRequest(func(r *colly.Request) {
		fmt.Printf("Visiting: %s\n", r.URL)
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36")
	})

	// Handle any errors
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %s failed with response code: %d\nError: %v", r.Request.URL, r.StatusCode, err)
	})

	// After scraping is done
	c.OnScraped(func(r *colly.Response) {
		fmt.Println("\nFinished scraping", r.Request.URL)
	})

	// Start scraping
	fmt.Println("\nVisiting " + startURL)
	err = c.Visit(startURL)
	if err != nil {
		log.Fatal("Fatal error visiting URL:", err)
	}

	for _, archetypeData := range results {
		fmt.Printf("\nArchetype: %s (Format: %s)\n", archetypeData.Name, archetypeData.Format)
		fmt.Printf("  Found %d Decks\n", len(archetypeData.Decks))

		for i, deckData := range archetypeData.Decks {
			deckCardCount := 0
			deckTotalPrice := 0.0
			for _, card := range deckData.Cards {
				deckCardCount += card.Count
				deckTotalPrice += float64(card.Count) * card.Price
			}

			fmt.Printf("   Deck %d: %s (URL: %s)\n", i+1, deckData.Name, deckData.URL)
			fmt.Printf("   Cards: %d | Estimated Price: %.2f\n", deckCardCount, deckTotalPrice)
		}
	}
}
