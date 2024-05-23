package feed

import (
	"fmt"
	"os"
	"time"

	"github.com/mstcl/pher/internal/config"
	"github.com/mstcl/pher/internal/entry"
)

type Meta struct {
	C      *config.Config
	D      map[string]entry.Entry
	InDir  string
	OutDir string
	IsDry  bool
}

func (m *Meta) ConstructFeed() (string, error) {
	now := time.Now()
	author := &Author{Name: m.C.AuthorName, Email: m.C.AuthorEmail}
	feed := &Feed{
		Title:       m.C.Title,
		Link:        &Link{Href: m.C.Url},
		Description: m.C.Description,
		Author:      author,
		Created:     now,
	}

	feed.Items = []*Item{}

	for _, v := range m.D {
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
			Link:        &Link{Href: m.C.Url + v.Href},
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

func (m *Meta) SaveFeed(atom string) error {
	if m.IsDry {
		return nil
	}
	b := []byte(atom)
	if err := os.WriteFile(m.OutDir+"/feed.xml", b, 0o644); err != nil {
		return fmt.Errorf("writing article: %w", err)
	}
	return nil
}
