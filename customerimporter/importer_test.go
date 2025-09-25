// customerimporter/importer_test.go
package customerimporter

import (
	"os"
	"path/filepath"
	"testing"
)

// Creates a temp CSV file and returns its path.
func mustWriteTempCSV(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "in.csv")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return p
}

func TestImporter_Scenarios(t *testing.T) {
	type want struct {
		total, bad, unique int
		domains            map[string]int
	}

	tests := []struct {
		name string
		body string
		want want
	}{
		{
			name: "Handles_lowercase_header",
			body: "email,name\nAlice@Example.COM,A\nbob@example.com,B\n",
			want: want{total: 2, bad: 0, unique: 1, domains: map[string]int{"example.com": 2}},
		},
		{
			name: "Handles_uppercase_header",
			body: "EMAIL\nx@foo.com\n",
			want: want{total: 1, bad: 0, unique: 1, domains: map[string]int{"foo.com": 1}},
		},
		{
			name: "Handles_shuffled_header",
			body: "name, email ,id\nAlice,A@Foo.COM,1\nBob,b@foo.com,2\n",
			want: want{total: 2, bad: 0, unique: 1, domains: map[string]int{"foo.com": 2}},
		},
		{
			name: "Skips_bad_rows_and_tracks_stats",
			body: "email,name\n ,nobody\n@b,x\na@,y\na@b@,z\nnoatsign,k\n a@B.com ,A\nAlice@Example.com,Alice\nonlyname\n",
			want: want{
				total:  8,
				bad:    6,
				unique: 2,
				domains: map[string]int{
					"b.com":       1,
					"example.com": 1,
				},
			},
		},
		{
			name: "Sorts_by_count_then_domain",
			body: "email\nx@a.com\ny@a.com\nz@a.com\nm@b.com\nn@b.com\no@b.com\np@c.com\nq@c.com\nr@d.com\n",
			want: want{
				total:  9,
				bad:    0,
				unique: 4,
				domains: map[string]int{
					"a.com": 3,
					"b.com": 3,
					"c.com": 2,
					"d.com": 1,
				},
			},
		},
		{
			name: "Allows_ragged_rows",
			body: "name,email,city\nAlice,a@x.com,Paris\nBob\nCarol,c@y.com,NYC\n",
			want: want{
				total:  3,
				bad:    1,
				unique: 2,
				domains: map[string]int{
					"x.com": 1,
					"y.com": 1,
				},
			},
		},
		{
			name: "Counts_unique_domains",
			body: "email\na@u.com\nb@v.com\nc@w.com\n",
			want: want{
				total:  3,
				bad:    0,
				unique: 3,
				domains: map[string]int{
					"u.com": 1, "v.com": 1, "w.com": 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := mustWriteTempCSV(t, tt.body)
			imp := New(Config{Path: path, EmailHeader: "email"})
			got, err := imp.ImportDomainData()
			if err != nil {
				t.Fatalf("ImportDomainData error: %v", err)
			}
			if got.Stats.TotalRows != tt.want.total {
				t.Errorf("TotalRows got=%d want=%d", got.Stats.TotalRows, tt.want.total)
			}
			if got.Stats.BadRows != tt.want.bad {
				t.Errorf("BadRows got=%d want=%d", got.Stats.BadRows, tt.want.bad)
			}
			if got.Stats.UniqueDomains != tt.want.unique {
				t.Errorf("UniqueDomains got=%d want=%d", got.Stats.UniqueDomains, tt.want.unique)
			}
			if tt.want.domains != nil {
				for dom, expCount := range tt.want.domains {
					found := false
					for _, d := range got.Data {
						if d.Domain == dom {
							found = true
							if d.CustomerQuantity != expCount {
								t.Errorf("domain %q count got=%d want=%d", dom, d.CustomerQuantity, expCount)
							}
						}
					}
					if !found {
						t.Errorf("domain %q missing from result", dom)
					}
				}
			}
		})
	}
}

func TestImporter_SortOrderStable(t *testing.T) {
	type pair = DomainData
	tests := []struct {
		name      string
		body      string
		wantOrder []pair
	}{
		{
			name: "Ties_sorted_by_domain",
			body: "email\n" +
				"x@a.com\n" +
				"y@a.com\n" +
				"z@a.com\n" +
				"m@b.com\n" +
				"n@b.com\n" +
				"o@b.com\n" +
				"p@c.com\n" +
				"q@c.com\n" +
				"r@d.com\n",
			wantOrder: []pair{
				{Domain: "a.com", CustomerQuantity: 3},
				{Domain: "b.com", CustomerQuantity: 3},
				{Domain: "c.com", CustomerQuantity: 2},
				{Domain: "d.com", CustomerQuantity: 1},
			},
		},
		{
			name: "All_ties_same_counts",
			body: "email\n" +
				"a@x.com\n" + "b@x.com\n" +
				"a@y.com\n" + "b@y.com\n" +
				"a@z.com\n" + "b@z.com\n",
			wantOrder: []pair{
				{Domain: "x.com", CustomerQuantity: 2},
				{Domain: "y.com", CustomerQuantity: 2},
				{Domain: "z.com", CustomerQuantity: 2},
			},
		},
	}

	for _, tt := range tests {
		path := mustWriteTempCSV(t, tt.body)
		imp := New(Config{Path: path, EmailHeader: "email"})
		got, err := imp.ImportDomainData()
		if err != nil {
			t.Fatalf("[%s] ImportDomainData error: %v", tt.name, err)
		}
		if len(got.Data) != len(tt.wantOrder) {
			t.Fatalf("[%s] len mismatch: got=%d want=%d", tt.name, len(got.Data), len(tt.wantOrder))
		}
		for i := range tt.wantOrder {
			if got.Data[i] != tt.wantOrder[i] {
				t.Fatalf("[%s] order[%d] mismatch: got=%v want=%v", tt.name, i, got.Data[i], tt.wantOrder[i])
			}
		}
	}
}

func TestImporter_MissingEmailHeader(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "Missing_email_column_name",
			body: "not_email,name\n" +
				"a@x.com,Alice\n",
		},
		{
			name: "Empty_header_row",
			body: "\n" +
				"a@x.com\n",
		},
		{
			name: "No_email_column_among_others",
			body: "id,name\n1,Alice\n",
		},
	}

	for _, tt := range tests {
		path := mustWriteTempCSV(t, tt.body)
		imp := New(Config{Path: path, EmailHeader: "email"})
		_, err := imp.ImportDomainData()
		if err == nil {
			t.Fatalf("[%s] expected error, got nil", tt.name)
		}
		if err != ErrEmailHeaderMissing {
			t.Fatalf("[%s] expected ErrEmailHeaderMissing, got %v", tt.name, err)
		}
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		domain string
		ok     bool
	}{
		{"Empty_input", "", "", false},
		{"Only_spaces", "   ", "", false},
		{"Just_at_sign", "@", "", false},
		{"Missing_domain", "a@", "", false},
		{"Missing_local", "@b", "", false},
		{"Ends_with_at", "a@b@", "", false},
		{"No_at_sign", "noatsign", "", false},
		{"Trims_and_lowers", " a@B.com ", "b.com", true},
		{"Mixed_case", "Alice@Example.COM", "example.com", true},
		{"Preserves_subdomains_lowercased", "user@sub.Example.Co.UK", "sub.example.co.uk", true},
		{"Last_at_segment_wins", "a@b@c", "c", true},
	}

	for _, tt := range tests {
		got, ok := extractDomain(tt.in)
		if ok != tt.ok || got != tt.domain {
			t.Fatalf("[%s] extractDomain(%q)=(%q,%v); want (%q,%v)",
				tt.name, tt.in, got, ok, tt.domain, tt.ok)
		}
	}
}

