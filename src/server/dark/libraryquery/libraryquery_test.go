package libraryquery_test

import (
	"errors"
	"strings"
	"testing"

	"dekart/src/server/dark/libraryquery"
)

var places = libraryquery.Table{
	Schema:  "lib",
	Name:    "tempat_usaha",
	Columns: []string{"nama", "brand", "lat", "lon"},
	AreaColumns: []libraryquery.AreaColumn{
		{Length: 2, Column: "kode_provinsi"},
		{Length: 4, Column: "kode_kabkota"},
		{Length: 7, Column: "kode_kecamatan"},
	},
	CategoryColumn: "category_id",
	BrandColumn:    "brand_id",
}

var boundaries = libraryquery.Table{
	Schema:      "lib",
	Name:        "kecamatan_bps_2024",
	Columns:     []string{"kode", "nama", "geometry"},
	AreaColumns: []libraryquery.AreaColumn{{Length: 4, Column: "kode_kabkota"}},
}

func TestBuildFiltersByAreaCategoryAndBrand(t *testing.T) {
	got, err := libraryquery.Build(places, libraryquery.Selection{
		Area:       "3273",
		Categories: []string{"kedai_kopi", "kafe"},
		Brands:     []string{"starbucks", "kopi_kenangan"},
	}, []string{"kafe", "kedai_kopi", "restoran"})
	if err != nil {
		t.Fatal(err)
	}
	want := `SELECT "nama", "brand", "lat", "lon"
FROM "lib"."tempat_usaha"
WHERE "kode_kabkota" = '3273'
  AND "category_id" IN ('kafe', 'kedai_kopi')
  AND "brand_id" IN ('kopi_kenangan', 'starbucks')`
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildWithoutFiltersHasNoWhere(t *testing.T) {
	got, err := libraryquery.Build(boundaries, libraryquery.Selection{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "SELECT \"kode\", \"nama\", \"geometry\"\nFROM \"lib\".\"kecamatan_bps_2024\""
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildPicksAreaColumnByCodeLength(t *testing.T) {
	for code, column := range map[string]string{"32": "kode_provinsi", "3273": "kode_kabkota", "3273010": "kode_kecamatan"} {
		got, err := libraryquery.Build(places, libraryquery.Selection{Area: code, Categories: []string{"kafe"}}, []string{"kafe"})
		if err != nil {
			t.Fatalf("%s: %v", code, err)
		}
		if !strings.Contains(got, `"`+column+`" = '`+code+`'`) {
			t.Fatalf("%s: wrong column in\n%s", code, got)
		}
	}
}

func TestBuildIsDeterministicForReorderedAndDuplicatedLists(t *testing.T) {
	a, err := libraryquery.Build(places, libraryquery.Selection{Categories: []string{"kafe", "bar", "kafe"}}, []string{"bar", "kafe"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := libraryquery.Build(places, libraryquery.Selection{Categories: []string{"bar", "kafe"}}, []string{"kafe", "bar"})
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("same selection produced different SQL:\n%s\n%s", a, b)
	}
}

// Category intersection: the filter holds only categories that are both requested and allowed.
func TestBuildKeepsOnlyAllowedCategories(t *testing.T) {
	got, err := libraryquery.Build(places, libraryquery.Selection{Categories: []string{"kafe", "bar", "apotek"}}, []string{"kafe", "apotek", "bank"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"category_id" IN ('apotek', 'kafe')`) {
		t.Fatalf("unexpected filter:\n%s", got)
	}
	if strings.Contains(got, "bar") || strings.Contains(got, "bank") {
		t.Fatalf("category outside the intersection leaked into SQL:\n%s", got)
	}
}

func TestBuildRefusals(t *testing.T) {
	cases := []struct {
		name      string
		table     libraryquery.Table
		selection libraryquery.Selection
		allowed   []string
		want      error
	}{
		{"no category on categorised table", places, libraryquery.Selection{}, []string{"kafe"}, libraryquery.ErrCategoryRequired},
		{"nothing allowed", places, libraryquery.Selection{Categories: []string{"kafe"}}, nil, libraryquery.ErrNoAllowedCategory},
		{"requested outside allowed", places, libraryquery.Selection{Categories: []string{"bar"}}, []string{"kafe"}, libraryquery.ErrNoAllowedCategory},
		{"categories on plain table", boundaries, libraryquery.Selection{Categories: []string{"kafe"}}, []string{"kafe"}, libraryquery.ErrCategoriesUnsupported},
		{"brands on plain table", boundaries, libraryquery.Selection{Brands: []string{"starbucks"}}, nil, libraryquery.ErrBrandsUnsupported},
		{"area level without a column", boundaries, libraryquery.Selection{Area: "32"}, nil, libraryquery.ErrAreaLevelUnsupported},
		{"no columns", libraryquery.Table{Name: "t"}, libraryquery.Selection{}, nil, libraryquery.ErrNoColumns},
		{"empty table name", libraryquery.Table{Columns: []string{"a"}}, libraryquery.Selection{}, nil, libraryquery.ErrInvalidIdentifier},
		{"empty column name", libraryquery.Table{Name: "t", Columns: []string{""}}, libraryquery.Selection{}, nil, libraryquery.ErrInvalidIdentifier},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := libraryquery.Build(c.table, c.selection, c.allowed)
			if !errors.Is(err, c.want) {
				t.Fatalf("got error %v, want %v", err, c.want)
			}
			if got != "" {
				t.Fatalf("partial SQL returned with an error: %q", got)
			}
		})
	}
}

// hostile is input that tries to break out of a literal, comment out the rest of the statement or smuggle in a second one.
var hostile = []string{
	`'; DROP TABLE lib.tempat_usaha; --`,
	`kafe' OR '1'='1`,
	`kafe'--`,
	`kafe\`,
	`kafe\'`,
	"kafe\x00",
	"kafe\n",
	"kafe\r\n",
	"kafe\t",
	" kafe",
	"kafe ",
	`KAFE`,
	`kafe;`,
	`kafe)`,
	`kafe") --`,
	`kafe/**/`,
	`$1`,
	`%`,
	`kafé`,
	"аfe",  // Cyrillic a
	`ｋａｆｅ`, // full-width letters
	``,
	strings.Repeat("a", 65),
}

func TestBuildRejectsHostileCategoryAndBrandIDs(t *testing.T) {
	for _, value := range hostile {
		for _, selection := range []libraryquery.Selection{
			{Categories: []string{value}},
			{Categories: []string{"kafe"}, Brands: []string{value}},
			{Categories: []string{"kafe", value}},
		} {
			got, err := libraryquery.Build(places, selection, []string{"kafe", value})
			if !errors.Is(err, libraryquery.ErrInvalidID) {
				t.Errorf("%q: got error %v, want ErrInvalidID (sql %q)", value, err, got)
			}
			if got != "" {
				t.Errorf("%q: SQL returned for hostile input: %q", value, got)
			}
		}
	}
}

func TestBuildRejectsHostileAreaCodes(t *testing.T) {
	areas := []string{
		`3273' OR '1'='1`, `3273'; DROP TABLE x; --`, "3273\n", "3273\x00", " 3273", "3273 ", `32\73`, `３２７３`,
		`3`, `327`, `32730`, `327301`, `32730101`, `-3273`, `3.273`, `0x3273`, `32;`, `32)--`,
	}
	for _, area := range areas {
		got, err := libraryquery.Build(places, libraryquery.Selection{Area: area, Categories: []string{"kafe"}}, []string{"kafe"})
		if !errors.Is(err, libraryquery.ErrInvalidArea) {
			t.Errorf("%q: got error %v, want ErrInvalidArea (sql %q)", area, err, got)
		}
		if got != "" {
			t.Errorf("%q: SQL returned for hostile input: %q", area, got)
		}
	}
}

// The allowed set is trusted input from the caller's records, but an id that is not well formed must still never be emitted.
func TestBuildNeverEmitsMalformedAllowedIDs(t *testing.T) {
	_, err := libraryquery.Build(places, libraryquery.Selection{Categories: []string{"kafe"}}, []string{`kafe' OR '1'='1`})
	if !errors.Is(err, libraryquery.ErrNoAllowedCategory) {
		t.Fatalf("got %v", err)
	}
}

func TestBuildQuotesHostileIdentifiersInsteadOfRejectingThem(t *testing.T) {
	table := libraryquery.Table{
		Schema:      `li"b`,
		Name:        `t"; DROP TABLE x; --`,
		Columns:     []string{`a"b`, `semi;colon`},
		AreaColumns: []libraryquery.AreaColumn{{Length: 2, Column: `p"; --`}},
	}
	got, err := libraryquery.Build(table, libraryquery.Selection{Area: "32"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `SELECT "a""b", "semi;colon"
FROM "li""b"."t""; DROP TABLE x; --"
WHERE "p""; --" = '32'`
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildRejectsUnusableIdentifiers(t *testing.T) {
	for _, name := range []string{"a\x00b", strings.Repeat("a", 129)} {
		for _, table := range []libraryquery.Table{
			{Name: name, Columns: []string{"a"}},
			{Name: "t", Columns: []string{name}},
			{Schema: name, Name: "t", Columns: []string{"a"}},
			{Name: "t", Columns: []string{"a"}, AreaColumns: []libraryquery.AreaColumn{{Length: 2, Column: name}}},
		} {
			selection := libraryquery.Selection{Area: "32"}
			if _, err := libraryquery.Build(table, selection, nil); !errors.Is(err, libraryquery.ErrInvalidIdentifier) {
				t.Errorf("%q in %+v: got %v, want ErrInvalidIdentifier", name, table, err)
			}
		}
	}
}

func TestBuildLimitsListSize(t *testing.T) {
	many := make([]string, libraryquery.MaxListSize+1)
	for i := range many {
		many[i] = "c" + strings.Repeat("1", i%30) + string(rune('a'+i%26))
	}
	_, err := libraryquery.Build(places, libraryquery.Selection{Categories: many}, many)
	if !errors.Is(err, libraryquery.ErrTooManyValues) {
		t.Fatalf("got %v", err)
	}
}

func TestBuildSampleAddsLimit(t *testing.T) {
	got, err := libraryquery.BuildSample(boundaries, libraryquery.Selection{Area: "3273"}, nil, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, "\nLIMIT 20") {
		t.Fatalf("missing limit:\n%s", got)
	}
}

func TestBuildSampleRejectsLimitsOutsideRange(t *testing.T) {
	for _, limit := range []int{-1, 0, libraryquery.MaxSampleLimit + 1} {
		if _, err := libraryquery.BuildSample(boundaries, libraryquery.Selection{}, nil, limit); !errors.Is(err, libraryquery.ErrInvalidLimit) {
			t.Errorf("limit %d: got %v", limit, err)
		}
	}
}

func TestIntersectIsSortedDeduplicatedAndDropsMalformedIDs(t *testing.T) {
	got := libraryquery.Intersect([]string{"b", "a", "b", "c", `x'y`}, []string{"a", "b", `x'y`})
	if strings.Join(got, ",") != "a,b" {
		t.Fatalf("got %v", got)
	}
	if got := libraryquery.Intersect(nil, []string{"a"}); got == nil || len(got) != 0 {
		t.Fatalf("empty intersection must be a non-nil empty slice, got %#v", got)
	}
}

func TestIntersectDoesNotModifyItsArguments(t *testing.T) {
	requested := []string{"b", "a"}
	allowed := []string{"b", "a"}
	libraryquery.Intersect(requested, allowed)
	if requested[0] != "b" || allowed[0] != "b" {
		t.Fatalf("arguments were reordered: %v %v", requested, allowed)
	}
}

func TestValidators(t *testing.T) {
	for _, id := range []string{"kafe", "kedai_kopi", "a1", strings.Repeat("a", 64)} {
		if !libraryquery.ValidID(id) {
			t.Errorf("%q should be valid", id)
		}
	}
	for _, id := range hostile {
		if libraryquery.ValidID(id) {
			t.Errorf("%q should be invalid", id)
		}
	}
	for _, code := range []string{"", "32", "3273", "3273010"} {
		if !libraryquery.ValidArea(code) {
			t.Errorf("%q should be valid", code)
		}
	}
	for _, code := range []string{"3", "327", "3273\n", "３２"} {
		if libraryquery.ValidArea(code) {
			t.Errorf("%q should be invalid", code)
		}
	}
}
