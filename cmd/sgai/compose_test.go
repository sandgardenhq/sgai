package main

import (
	"testing"
)

func TestSplitFrontmatterAndBody(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		wantFrontmatter string
		wantBody        string
	}{
		{
			name:            "validFrontmatterWithBody",
			content:         "---\nfoo: bar\n---\n\nbody here",
			wantFrontmatter: "---\nfoo: bar\n---",
			wantBody:        "\nbody here",
		},
		{
			name:            "validFrontmatterNoBody",
			content:         "---\nfoo: bar\n---\n",
			wantFrontmatter: "---\nfoo: bar\n---",
			wantBody:        "",
		},
		{
			name:            "noFrontmatter",
			content:         "just body content",
			wantFrontmatter: "",
			wantBody:        "just body content",
		},
		{
			name:            "frontmatterEOFNoTrailingNewline",
			content:         "---\nfoo: bar\n---",
			wantFrontmatter: "---\nfoo: bar\n---",
			wantBody:        "",
		},
		{
			name:            "malformedOnlyOpening",
			content:         "---\nfoo: bar",
			wantFrontmatter: "",
			wantBody:        "---\nfoo: bar",
		},
		{
			name:            "emptyContent",
			content:         "",
			wantFrontmatter: "",
			wantBody:        "",
		},
		{
			name:            "frontmatterWithMultipleFields",
			content:         "---\nfoo: bar\nbaz: qux\nnum: 123\n---\n\n# Heading\n\nBody content",
			wantFrontmatter: "---\nfoo: bar\nbaz: qux\nnum: 123\n---",
			wantBody:        "\n# Heading\n\nBody content",
		},
		{
			name:            "frontmatterWithEmptyBody",
			content:         "---\nkey: value\n---\n",
			wantFrontmatter: "---\nkey: value\n---",
			wantBody:        "",
		},
		{
			name:            "contentStartsWithDashesNotFrontmatter",
			content:         "-- not frontmatter\nbody content",
			wantFrontmatter: "",
			wantBody:        "-- not frontmatter\nbody content",
		},
		{
			name:            "frontmatterWithBodyContainingDashes",
			content:         "---\nmeta: data\n---\n\n---\nThis is not frontmatter\n---",
			wantFrontmatter: "---\nmeta: data\n---",
			wantBody:        "\n---\nThis is not frontmatter\n---",
		},
		{
			name:            "bodyStartsDirectlyAfterFrontmatter",
			content:         "---\nkey: value\n---\nbody here",
			wantFrontmatter: "---\nkey: value\n---",
			wantBody:        "body here",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotFrontmatter, gotBody := splitFrontmatterAndBody(tc.content)
			if gotFrontmatter != tc.wantFrontmatter {
				t.Errorf("frontmatter = %q; want %q", gotFrontmatter, tc.wantFrontmatter)
			}
			if gotBody != tc.wantBody {
				t.Errorf("body = %q; want %q", gotBody, tc.wantBody)
			}
		})
	}
}
