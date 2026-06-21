package cli

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

// newMarkdownTestCmd builds a command carrying the flags getMarkdownFromDaemon
// reads, with the given selector preset.
func newMarkdownTestCmd(selector string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().StringP("select", "s", "", "")
	cmd.Flags().StringP("find", "f", "", "")
	cmd.Flags().IntP("before", "B", 0, "")
	cmd.Flags().IntP("after", "A", 0, "")
	cmd.Flags().IntP("context", "C", 0, "")
	if selector != "" {
		_ = cmd.Flags().Set("select", selector)
	}
	return cmd
}

func TestGetMarkdownFromDaemon_MultiElementConcatenation(t *testing.T) {
	htmlData := ipc.HTMLData{
		HTMLMulti: []ipc.ElementWithHTML{
			{ElementMeta: ipc.ElementMeta{Tag: "h2"}, HTML: "<h2>One</h2>"},
			{ElementMeta: ipc.ElementMeta{Tag: "h2"}, HTML: "<h2>Two</h2>"},
		},
	}
	htmlJSON, _ := json.Marshal(htmlData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "html" {
				t.Errorf("expected cmd=html, got %s", req.Cmd)
			}
			return ipc.Response{OK: true, Data: htmlJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{daemonRunning: true, executor: exec})
	defer restore()

	md, err := getMarkdownFromDaemon(newMarkdownTestCmd("h2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(md, "One") || !strings.Contains(md, "Two") {
		t.Errorf("expected both elements converted, got: %q", md)
	}

	if strings.Contains(md, ipc.MultiElementSeparator) {
		t.Errorf("expected no %q separator between elements, got: %q", ipc.MultiElementSeparator, md)
	}

	identifier := format.FormatElementIdentifier(htmlData.HTMLMulti[1].ElementMeta, 1)
	if strings.Contains(md, identifier) {
		t.Errorf("expected no element identifier %q in markdown, got: %q", identifier, md)
	}
}

func TestGetMarkdownFromDaemon_NoMatch(t *testing.T) {
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: false, Error: "selector '.missing' matched no elements"}, nil
		},
	}

	restore := setMockFactory(&mockFactory{daemonRunning: true, executor: exec})
	defer restore()

	_, err := getMarkdownFromDaemon(newMarkdownTestCmd(".missing"))
	if !errors.Is(err, ErrNoElements) {
		t.Errorf("expected ErrNoElements, got: %v", err)
	}
}
