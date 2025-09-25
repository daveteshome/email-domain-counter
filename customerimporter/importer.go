package customerimporter

import (
	"bufio"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"sort"
	"strings"
)

var ErrEmailHeaderMissing = errors.New("email header not found")

type Config struct {
	Path                   string
	EmailHeader            string
	AllowSingleLabelDomain bool
}

type DomainData struct {
	Domain           string
	CustomerQuantity int
}

type Stats struct {
	TotalRows     int
	BadRows       int
	UniqueDomains int
}

type Result struct {
	Data  []DomainData
	Stats Stats
}

type Importer struct {
	cfg Config
}

func New(cfg Config) *Importer {
	return &Importer{cfg: cfg}
}

func (i *Importer) ImportDomainData() (Result, error) {
	var res Result

	f, err := os.Open(i.cfg.Path)
	if err != nil {
		return res, err
	}
	defer f.Close()

	r := csv.NewReader(bufio.NewReaderSize(f, 256<<10))
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true
	// ReuseRecord reduces allocations per row. Safe because we consume header immediately,
	// and in the loop we fully process each record before next Read.
	r.ReuseRecord = true

	header, err := r.Read()
	if err != nil {
		return res, err
	}
	emailIdx := findHeaderIndex(header, i.cfg.EmailHeader)
	if emailIdx < 0 {
		return res, ErrEmailHeaderMissing
	}

	var counts map[string]int
	if fi, _ := f.Stat(); fi != nil {
		// assume ~40 bytes/row to estimate initial map capacity; reduces rehashing on large files.
		estRows := int(fi.Size()/40) + 1
		if estRows < 1024 {
			estRows = 1024
		}
		counts = make(map[string]int, estRows)
	} else {
		counts = make(map[string]int, 1024)
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return res, err
		}

		res.Stats.TotalRows++

		if emailIdx >= len(rec) {
			res.Stats.BadRows++
			continue
		}

		if domain, ok := extractDomain(rec[emailIdx]); ok && isValidDomain(domain, i.cfg.AllowSingleLabelDomain) {
			counts[domain]++
			continue
		}

		res.Stats.BadRows++
	}

	data := makeSortedData(counts)
	res.Data = data
	res.Stats.UniqueDomains = len(data)
	return res, nil
}

func isValidDomain(domain string, allowSingle bool) bool {
	if domain == "" || len(domain) > 253 {
		return false
	}

	dotCount := 0
	labelLen := 0

	for i := 0; i < len(domain); i++ {
		c := domain[i]
		if c == '.' {
			if labelLen == 0 || labelLen > 63 {
				return false
			}
			if domain[i-1] == '-' {
				return false
			}
			dotCount++
			labelLen = 0
			continue
		}

		isAlpha := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		isDigit := (c >= '0' && c <= '9')
		if !(isAlpha || isDigit || c == '-') {
			return false
		}

		if labelLen == 0 && c == '-' {
			return false
		}

		labelLen++
		if labelLen > 63 {
			return false
		}
	}

	if labelLen == 0 || labelLen > 63 {
		return false
	}
	if domain[len(domain)-1] == '-' {
		return false
	}
	if !allowSingle && dotCount == 0 {
		return false
	}
	return true
}

func findHeaderIndex(header []string, name string) int {
	target := strings.ToLower(strings.TrimSpace(name))
	for i, h := range header {
		if strings.ToLower(strings.TrimSpace(h)) == target {
			return i
		}
	}
	return -1
}

func extractDomain(email string) (string, bool) {
	e := email
	if n := len(e); n > 0 && (e[0] == ' ' || e[n-1] == ' ' || e[0] == '\t' || e[n-1] == '\t') {
		e = strings.TrimSpace(e)
	}

	at := strings.LastIndexByte(e, '@')
	if e == "" || at <= 0 || at+1 >= len(e) {
		return "", false
	}

	dom := e[at+1:]

	needLower := false
	for i := 0; i < len(dom); i++ {
		c := dom[i]
		if c >= 'A' && c <= 'Z' {
			needLower = true
			break
		}
	}
	if !needLower {
		return dom, true
	}

	buf := make([]byte, len(dom))
	for i := 0; i < len(dom); i++ {
		c := dom[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		buf[i] = c
	}
	return string(buf), true
}

func makeSortedData(counts map[string]int) []DomainData {
	data := make([]DomainData, 0, len(counts))
	for d, c := range counts {
		data = append(data, DomainData{
			Domain: d, CustomerQuantity: c,
		})
	}
	sort.SliceStable(data, func(i, j int) bool {
		if data[i].CustomerQuantity != data[j].CustomerQuantity {
			return data[i].CustomerQuantity > data[j].CustomerQuantity
		}
		return data[i].Domain < data[j].Domain
	})
	return data
}
