package feed

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/mstcl/pher/v2/internal/state"
)

func Construct(s *state.State, logger *slog.Logger) (string, error) {
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
		child := logger.With(slog.String("href", v.Href), slog.String("context", "atom feed"))

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

		child.Debug("Atom entry created")
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
