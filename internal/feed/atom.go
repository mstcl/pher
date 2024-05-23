package feed

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"sort"
	"time"
)

// Generates Atom feed as XML
// Taken relevant bits from https://github.com/gorilla/feeds/blob/main/feed.go
// and https://github.com/gorilla/feeds/blob/main/atom.go

const ns = "http://www.w3.org/2005/Atom"

// add a new Item to a Feed
func (f *Feed) Add(item *Item) {
	f.Items = append(f.Items, item)
}

type Link struct {
	Href, Rel, Type, Length string
}

type Author struct {
	Name, Email string
}

type Image struct {
	Url, Title, Link string
	Width, Height    int
}

type Enclosure struct {
	Url, Length, Type string
}

type Item struct {
	Link        *Link
	Source      *Link
	Author      *Author
	Enclosure   *Enclosure
	Title       string
	Description string // used as description in rss, summary in atom
	Id          string // used as guid in rss, id in atom
	Updated     time.Time
	Created     time.Time
	Content     string
	Categories  []string
}

type Feed struct {
	Updated     time.Time
	Created     time.Time
	Link        *Link
	Author      *Author
	Image       *Image
	Title       string
	Description string
	Id          string
	Subtitle    string
	Copyright   string
	Items       []*Item
}

// interface used by ToXML to get a object suitable for exporting XML.
type XmlFeed interface {
	FeedXml() interface{}
}

// turn a feed object (either a Feed, AtomFeed, or RssFeed) into xml
// returns an error if xml marshaling fails
func ToXML(feed XmlFeed) (string, error) {
	x := feed.FeedXml()
	data, err := xml.MarshalIndent(x, "", "  ")
	if err != nil {
		return "", err
	}
	// strip empty line from default xml header
	s := xml.Header[:len(xml.Header)-1] + string(data)
	return s, nil
}

// WriteXML writes a feed object (either a Feed, AtomFeed, or RssFeed) as XML into
// the writer. Returns an error if XML marshaling fails.
func WriteXML(feed XmlFeed, w io.Writer) error {
	x := feed.FeedXml()
	// write default xml header, without the newline
	if _, err := w.Write([]byte(xml.Header[:len(xml.Header)-1])); err != nil {
		return err
	}
	e := xml.NewEncoder(w)
	e.Indent("", "  ")
	return e.Encode(x)
}

// // creates an Atom representation of this feed
func (f *Feed) ToAtom() (string, error) {
	a := &Atom{f}
	return ToXML(a)
}

// WriteAtom writes an Atom representation of this feed to the writer.
func (f *Feed) WriteAtom(w io.Writer) error {
	return WriteXML(&Atom{f}, w)
}

// Sort sorts the Items in the feed with the given less function.
func (f *Feed) Sort(less func(a, b *Item) bool) {
	lessFunc := func(i, j int) bool {
		return less(f.Items[i], f.Items[j])
	}
	sort.SliceStable(f.Items, lessFunc)
}

type AtomPerson struct {
	Name  string `xml:"name,omitempty"`
	Uri   string `xml:"uri,omitempty"`
	Email string `xml:"email,omitempty"`
}

type AtomSummary struct {
	XMLName xml.Name `xml:"summary"`
	Content string   `xml:",chardata"`
	Type    string   `xml:"type,attr"`
}

type AtomContent struct {
	XMLName xml.Name `xml:"content"`
	Content string   `xml:",chardata"`
	Type    string   `xml:"type,attr"`
}

type AtomAuthor struct {
	XMLName xml.Name `xml:"author"`
	AtomPerson
}

type AtomContributor struct {
	XMLName xml.Name `xml:"contributor"`
	AtomPerson
}

type AtomCategories []string

type AtomEntry struct {
	Content     *AtomContent
	Author      *AtomAuthor
	Summary     *AtomSummary
	Contributor *AtomContributor
	XMLName     xml.Name `xml:"entry"`
	Updated     string   `xml:"updated"`
	Rights      string   `xml:"rights,omitempty"`
	Source      string   `xml:"source,omitempty"`
	Published   string   `xml:"published,omitempty"`
	Id          string   `xml:"id"`
	Title       string   `xml:"title"`
	Xmlns       string   `xml:"xmlns,attr,omitempty"`
	Links       []AtomLink
	Categories  AtomCategories `xml:"category,omitempty"`
}

