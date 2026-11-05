package embedded

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const (
	docsWebRoot = "docs/web"
	accentToken = "--accent"
)

// paletteTokens are the canonical colors of docs/web, owned by tokens.css:
// no page may redeclare them, with or without an exception (ADR-018).
var paletteTokens = []string{
	"--bg", "--bg-elev", "--bg-soft", "--text", "--muted",
	"--line", "--line-strong", "--border", "--border-strong",
	"--page", "--surface", "--raised", "--ink",
	"--accent", "--accent-2", "--accent-3", "--accent-soft", "--accent-on-dark",
}

// docsWebPages are the three published pages; docsWebScript is the shared
// script the pages load.
var (
	docsWebPages  = []string{"index.html", "design-system.html", "landing.html"}
	docsWebScript = "assets/js/docs.js"
)

var (
	// resourceAttr matches href and src attributes but not Vue bindings (:href).
	resourceAttr = regexp.MustCompile(`(?:^|[\s"'])(?:href|src)="([^"]*)"`)
	cssURL       = regexp.MustCompile(`url\(\s*['"]?([^'")]*?)['"]?\s*\)`)
	cssComment   = regexp.MustCompile(`(?s)/\*.*?\*/`)
	styleBlock   = regexp.MustCompile(`(?s)<style[^>]*>(.*?)</style>`)
	cssName      = regexp.MustCompile(`(--[A-Za-z0-9_-]+)\s*:`)
	urlScheme    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)
	loadTag      = regexp.MustCompile(`(?is)<(?:link|script|img|source|iframe)\b[^>]*>`)
	configScript = regexp.MustCompile(`(?s)<script type="application/json" id="docs-config">(.*?)</script>`)
	guideScript  = regexp.MustCompile(`(?s)<script type="application/json" id="tasks-ai-en">(.*?)</script>`)
)

// resourceRefs returns every URL the content loads: href and src attributes
// plus CSS url() values.
func resourceRefs(content string) []string {
	var refs []string

	for _, match := range resourceAttr.FindAllStringSubmatch(content, -1) {
		refs = append(refs, match[1])
	}

	for _, match := range cssURL.FindAllStringSubmatch(content, -1) {
		refs = append(refs, match[1])
	}

	return refs
}

// offsiteReference reports whether a URL intentionally leaves the published
// folder: an anchor, a mailto, a data URI or an absolute http(s) address.
func offsiteReference(ref string) bool {
	for _, prefix := range []string{"#", "mailto:", "data:", "http://", "https://"} {
		if strings.HasPrefix(ref, prefix) {
			return true
		}
	}

	return false
}

// referenceLeavesDocsWeb reports whether a relative resource reference resolves
// outside docs/web. A scheme that is not offsite is also a dependency.
func referenceLeavesDocsWeb(ref string) bool {
	if ref == "" || offsiteReference(ref) {
		return false
	}

	if urlScheme.MatchString(ref) || strings.HasPrefix(ref, "/") {
		return true
	}

	ref = strings.SplitN(strings.SplitN(ref, "#", 2)[0], "?", 2)[0]
	if ref == "" {
		return false
	}

	resolved := filepath.Clean(filepath.Join(docsWebRoot, ref))

	return resolved != docsWebRoot && !strings.HasPrefix(resolved, docsWebRoot+string(filepath.Separator))
}

// externalLoads returns the http(s) addresses loaded by link, script, img,
// source or iframe tags: published offline, they cannot be fetched.
func externalLoads(content string) []string {
	var refs []string

	for _, tag := range loadTag.FindAllString(content, -1) {
		for _, match := range resourceAttr.FindAllStringSubmatch(tag, -1) {
			if strings.HasPrefix(match[1], "http://") || strings.HasPrefix(match[1], "https://") {
				refs = append(refs, match[1])
			}
		}
	}

	return refs
}

