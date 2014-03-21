package jpath

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// TagIdentifier determines the namespace of the tags accepted for the Unmarshaler.
var TagIdentifier = "jpath"

type filter func(f string, v interface{}) []interface{}

var filterMapping = map[string]filter{
	"\\.{2}\\w+": descendingAttributeFilter,
	"\\.\\w+":    attributeFilter,
	"\\[.+\\]":   idxFilter,
}

var matcher *regexp.Regexp
var filters []filter

// build the complete regex from all of the registered filters
func init() {
	matchers := make([]string, 0)

	for regex, filter := range filterMapping {
		matchers = append(matchers, fmt.Sprintf("(%s)", regex))
		filters = append(filters, filter)
	}
	matcher = regexp.MustCompile(strings.Join(matchers, "|"))
}

// filterForSegmentResults takes segment result pieces and finds the filter that handles
// that segment type
func filterForSegmentResults(segmentResults []string) (filter, string) {
	for i, segmentResult := range segmentResults {
		if segmentResult == "" {
			continue
		}
		return filters[i], segmentResult
	}
	return nil, ""
}

func New(m map[string]interface{}) *Jpath {
	return &Jpath{m}
}

func NewFromBytes(data []byte) (*Jpath, error) {
	m := make(map[string]interface{})
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return New(m), nil
}

type Jpath struct {
	m map[string]interface{}
}

// Query takes a JSON path and returns a list of objects that matched.
func (j *Jpath) Query(sel string) []interface{} {
	segments := matcher.FindAllStringSubmatch(sel, -1)

	// make the initial nest for the objects to scan
	objs := []interface{}{j.m}

	// loop through all segments in the path
	for _, segmentResults := range segments {
		f, s := filterForSegmentResults(segmentResults[1:])

		// make a place to store the results of the filter
		tempObjs := make([]interface{}, 0)

		// loop through
		for _, o := range objs {
			tempObjs = append(tempObjs, f(s, o)...)
		}
		objs = tempObjs
	}

	return objs
}

// Strings is a convenience method for returning a JSON path query as a slice of strings.
// Skips results that matched the query, but are not strings.
func (j *Jpath) Strings(sel string) []string {
	ret := make([]string, 0)
	for _, result := range j.Query(sel) {
		if r, ok := result.(string); ok {
			ret = append(ret, r)
		}
	}
	return ret
}

func (j *Jpath) String(sel string) (string, bool) {
	for _, result := range j.Query(sel) {
		r, ok := result.(string)
		if !ok {
			return "", false
		}
		return r, true
	}
	return "", false
}

// Bools is a convenience method for returning a JSON path query as a slice of bools.
// Skips results that matched the query, but are not bools.
func (j *Jpath) Bools(sel string) []bool {
	ret := make([]bool, 0)
	for _, result := range j.Query(sel) {
		if r, ok := result.(bool); ok {
			ret = append(ret, r)
		}
	}
	return ret
}

func (j *Jpath) Bool(sel string) (bool, bool) {
	for _, result := range j.Query(sel) {
		r, ok := result.(bool)
		if !ok {
			return false, false
		}
		return r, true
	}
	return false, false
}

// Floats is a convenience method for returning a JSON path query as a slice of floats.
// Skips results that matched the query, but are not floats.
func (j *Jpath) Floats(sel string) []float64 {
	ret := make([]float64, 0)
	for _, result := range j.Query(sel) {
		if r, ok := result.(float64); ok {
			ret = append(ret, r)
		}
	}
	return ret
}

func (j *Jpath) Float(sel string) (float64, bool) {
	for _, result := range j.Query(sel) {
		r, ok := result.(float64)
		if !ok {
			return 0, false
		}
		return r, true
	}
	return 0, false
}

//
func attributeFilter(f string, v interface{}) []interface{} {
	ret := make([]interface{}, 0)

	// shave off the prepended period
	f = f[1:]

	// attribute filter only works on maps
	msi, ok := v.(map[string]interface{})
	if !ok {
		return ret
	}

	// grab the value at the other end of the attribute if it's available
	attr, ok := msi[f]
	if !ok {
		return ret
	}

	return append(ret, attr)
}

