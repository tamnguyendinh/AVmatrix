package cobol

import (
	"reflect"
	"testing"
)

func TestParseReplacingClause(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []CopyReplacing
	}{
		{
			name: "quoted exact replacement",
			text: ` "OLD-NAME" BY "NEW-NAME" `,
			want: []CopyReplacing{{Type: "EXACT", From: "OLD-NAME", To: "NEW-NAME"}},
		},
		{
			name: "leading replacement",
			text: ` LEADING "ESP-" BY "LK-ESP-" `,
			want: []CopyReplacing{{Type: "LEADING", From: "ESP-", To: "LK-ESP-"}},
		},
		{
			name: "trailing replacement",
			text: ` TRAILING "-IN" BY "-OUT" `,
			want: []CopyReplacing{{Type: "TRAILING", From: "-IN", To: "-OUT"}},
		},
		{
			name: "basic pseudotext",
			text: ` ==WS-OLD== BY ==WS-NEW== `,
			want: []CopyReplacing{{Type: "EXACT", From: "WS-OLD", To: "WS-NEW", IsPseudotext: true}},
		},
		{
			name: "empty pseudotext deletion",
			text: ` ==REMOVE-ME== BY ==== `,
			want: []CopyReplacing{{Type: "EXACT", From: "REMOVE-ME", To: "", IsPseudotext: true}},
		},
		{
			name: "pseudotext with spaces",
			text: ` ==WORKING STORAGE== BY ==LOCAL STORAGE== `,
			want: []CopyReplacing{{Type: "EXACT", From: "WORKING STORAGE", To: "LOCAL STORAGE", IsPseudotext: true}},
		},
		{
			name: "pseudotext with embedded equals",
			text: ` ==A=B== BY ==C=D== `,
			want: []CopyReplacing{{Type: "EXACT", From: "A=B", To: "C=D", IsPseudotext: true}},
		},
		{
			name: "mixed quoted and pseudotext",
			text: ` "OLD-NAME" BY "NEW-NAME" ==DEL-PREFIX== BY ==== `,
			want: []CopyReplacing{
				{Type: "EXACT", From: "OLD-NAME", To: "NEW-NAME"},
				{Type: "EXACT", From: "DEL-PREFIX", To: "", IsPseudotext: true},
			},
		},
		{
			name: "leading modifier alongside pseudotext",
			text: ` LEADING "ESP-" BY "LK-ESP-" ==OLD-EXACT== BY ==NEW-EXACT== `,
			want: []CopyReplacing{
				{Type: "LEADING", From: "ESP-", To: "LK-ESP-"},
				{Type: "EXACT", From: "OLD-EXACT", To: "NEW-EXACT", IsPseudotext: true},
			},
		},
		{
			name: "empty input",
			text: `   `,
			want: []CopyReplacing{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseReplacingClause(tt.text)
			if got == nil {
				got = []CopyReplacing{}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseReplacingClause(%q) = %#v, want %#v", tt.text, got, tt.want)
			}
		})
	}
}
