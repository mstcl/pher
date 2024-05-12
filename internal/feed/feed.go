package feed

import (
	"fmt"
	"os"
	"time"

	"github.com/gorilla/feeds"
	"github.com/mstcl/pher/internal/config"
	"github.com/mstcl/pher/internal/entry"
)

func constructFeed(
	cfg config.Config,
	d map[string]entry.Entry,
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

	for _, v := range d {
		md := v.Metadata
		if len(md.Date) == 0 {
			continue
		}
		t, err := time.Parse("2006-01-02", md.Date)
		if err != nil {
			return "", fmt.Errorf("parse time: %w", err)
		}
		entry := &feeds.Item{
			Title:       md.Title,
			Link:        &feeds.Link{Href: cfg.Url + v.Href},
			Description: md.Description,
			Author:      author,
			Created:     t,
			Content:     string(v.Body),
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
	if err := os.WriteFile(outDir+"/feed.xml", b, 0o644); err != nil {
		return fmt.Errorf("writing article: %w", err)
	}
	return nil
}

func RenderFeed(
	cfg config.Config,
	d map[string]entry.Entry,
	outDir string,
	isDry bool,
) error {
	atom, err := constructFeed(cfg, d)
	if err != nil {
		return fmt.Errorf("make atom feed: %w", err)
	}
	if err := saveFeed(outDir, atom, isDry); err != nil {
		return fmt.Errorf("write atom feed: %w", err)
	}
	return nil
}
