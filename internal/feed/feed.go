package feed

import (
	"fmt"
	"os"
	"time"

	"github.com/mstcl/pher/internal/config"
	"github.com/mstcl/pher/internal/entry"
)

type FeedDeps struct {
	Config  *config.Config
	Entries map[string]entry.Entry
	InDir   string
	OutDir  string
	DryRun   bool
}

func (d *FeedDeps) ConstructFeed() (string, error) {
	now := time.Now()
	author := &Author{Name: d.Config.AuthorName, Email: d.Config.AuthorEmail}
	feed := &Feed{
		Title:       d.Config.Title,
		Link:        &Link{Href: d.Config.Url},
		Description: d.Config.Description,
		Author:      author,
		Created:     now,
	}

	feed.Items = []*Item{}

	for _, v := range d.Entries {
		md := v.Metadata
		if len(md.Date) == 0 {
			continue
		}
		t, err := time.Parse("2006-01-02", md.Date)
		if err != nil {
			return "", fmt.Errorf("parse time: %w", err)
		}
		entry := &Item{
			Title:       md.Title,
			Link:        &Link{Href: d.Config.Url + v.Href},
			Description: md.Description,
			Author:      author,
			Created:     t,
			Content:     string(v.Body),
			Categories:  md.Tags,
		}
		feed.Items = append(feed.Items, entry)
	}

	atom, err := feed.ToAtom()
	if err != nil {
		return "", fmt.Errorf("parse time: %w", err)
	}

	return atom, nil
}

func (d *FeedDeps) SaveFeed(atom string) error {
	if d.DryRun {
		return nil
	}
	b := []byte(atom)
	if err := os.WriteFile(d.OutDir+"/feed.xml", b, 0o644); err != nil {
		return fmt.Errorf("writing article: %w", err)
	}
	return nil
}
