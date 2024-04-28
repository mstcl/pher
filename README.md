# pher

**A wiki-style static site generator**

Written in go, inspired by [ter](https://github.com/kkga/ter).

## Features

pher aims to match [ter](https://github.com/kkga/ter)'s goals and features, but
there are a few differences:

- Comes with markdown extensions like footnotes and smartypants.
- Wikilinks are supported thanks to abhinav's
  [extension](https://github.com/abhinav/goldmark-wikilink).
- No CSS framework.
- Comes a small standalone binary (~13M). No need for a runtime.
- Slight visual tweaks (personal preference).
- The atom feed contains only dated entries
- The atom feed does not have the full text (RSS clients apps can handle this).

## Installation

```bash
$ go install github.com/mstcl/pher@latest
```

## Usage

```
Usage of pher:
  -c string
        Path to config file (default "config.yaml")
  -d    Dry run---don't render (default false)
  -i string
        Input directory (default ".")
  -o string
        Output directory (default "_site")
```

## Configuration

```yaml
# atom feed options
title: "" # wiki title
description: "" # wiki description
url: "" # external link to the wiki (https://...)
authorName: "" # author's name
authorEmail: "" # author's email

# rendering options
rootCrumb: "~" # render root link in navbar with this string.
codeHighlight: true # render code with syntax highlighting.

# footer links, leave empty e.g. `footer: []` to disable
footer:
  - text: "license"
    href: "https://link.to.license"
  - text: "feed"
    href: "/feed.xml"
```

## Frontmatter

pher reads in frontmatter in YAML format. Available fields and default values
are:

```yaml
---
title: "" # Entry's title
description: "" # Entry's description
tags: [] # Entry's list of tags
date: "" # Entry's date YYYY-MM-DD format
pinned: false # Pin entry at the top of the listing
unlisted: false # Remove entry from the listing
draft: false # Don't render this entry
toc: false # Render a table of contents for this entry
showHeader: true # Show the header (title, description, tags, date)
layout: "list" # Available values: "grid", "list". Only for index entry of each subdirectory.
head: "" # String to inject inside HTML <head>
---
```

## To do

- [x] Implement navigation breadcrumbs
- [x] Listing (lists/grid)
- [x] Fix TOC
- [x] Implement logic
  - [x] hiding TOC
  - [x] pinning entries
  - [x] unlisting entries
- [x] Tags page
- [x] Related links (by tags)
- [x] Atom feed
- [x] Configuration to inject into HTML <head>
- [x] Frontmatter field for `dateUpdated`

## Notes

pher embeds the templates in `web/templates` with go:embed. This means pher can
run as a standalone binary. Unfortunately, to modify the templates, we have to
recompile.

## Credits

- [ter](https://github.com/kkga/ter) - main inspiration
- [go-vite](https://github.com/icyphox/go-vite) - aesthetic
- [goldmark](https://github.com/yuin/goldmark) - markdown parser
