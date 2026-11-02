package libraryquery_test

import (
	"fmt"

	"dekart/src/server/dark/libraryquery"
)

// Build a query for two categories in one city, keeping only the categories the reader may see.
func ExampleBuild() {
	table := libraryquery.Table{
		Schema:         "lib",
		Name:           "places",
		Columns:        []string{"name", "lat", "lon"},
		AreaColumns:    []libraryquery.AreaColumn{{Length: 4, Column: "city_code"}},
		CategoryColumn: "category_id",
	}
	sql, err := libraryquery.Build(table, libraryquery.Selection{
		Area:       "3273",
		Categories: []string{"cafe", "bar", "pharmacy"},
	}, []string{"cafe", "pharmacy"})
	fmt.Println(sql)
	fmt.Println(err)
	// Output:
	// SELECT "name", "lat", "lon"
	// FROM "lib"."places"
	// WHERE "city_code" = '3273'
	//   AND "category_id" IN ('cafe', 'pharmacy')
	// <nil>
}

// Hostile values are rejected, not escaped: nothing is built from them.
func ExampleBuild_hostileInput() {
	table := libraryquery.Table{Name: "places", Columns: []string{"name"}, CategoryColumn: "category_id"}
	sql, err := libraryquery.Build(table, libraryquery.Selection{
		Categories: []string{"cafe' OR '1'='1"},
	}, []string{"cafe"})
	fmt.Printf("%q\n", sql)
	fmt.Println(err)
	// Output:
	// ""
	// libraryquery: invalid id: categories
}

// Identifiers are quoted with embedded double quotes doubled.
func ExampleBuild_hostileIdentifier() {
	table := libraryquery.Table{Name: `places"; DROP TABLE places; --`, Columns: []string{`na"me`}}
	sql, err := libraryquery.Build(table, libraryquery.Selection{}, nil)
	fmt.Println(sql)
	fmt.Println(err)
	// Output:
	// SELECT "na""me"
	// FROM "places""; DROP TABLE places; --"
	// <nil>
}

// A sample is the same query with a row limit.
func ExampleBuildSample() {
	table := libraryquery.Table{Name: "roads", Columns: []string{"name"}}
	sql, _ := libraryquery.BuildSample(table, libraryquery.Selection{}, nil, 20)
	fmt.Println(sql)
	// Output:
	// SELECT "name"
	// FROM "roads"
	// LIMIT 20
}

// Intersect keeps ids present in both lists.
func ExampleIntersect() {
	fmt.Println(libraryquery.Intersect([]string{"cafe", "bar", "cafe"}, []string{"bar", "bank"}))
	// Output:
	// [bar]
}

// Area codes are 2, 4 or 7 ASCII digits.
func ExampleValidArea() {
	fmt.Println(libraryquery.ValidArea("3273"), libraryquery.ValidArea("3273' OR 1=1"), libraryquery.ValidArea(""))
	// Output:
	// true false true
}