// inlineStyle joins the inline <style> blocks of a page.
func inlineStyle(content string) string {
	var css strings.Builder

	for _, match := range styleBlock.FindAllStringSubmatch(content, -1) {
		css.WriteString(match[1])
		css.WriteByte('\n')
	}

	return css.String()
}

// cssRule is one leaf selector and body pair of a style sheet.
type cssRule struct {
	selector string
	body     string
}

// cssRules extracts the leaf rules of a style sheet, with whitespace collapsed
// and comments removed. A rule nested in @media or @supports keeps its at-rule
// as part of the selector, so the same declarations under different conditions
// are different rules.
func cssRules(css string) []cssRule {
	var rules []cssRule

	collectCSSRules(cssComment.ReplaceAllString(css, ""), "", &rules)

	return rules
}

func collectCSSRules(css, prefix string, rules *[]cssRule) {
	for {
		open := strings.IndexByte(css, '{')
		if open < 0 {
			return
		}

		end := matchingBrace(css, open)
		if end < 0 {
			return
		}

		selector := collapseCSS(css[:open])
		body := css[open+1 : end]

		switch {
		case strings.HasPrefix(selector, "@media") || strings.HasPrefix(selector, "@supports"):
			collectCSSRules(body, selector, rules)
		case strings.HasPrefix(selector, "@"):
			*rules = append(*rules, cssRule{selector: selector, body: collapseCSS(body)})
		default:
			*rules = append(*rules, cssRule{
				selector: collapseCSS(prefix + " " + selector),
				body:     collapseCSS(body),
			})
		}

		css = css[end+1:]
	}
}