func TestFindHeaderIndex(t *testing.T) {
	tests := []struct {
		name   string
		header []string
		field  string
		want   int
	}{
		{"Exact", []string{"email"}, "email", 0},
		{"Uppercase", []string{"EMAIL"}, "email", 0},
		{"Trimmed", []string{"  Email  "}, "email", 0},
		{"Shuffled", []string{"name", " email ", "id"}, "email", 1},
		{"Absent", []string{"name", "mail"}, "email", -1},
	}

	for _, tt := range tests {
		if got := findHeaderIndex(tt.header, tt.field); got != tt.want {
			t.Fatalf("[%s] findHeaderIndex(%v,%q)=%d; want %d", tt.name, tt.header, tt.field, got, tt.want)
		}
	}
}

func TestDomainValidation_StrictMode(t *testing.T) {
	type T struct {
		in     string
		valid  bool
		domain string
	}
	tests := []T{
		// basic invalid email forms
		{"", false, ""},
		{"   ", false, ""},
		{"@", false, ""},
		{"a@", false, ""},
		{"@b.com", false, ""},
		{"noatsign", false, ""},

		// single-label domains should be invalid in STRICT mode
		{"u@com", false, ""},
		{"a@b@c", false, ""},
		{"a@b.com", true, "b.com"},
		{"A@B.Co.UK", true, "b.co.uk"},
		{" user@Example.COM ", true, "example.com"},
		{"a@-b.com", false, ""},
		{"a@b-.com", false, ""},
		{"a@b..com", false, ""},
		{"a@.com", false, ""},
	}

	for _, tt := range tests {
		d, ok := extractDomain(tt.in)
		if !ok {
			if tt.valid {
				t.Fatalf("extractDomain(%q) unexpectedly invalid", tt.in)
			}
			continue
		}
		gotValid := isValidDomain(d, false)
		if gotValid != tt.valid {
			t.Fatalf("strict: %q domain=%q valid=%v; want %v", tt.in, d, gotValid, tt.valid)
		}
		if tt.valid && d != tt.domain {
			t.Fatalf("strict: %q domain got=%q want=%q", tt.in, d, tt.domain)
		}
	}
}