// Multiple links with different rel can coexist
type AtomLink struct {
	// Atom 1.0 <link rel="enclosure" type="audio/mpeg" title="MP3" href="http://www.example.org/myaudiofile.mp3" length="1234" />
	XMLName xml.Name `xml:"link"`
	Href    string   `xml:"href,attr"`
	Rel     string   `xml:"rel,attr,omitempty"`
	Type    string   `xml:"type,attr,omitempty"`
	Length  string   `xml:"length,attr,omitempty"`
}

type AtomFeed struct {
	XMLName     xml.Name `xml:"feed"`
	Xmlns       string   `xml:"xmlns,attr"`
	Title       string   `xml:"title"`   // required
	Id          string   `xml:"id"`      // required
	Updated     string   `xml:"updated"` // required
	Icon        string   `xml:"icon,omitempty"`
	Logo        string   `xml:"logo,omitempty"`
	Rights      string   `xml:"rights,omitempty"` // copyright used
	Subtitle    string   `xml:"subtitle,omitempty"`
	Link        *AtomLink
	Author      *AtomAuthor `xml:"author,omitempty"`
	Contributor *AtomContributor
	Entries     []*AtomEntry `xml:"entry"`
}

type Atom struct {
	*Feed
}

// returns the first non-zero time formatted as a string or ""
func anyTimeFormat(format string, times ...time.Time) string {
	for _, t := range times {
		if !t.IsZero() {
			return t.Format(format)
		}
	}
	return ""
}

func newAtomEntry(i *Item) *AtomEntry {
	id := i.Id
	link := i.Link
	if link == nil {
		link = &Link{}
	}
	if len(id) == 0 {
		// if there's no id set, try to create one, either from data or just a uuid
		if len(link.Href) > 0 && (!i.Created.IsZero() || !i.Updated.IsZero()) {
			dateStr := anyTimeFormat("2006-01-02", i.Updated, i.Created)
			host, path := link.Href, "/invalid.html"
			if url, err := url.Parse(link.Href); err == nil {
				host, path = url.Host, url.Path
			}
			id = fmt.Sprintf("tag:%s,%s:%s", host, dateStr, path)
		} else {
			id = "urn:uuid:" + NewUUID().String()
		}
	}
	var name, email string
	if i.Author != nil {
		name, email = i.Author.Name, i.Author.Email
	}

	link_rel := link.Rel
	if link_rel == "" {
		link_rel = "alternate"
	}
	x := &AtomEntry{
		Title:   i.Title,
		Links:   []AtomLink{{Href: link.Href, Rel: link_rel, Type: link.Type}},
		Id:      id,
		Updated: anyTimeFormat(time.RFC3339, i.Updated, i.Created),
	}

	// if there's a description, assume it's html
	if len(i.Description) > 0 {
		x.Summary = &AtomSummary{Content: i.Description, Type: "html"}
	}

	// if there's a content, assume it's html
	if len(i.Content) > 0 {
		x.Content = &AtomContent{Content: i.Content, Type: "html"}
	}

	if i.Enclosure != nil && link_rel != "enclosure" {
		x.Links = append(x.Links, AtomLink{Href: i.Enclosure.Url, Rel: "enclosure", Type: i.Enclosure.Type, Length: i.Enclosure.Length})
	}

	if len(name) > 0 || len(email) > 0 {
		x.Author = &AtomAuthor{AtomPerson: AtomPerson{Name: name, Email: email}}
	}

	if len(i.Categories) > 0 {
		x.Categories = i.Categories
	}
	return x
}

// create a new AtomFeed with a generic Feed struct's data
func (a *Atom) AtomFeed() *AtomFeed {
	updated := anyTimeFormat(time.RFC3339, a.Updated, a.Created)
	link := a.Link

	if link == nil {
		link = &Link{}
	}

	feed := &AtomFeed{
		Xmlns:    ns,
		Title:    a.Title,
		Link:     &AtomLink{Href: link.Href, Rel: link.Rel},
		Subtitle: a.Description,
		Id:       link.Href,
		Updated:  updated,
		Rights:   a.Copyright,
	}

	if a.Author != nil {
		feed.Author = &AtomAuthor{AtomPerson: AtomPerson{Name: a.Author.Name, Email: a.Author.Email}}
	}

	for _, e := range a.Items {
		feed.Entries = append(feed.Entries, newAtomEntry(e))
	}

	return feed
}

// FeedXml returns an XML-Ready object for an Atom object
func (a *Atom) FeedXml() interface{} {
	return a.AtomFeed()
}

// FeedXml returns an XML-ready object for an AtomFeed object
func (a *AtomFeed) FeedXml() interface{} {
	return a
}