func matchingBrace(css string, open int) int {
	depth := 0

	for i := open; i < len(css); i++ {
		switch css[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

func collapseCSS(css string) string {
	return strings.Join(strings.Fields(css), " ")
}

// cssSheet is one named source of rules: a page's inline style or a shared
// style sheet.
type cssSheet struct {
	name string
	css  string
}

// repeatedRules returns the rules that belong to more than one sheet.
func repeatedRules(sheets []cssSheet) []string {
	sources := map[cssRule]map[string]bool{}

	for _, sheet := range sheets {
		for _, rule := range cssRules(sheet.css) {
			if sources[rule] == nil {
				sources[rule] = map[string]bool{}
			}

			sources[rule][sheet.name] = true
		}
	}

	var repeated []string

	for rule, owners := range sources {
		if len(owners) < 2 {
			continue
		}

		names := make([]string, 0, len(owners))
		for name := range owners {
			names = append(names, name)
		}

		slices.Sort(names)
		repeated = append(repeated, fmt.Sprintf("%s {%s} in %s", rule.selector, rule.body, strings.Join(names, ", ")))
	}

	slices.Sort(repeated)

	return repeated
}

// forbiddenRootTokens returns the forbidden custom properties a sheet declares
// inside :root blocks, except the names the page owns.
func forbiddenRootTokens(css string, allowed []string) []string {
	forbidden := slices.Clone(paletteTokens)
	found := map[string]bool{}

	for _, rule := range cssRules(css) {
		if !strings.HasPrefix(rule.selector, ":root") {
			continue
		}

		for _, match := range cssName.FindAllStringSubmatch(rule.body, -1) {
			if slices.Contains(forbidden, match[1]) && !slices.Contains(allowed, match[1]) {
				found[match[1]] = true
			}
		}
	}

	var tokens []string

	for name := range found {
		tokens = append(tokens, name)
	}

	slices.Sort(tokens)

	return tokens
}

// sameFile fails when the two files differ.
func sameFile(got, want string) error {
	//nolint:gosec // the path comes from this repository, not from outside
	gotBytes, err := os.ReadFile(got)
	if err != nil {
		return err
	}

	//nolint:gosec // the path comes from this repository, not from outside
	wantBytes, err := os.ReadFile(want)
	if err != nil {
		return err
	}

	if !bytes.Equal(gotBytes, wantBytes) {
		return fmt.Errorf("%s differs from %s", got, want)
	}

	return nil
}

// sameTree fails when the two directories do not hold the same files with the
// same contents.
func sameTree(got, want string) error {
	gotFiles, err := treeFiles(got)
	if err != nil {
		return err
	}

	wantFiles, err := treeFiles(want)
	if err != nil {
		return err
	}

	paths := make([]string, 0, len(gotFiles))

	for path := range gotFiles {
		paths = append(paths, path)
	}

	slices.Sort(paths)

	for _, path := range paths {
		wantContent, ok := wantFiles[path]
		if !ok {
			return fmt.Errorf("%s: %s is missing from %s", got, path, want)
		}

		if !bytes.Equal(gotFiles[path], wantContent) {
			return fmt.Errorf("%s: %s differs from %s", got, path, filepath.Join(want, path))
		}
	}

	for _, path := range slices.Sorted(maps.Keys(wantFiles)) {
		if _, ok := gotFiles[path]; !ok {
			return fmt.Errorf("%s: %s is missing from %s", got, path, got)
		}
	}

	return nil
}

func treeFiles(root string) (map[string][]byte, error) {
	files := map[string][]byte{}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		//nolint:gosec // the path comes from this repository, not from outside
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		files[rel] = content

		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

// nonEmptyFile fails when the file is missing or empty.
func nonEmptyFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.Size() == 0 {
		return fmt.Errorf("%s is empty", path)
	}

	return nil
}

// TestDocsWebPagesAreSelfContained fails when a page or the shared script loads
// something from outside docs/web or from the network.
func TestDocsWebPagesAreSelfContained(t *testing.T) {
	t.Parallel()

	sources := make([]cssSheet, 0, len(docsWebPages)+1)

	for _, page := range docsWebPages {
		sources = append(sources, cssSheet{page, readFile(t, filepath.Join(docsWebRoot, page))})
	}

	sources = append(sources, cssSheet{docsWebScript, readFile(t, filepath.Join(docsWebRoot, docsWebScript))})

	for _, source := range sources {
		for _, ref := range resourceRefs(source.css) {
			if referenceLeavesDocsWeb(ref) {
				t.Errorf("%s: %q resolves outside %s", source.name, ref, docsWebRoot)
			}
		}

		for _, ref := range externalLoads(source.css) {
			t.Errorf("%s: loads %q from the network", source.name, ref)
		}
	}
}

// TestDocsWebStyleReport covers the pure checks behind the furniture and token
// guards with crafted snippets, so each guard is known to bite.
func TestDocsWebStyleReport(t *testing.T) {
	t.Parallel()

	t.Run("resource references", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			ref  string
			want bool
		}{
			{"anchor", "#top", false},
			{"mailto", "mailto:someone@example.com", false},
			{"data URI", "data:image/png;base64,AAAA", false},
			{"website", "https://heroicons.com/", false},
			{"inside", "assets/img/favicons/favicon-32.png", false},
			{"fragment on page", "index.html#limites", false},
			{"parent folder", "../PRD.md", true},
			{"two levels up", "../../CHANGELOG.md", true},
			{"root absolute", "/assets/css/chrome.css", true},
			{"js scheme", "javascript:alert(1)", true},
			{"two levels up again", "../../web/public/icons/icons.css", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				if got := referenceLeavesDocsWeb(tt.ref); got != tt.want {
					t.Errorf("referenceLeavesDocsWeb(%q) = %v, want %v", tt.ref, got, tt.want)
				}
			})
		}

		if refs := resourceRefs(`.x { background: url(../outside.png); }`); len(refs) != 1 || !referenceLeavesDocsWeb(refs[0]) {
			t.Errorf("resourceRefs on a CSS url = %v, want one reference outside %s", refs, docsWebRoot)
		}
	})

	t.Run("external loads", func(t *testing.T) {
		t.Parallel()

		if got := externalLoads(`<script src="https://cdn.example/x.js"></script>`); len(got) != 1 {
			t.Errorf("externalLoads on a CDN script = %v, want one address", got)
		}

		if got := externalLoads(`<a href="https://heroicons.com/">prose</a>`); len(got) != 0 {
			t.Errorf("externalLoads on a prose link = %v, want none", got)
		}
	})

	t.Run("repeated rules", func(t *testing.T) {
		t.Parallel()

		repeated := repeatedRules([]cssSheet{
			{"one.html", `a { color: red; }`},
			{"two.html", "a {\n  color:   red;\n}"},
			{"chrome.css", `b { color: blue; }`},
		})
		if len(repeated) != 1 {
			t.Fatalf("repeatedRules = %v, want the shared a rule", repeated)
		}

		if repeated := repeatedRules([]cssSheet{{"one.html", `a { color: red; } a { color: red; }`}}); len(repeated) != 0 {
			t.Errorf("repeatedRules within one sheet = %v, want none", repeated)
		}

		repeated = repeatedRules([]cssSheet{
			{"one.html", `a { color: red; }`},
			{"chrome.css", `a { color: red; }`},
		})
		if len(repeated) != 1 {
			t.Errorf("repeatedRules page against chrome.css = %v, want one", repeated)
		}
	})

	t.Run("forbidden root tokens", func(t *testing.T) {
		t.Parallel()

		css := `:root { --accent: red; --bg: white; } :root[data-theme="dark"] { --text: black; } body { --accent: blue; }`

		if got := forbiddenRootTokens(css, nil); !slices.Equal(got, []string{accentToken, "--bg", "--text"}) {
			t.Errorf("forbiddenRootTokens = %v, want every :root token", got)
		}

		if got := forbiddenRootTokens(css, []string{accentToken}); slices.Contains(got, accentToken) {
			t.Errorf("forbiddenRootTokens with an exception = %v, want --accent allowed", got)
		}
	})
}