func TestDomainValidation_PermissiveMode_SingleLabelAllowed(t *testing.T) {
	type T struct {
		in     string
		valid  bool
		domain string
	}
	tests := []T{
		{"u@com", true, "com"},
		{"a@b@c", true, "c"},
		{"a@-corp", false, ""},
		{"a@corp-", false, ""},
		{"a@co..rp", false, ""},
		{"a@foo.com", true, "foo.com"},
	}

	for _, tt := range tests {
		d, ok := extractDomain(tt.in)
		if !ok {
			if tt.valid {
				t.Fatalf("extractDomain(%q) unexpectedly invalid", tt.in)
			}
			continue
		}
		gotValid := isValidDomain(d, true)
		if gotValid != tt.valid {
			t.Fatalf("perm: %q domain=%q valid=%v; want %v", tt.in, d, gotValid, tt.valid)
		}
		if tt.valid && d != tt.domain {
			t.Fatalf("perm: %q domain got=%q want=%q", tt.in, d, tt.domain)
		}
	}
}

func TestImporter_SingleLabel_Strict_vs_Permissive(t *testing.T) {
	body := "email\n" +
		"a@corp\n" +
		"b@corp\n" +
		"c@corp\n"
	path := mustWriteTempCSV(t, body)

	impStrict := New(Config{Path: path, EmailHeader: "email", AllowSingleLabelDomain: false})
	resS, err := impStrict.ImportDomainData()
	if err != nil {
		t.Fatalf("strict ImportDomainData error: %v", err)
	}
	if resS.Stats.TotalRows != 3 || resS.Stats.BadRows != 3 || len(resS.Data) != 0 {
		t.Fatalf("strict stats mismatch: %+v data=%v", resS.Stats, resS.Data)
	}

	impPerm := New(Config{Path: path, EmailHeader: "email", AllowSingleLabelDomain: true})
	resP, err := impPerm.ImportDomainData()
	if err != nil {
		t.Fatalf("perm ImportDomainData error: %v", err)
	}
	if resP.Stats.TotalRows != 3 || resP.Stats.BadRows != 0 || len(resP.Data) != 1 || resP.Data[0].Domain != "corp" || resP.Data[0].CustomerQuantity != 3 {
		t.Fatalf("perm stats/order mismatch: %+v data=%v", resP.Stats, resP.Data)
	}
}

func BenchmarkImportDomainData(b *testing.B) {
	// Path is relative to the package dir (customerimporter)
	path := filepath.Join("testdata", "benchmark10k.csv")

	if fi, err := os.Stat(path); err == nil {
		b.SetBytes(fi.Size())
	}

	imp := New(Config{Path: path, EmailHeader: "email"})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := imp.ImportDomainData(); err != nil {
			b.Fatal(err)
		}
	}
}