//
func idxFilter(f string, v interface{}) []interface{} {
	// shave off the brackets
	f = f[1 : len(f)-1]

	// make sure this is a slice
	slice, ok := v.([]interface{})
	if !ok {
		return make([]interface{}, 0)
	}

	// wildcard, so we return everything inside of the slice
	if f == "*" {
		return slice
	}

	// check to see if this is a simple index access
	if i, err := strconv.Atoi(f); err == nil {
		// make sure it's not out of range
		if outOfRange(i, slice) {
			return make([]interface{}, 0)
		}

		start := i

		// for reverse access, add the length of the slice
		if i < 0 {
			start += len(slice)
		}

		return slice[start : start+1]
	}

	// check if this a slice access
	if sl := strings.Split(f, ":"); len(sl) > 0 {
		var start, end int
		var err error

		// get the initial value for the start of the slice
		if sl[0] == "" {
			start = 0
		} else if start, err = strconv.Atoi(sl[0]); err != nil || outOfRange(start, slice) {
			return make([]interface{}, 0)
		}

		// if the start is a reverse access, put it into range
		if start < 0 {
			start += len(slice)
		}

		// get the initial value for the end of the slice
		if sl[1] == "" {
			end = len(slice)
		} else if end, err = strconv.Atoi(sl[1]); err != nil || outOfRange(end, slice) {
			return make([]interface{}, 0)
		}

		// if the end is a reverse access, put it into range
		if end < 0 {
			end += len(slice)
		}

		return slice[start:end]
	}

	// some invalid filter
	return make([]interface{}, 0)
}

func outOfRange(idx int, sl []interface{}) bool {
	return math.Abs(float64(idx)) >= float64(len(sl))
}

//
func descendingAttributeFilter(f string, v interface{}) []interface{} {
	ret := make([]interface{}, 0)

	switch o := v.(type) {
	// if this is a map, then we need to see if one of it's
	// attributes matches our selector.  even if none do,
	// we need to proceed by descending deeper into the object
	// for a match.
	case map[string]interface{}:
		// check to see if a attribute matches the selector, if it
		// does then we just return right now.
		if c, ok := o[f[2:]]; ok {
			return append(ret, c)
		}

		// recursively keep checking for a match
		for _, val := range o {
			ret = append(ret, descendingAttributeFilter(f, val)...)
		}

	// if this a slice, then we need to check each member
	// to see if it matches our selector
	case []interface{}:
		// recursively keep checking for a match
		for _, val := range o {
			ret = append(ret, descendingAttributeFilter(f, val)...)
		}
	}

	return ret
}

// Unmarshal functions similarly to json.Unmarshal, except reads the jpath tag
// and marshals results into struct fields based on the results of the queries.
func Unmarshal(data []byte, v interface{}) error {
	d, err := NewFromBytes(data)
	if err != nil {
		return err
	}
	return unmarshal(d, v)
}

func unmarshal(d *Jpath, v interface{}) error {
	vt := reflect.TypeOf(v).Elem()
	vv := reflect.ValueOf(v).Elem()

	if vt.Kind() != reflect.Struct {
		return fmt.Errorf("v must be a struct, got %T", v)
	}

	// we're going to loop through each field in the struct and
	// extract the json path from the tags, query the document with
	// those tags, then set the results on the fields.
	for i := 0; i < vt.NumField(); i++ {
		fieldType := vt.Field(i)
		fieldValue := vv.Field(i)
		fieldKind := fieldValue.Kind()

		// if we can't update this field, we're done with it
		if !fieldValue.CanSet() {
			continue
		}

		// extract the json path from the tag
		tag := fieldType.Tag.Get(TagIdentifier)
		if tag == "" {
			continue
		}

		// eventually we will support marshaling values out, which will be
		// delimited by a space, so this is in preparation for that.
		tagPieces := strings.Split(tag, " ")
		query := tagPieces[0]

		// query the document with the json path
		results := d.Query(query)

		// if this isn't a slice we're unmarshaling into, then just take the
		// first value and set it and we're done.
		if fieldKind != reflect.Slice {
			resultValue := reflect.ValueOf(results[0])

			// ensure that the kinds line up before we set
			if resultValue.Kind() != fieldKind {
				return fmt.Errorf("%s - value of type %s is not assignable to type %s",
					fieldType.Name, resultValue.Type(), fieldType.Type)
			}

			fieldValue.Set(resultValue)
			continue
		}

		// make a slice with the same type as the field
		sl := reflect.MakeSlice(fieldType.Type, 0, len(results))

		// loop through results and append them to the slice we created
		for _, result := range results {
			resultValue := reflect.ValueOf(result)

			// TODO: Find a way to sniff the type of items in the slice and just do a check to see
			// if the kind will fit inside. in the meantime, we're just going to trap panics.
			var err error
			func() {
				defer func() {
					if recover() != nil {
						err = fmt.Errorf("%s - value of type %s will not fit into slice of %s",
							fieldType.Name, resultValue.Type(), fieldType.Type)
					}
				}()

				sl = reflect.Append(sl, resultValue)
			}()

			if err != nil {
				return err
			}
		}

		// finally set the slice
		fieldValue.Set(sl)
	}

	return nil
}