// TestDocsWebFurnitureIsNotDuplicated fails when a rule appears in more than one
// page or both in a page and in the shared chrome.css.
func TestDocsWebFurnitureIsNotDuplicated(t *testing.T) {
	t.Parallel()

	sheets := make([]cssSheet, 0, len(docsWebPages)+1)

	for _, page := range docsWebPages {
		sheets = append(sheets, cssSheet{page, inlineStyle(readFile(t, filepath.Join(docsWebRoot, page)))})
	}

	sheets = append(sheets, cssSheet{"assets/css/chrome.css", readFile(t, filepath.Join(docsWebRoot, "assets/css/chrome.css"))})

	for _, duplicated := range repeatedRules(sheets) {
		t.Errorf("rule repeated: %s", duplicated)
	}
}

// TestDocsWebTokensHaveOneOwner fails when a page redeclares a canonical
// palette token. There are no exceptions: a page that wants its own accent
// stops reusing the palette, and that drift is what repeated the ES chip in
// three tones (revision of ADR-018, 2026-11-05).
func TestDocsWebTokensHaveOneOwner(t *testing.T) {
	t.Parallel()

	exceptions := map[string][]string{}

	for _, page := range docsWebPages {
		css := inlineStyle(readFile(t, filepath.Join(docsWebRoot, page)))

		for _, token := range forbiddenRootTokens(css, exceptions[page]) {
			t.Errorf("%s: %s is declared outside tokens.css", page, token)
		}
	}
}

