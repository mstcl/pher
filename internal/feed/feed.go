package feed

import (
	"fmt"
	"os"
	"time"

	"github.com/gorilla/feeds"
	"github.com/mstcl/pher.git/internal/config"
	"github.com/mstcl/pher.git/internal/parse"
)

func MakeFeed(
	cfg config.Config,
	m map[string]parse.Metadata,
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
		}
		feed.Items = append(feed.Items, entry)
	}

	atom, err := feed.ToAtom()
	if err != nil {
		return "", fmt.Errorf("parse time: %w", err)
	}

	return atom, nil
}

func SaveFeed(outDir, atom string, isDry bool) error {
	if isDry {
		return nil
	}
	b := []byte(atom)
	err := os.WriteFile(outDir+"/feed.xml", b, 0644)
	if err != nil {
		return fmt.Errorf("writing article: %w", err)
	}
	return nil
}
