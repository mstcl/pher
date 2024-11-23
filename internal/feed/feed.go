package feed

import (
	"fmt"
	"os"
	"time"

	"github.com/mstcl/pher/internal/state"
)

func Construct(s *state.State) (string, error) {
	now := time.Now()
	author := &Author{Name: s.Config.AuthorName, Email: s.Config.AuthorEmail}
	feed := &Feed{
		Title:       s.Config.Title,
		Link:        &Link{Href: s.Config.Url},
		Description: s.Config.Description,
		Author:      author,
		Created:     now,
	}

	feed.Items = []*Item{}

	for _, v := range s.Entries {
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
			Link:        &Link{Href: s.Config.Url + v.Href},
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

func Write(s *state.State, atom string) error {
	if s.DryRun {
		return nil
	}

	b := []byte(atom)

	if err := os.WriteFile(s.OutDir+"/feed.xml", b, 0o644); err != nil {
		return fmt.Errorf("writing article: %w", err)
	}

	return nil
}
