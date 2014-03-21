package jpath

import (
	"testing"
)

var documentBytes = []byte(`{
	"store": {
	    "bicycle": {
	        "color": "red",
	        "price": 19.95
	    },
	    "book": [
	        {
	            "author": "Nigel Rees",
	            "category": "reference",
	            "price": 8.95,
	            "title": "Sayings of the Century"
	        },
	        {
	            "author": "Evelyn Waugh",
	            "category": "fiction",
	            "price": 12.99,
	            "title": "Sword of Honour"
	        },
	        {
	            "author": "Herman Melville",
	            "category": "fiction",
	            "isbn": "0-553-21311-3",
	            "price": 8.99,
	            "title": "Moby Dick"
	        },
	        {
	            "author": "J. R. R. Tolkien",
	            "category": "fiction",
	            "isbn": "0-395-19395-8",
	            "price": 22.99,
	            "title": "The Lord of the Rings"
	        }
	    ]
	}
}`)

var document *Jpath

func init() {
	var err error
	if document, err = NewFromBytes(documentBytes); err != nil {
		panic(err.Error())
	}
}

func TestChildSelector(t *testing.T) {
	results := document.Query("$.store.bicycle.color")

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %v", len(results))
	}

	color, ok := results[0].(string)
	if !ok {
		t.Fatal("expected string, got %T", results[0])
	}

	if color != "red" {
		t.Fatal("expected value to be 'red', got %s", color)
	}
}

func TestDescendentSelector(t *testing.T) {
	// all prices
	{
		results := document.Query("$..price")

		if len(results) != 5 {
			t.Fatalf("expected 5 results, got %v", len(results))
		}

		for _, result := range results {
			_, ok := result.(float64)
			if !ok {
				t.Fatal("expected float64, got %T", result)
			}
		}
	}

	// just book prices
	{
		results := document.Query("$..book..price")

		if len(results) != 4 {
			t.Fatalf("expected 4 results, got %v", len(results))
		}

		for _, result := range results {
			_, ok := result.(float64)
			if !ok {
				t.Fatal("expected float64, got %T", result)
			}
		}
	}
}

func TestAttributeSelector(t *testing.T) {
	results := document.Query("$.store.book[0].title")

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %v", len(results))
	}

	title, ok := results[0].(string)
	if !ok {
		t.Fatalf("expected string, got %T", title)
	}

	if title != "Sayings of the Century" {
		t.Fatalf("expected value to be 'Sayings of the Century', got %s", title)
	}
}

func TestUnmarshal(t *testing.T) {
	st := new(struct {
		Color  string    `jpath:"$.store.bicycle.color"`
		Prices []float64 `jpath:"$..price"`
	})

	if err := Unmarshal(documentBytes, st); err != nil {
		t.Fatal(err.Error())
	}

	if st.Color != "red" {
		t.Fatalf("expected Color to be '%s', got '%s'", st.Color)
	}

	if len(st.Prices) != 5 {
		t.Fatalf("expected Prices to have 5 entries, got %v", len(st.Prices))
	}
}