// TestDocsWebVendorFidelity fails when a vendored copy drifts from its source in
// web/.
func TestDocsWebVendorFidelity(t *testing.T) {
	t.Parallel()

	files := []struct {
		name string
		got  string
		want string
	}{
		{"icons.css", "docs/web/assets/vendor/icons/icons.css", "web/public/icons/icons.css"},
		{"app-vue.css", "docs/web/assets/vendor/app-vue.css", "web/public/task-manager/app-vue.css"},
		{"vue.global.prod.js", "docs/web/assets/vendor/vue.global.prod.js", "web/public/vendor/vue.global.prod.js"},
		{"toast-anchor.mjs", "docs/web/assets/vendor/toast-anchor/toast-anchor.mjs", "web/public/vendor/toast-anchor.mjs"},
	}

	for _, file := range files {
		if err := sameFile(file.got, file.want); err != nil {
			t.Errorf("%s: %v", file.name, err)
		}
	}

	if err := sameTree("docs/web/assets/vendor/icons/heroicons", "web/public/icons/heroicons"); err != nil {
		t.Errorf("heroicons: %v", err)
	}
}

// TestDocsWebVendorChecksReportMissingFiles proves the fidelity helpers fail on
// a missing source.
func TestDocsWebVendorChecksReportMissingFiles(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing")

	if err := sameFile(missing, "go.mod"); err == nil {
		t.Error("sameFile accepted a missing file")
	}

	if err := sameTree(missing, "docs/web/assets/vendor/icons/heroicons"); err == nil {
		t.Error("sameTree accepted a missing directory")
	}

	if err := nonEmptyFile(missing); err == nil {
		t.Error("nonEmptyFile accepted a missing file")
	}
}

// TestDocsWebLicensesPresent fails when a vendored artifact loses its license.
func TestDocsWebLicensesPresent(t *testing.T) {
	t.Parallel()

	licenses := []string{
		"docs/web/assets/vendor/icons/LICENSE",
		"docs/web/assets/vendor/vue.LICENSE",
		"docs/web/assets/vendor/toast-anchor/LICENSE",
		"docs/web/assets/vendor/microlighter/LICENSE",
	}

	for _, path := range licenses {
		if err := nonEmptyFile(path); err != nil {
			t.Errorf("%v", err)
		}
	}

	if _, err := os.Stat("docs/web/assets/vendor/microlighter/NOTICE.txt"); err != nil {
		t.Errorf("%v", err)
	}
}

// TestDocsWebIslandsExist fails when a page loses a mount point or its config
// JSON no longer parses.
func TestDocsWebIslandsExist(t *testing.T) {
	t.Parallel()

	for _, page := range docsWebPages {
		content := readFile(t, filepath.Join(docsWebRoot, page))

		for _, id := range []string{"docs-topbar", "docs-footer", "docs-config"} {
			if !strings.Contains(content, `id="`+id+`"`) {
				t.Errorf("%s: no #%s", page, id)
			}
		}

		if count := strings.Count(content, `id="docs-config"`); count != 1 {
			t.Errorf("%s: %d docs-config mounts, want one", page, count)
		}

		blocks := configScript.FindAllStringSubmatch(content, -1)
		if len(blocks) != 1 {
			t.Errorf("%s: %d docs-config scripts, want one", page, len(blocks))

			continue
		}

		var config struct {
			Topbar map[string]json.RawMessage `json:"topbar"`
		}

		if err := json.Unmarshal([]byte(blocks[0][1]), &config); err != nil {
			t.Errorf("%s: the config does not parse: %v", page, err)

			continue
		}

		if config.Topbar == nil {
			t.Errorf("%s: the config has no topbar object", page)
		}
	}

	content := readFile(t, filepath.Join(docsWebRoot, "index.html"))

	blocks := guideScript.FindAllStringSubmatch(content, -1)
	if len(blocks) != 1 {
		t.Fatalf("index.html: %d tasks-ai-en dictionaries, want one", len(blocks))
	}

	var dictionary map[string]string

	if err := json.Unmarshal([]byte(blocks[0][1]), &dictionary); err != nil {
		t.Fatalf("index.html: the dictionary does not parse: %v", err)
	}

	if len(dictionary) == 0 {
		t.Error("index.html: the dictionary is empty")
	}
}
