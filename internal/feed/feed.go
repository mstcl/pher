package feed

import (
	"fmt"
	"os"
	"time"

	"github.com/gorilla/feeds"
	"github.com/mstcl/pher/internal/config"
	"github.com/mstcl/pher/internal/parse"
)

func makeFeed(
	cfg config.Config,
	m map[string]parse.Metadata,
	c map[string][]byte,
	h map[string]string,
) (string, error) {
	now := time.Now()
	author := &feeds.Author{Name: cfg.AuthorName, Email: cfg.AuthorEmail}
	feed := &feeds.Feed{
		Title:       cfg.Title,
		Link:        &feeds.Link{Href: cfg.Url},
		Description: cfg.Description,
		Author:      author,
		Created:     now,
	}

	feed.Items = []*feeds.Item{}

	for k, v := range m {
		if len(v.Date) == 0 {
			continue
		}
		t, err := time.Parse("2006-01-02", v.Date)
		if err != nil {
			return "", fmt.Errorf("parse time: %w", err)
		}
		entry := &feeds.Item{
			Title:       v.Title,
			Link:        &feeds.Link{Href: cfg.Url + h[k]},
			Description: v.Description,
			Author:      author,
			Created:     t,
			Content:     string(c[k]),
		}
		feed.Items = append(feed.Items, entry)
	}

	atom, err := feed.ToAtom()
	if err != nil {
		return "", fmt.Errorf("parse time: %w", err)
	}

	return atom, nil
}

func saveFeed(outDir, atom string, isDry bool) error {
	if isDry {
		return nil
	}
	b := []byte(atom)
	if err := os.WriteFile(outDir+"/feed.xml", b, 0644); err != nil {
		return fmt.Errorf("writing article: %w", err)
	}
	return nil
}

func FetchFeed(
	cfg config.Config,
	m map[string]parse.Metadata,
	c map[string][]byte,
	h map[string]string,
	outDir string,
	isDry bool,
) error {
	atom, err := makeFeed(cfg, m, c, h)
	if err != nil {
		return fmt.Errorf("make atom feed: %w", err)
	}
	if err := saveFeed(outDir, atom, isDry); err != nil {
		return fmt.Errorf("write atom feed: %w", err)
	}
	return nil
}
