package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	startMarker = "<!-- SNAPSHOTS:START -->"
	endMarker   = "<!-- SNAPSHOTS:END -->"
)

func main() {
	var (
		readme    string
		snapshots string
		cols      int
		width     int
	)
	flag.StringVar(&readme, "readme", "README.md", "Path to README file to update in place")
	flag.StringVar(&snapshots, "snapshots", filepath.Join("test", "integration", "testdata", "snapshots"), "Snapshots directory")
	flag.IntVar(&cols, "cols", 4, "Number of columns per row")
	flag.IntVar(&width, "width", 80, "Image width in pixels")
	flag.Parse()

	entries, err := os.ReadDir(snapshots)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: reading %s: %v\n", snapshots, err)
		os.Exit(1)
	}

	type item struct {
		Name    string
		Encoded string
	}
	var items []item
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".png") {
			continue
		}
		if strings.Contains(name, "_actual.") {
			continue
		}
		base := strings.TrimSuffix(name, ".png")
		items = append(items, item{Name: base, Encoded: url.PathEscape(name)})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	var buf bytes.Buffer
	buf.WriteString("<table>\n")
	if cols <= 0 {
		cols = 3
	}
	for i := 0; i < len(items); i += cols {
		buf.WriteString("  <tr>\n")
		for c := 0; c < cols; c++ {
			if i+c < len(items) {
				it := items[i+c]
				src := filepath.ToSlash(filepath.Join("test", "integration", "testdata", "snapshots", it.Encoded))
				fmt.Fprintf(&buf, "    <td align=\"center\"><div style=\"position:relative;display:inline-block;\"><img src=\"%s\" width=\"%d\" style=\"display:block;transition:transform 0.2s;\" onmouseover=\"this.style.transform='scale(2)';this.style.zIndex='999';\" onmouseout=\"this.style.transform='scale(1)';this.style.zIndex='1';\" /><br><sub>%s ✅</sub></div></td>\n", src, width, it.Name)
			} else {
				buf.WriteString("    <td></td>\n")
			}
		}
		buf.WriteString("  </tr>\n")
	}
	buf.WriteString("</table>\n")

	readmeBytes, err := os.ReadFile(readme)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: reading %s: %v\n", readme, err)
		os.Exit(1)
	}
	content := string(readmeBytes)
	start := strings.Index(content, startMarker)
	end := strings.Index(content, endMarker)
	if start == -1 || end == -1 || end < start {
		fmt.Fprintf(os.Stderr, "error: markers not found in %s. Ensure %s and %s exist.\n", readme, startMarker, endMarker)
		os.Exit(1)
	}
	before := content[:start+len(startMarker)]
	after := content[end:]
	var out bytes.Buffer
	out.WriteString(before)
	out.WriteString("\n")
	out.Write(buf.Bytes())
	if !strings.HasPrefix(after, "\n") {
		out.WriteString("\n")
	}
	out.WriteString(after)

	if err := os.WriteFile(readme, out.Bytes(), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error: writing %s: %v\n", readme, err)
		os.Exit(1)
	}
}
