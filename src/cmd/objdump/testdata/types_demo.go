package main

import (
	"fmt"
	"reflect"
)

type SimpleStruct struct {
	A int
	B string
	C float64
}

type WithArray struct {
	Count int
	Items [5]int
}

type WithSlice struct {
	Name  string
	Slice []byte
}

type WithMap struct {
	ID   int
	Data map[string]int
}

type SimpleInterface interface {
	Do() error
}

type AnotherInterface interface {
	Process(string) int
}

type Implementor struct{}

func (i *Implementor) Do() error            { return nil }
func (i *Implementor) Process(s string) int { return len(s) }

type StructWithGaps struct {
	A int64
	B int8
	C int64
	D int32
	E int8
	F int64
}

type CompactStruct struct {
	A int64
	B int64
	C int64
}

func useType(t reflect.Type) {
	fmt.Printf("Type: %s, Kind: %s, Size: %d\n", t.Name(), t.Kind(), t.Size())
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			fmt.Printf("  Field[%d]: Name:%s, Type:%s, Offset:%d, Size:%d\n",
				i, f.Name, f.Type.String(), f.Offset, f.Type.Size())
		}
	}
}

func useInterface(v interface{}) {}

func main() {
	var s SimpleStruct
	var arr WithArray
	var sl WithSlice
	var mp WithMap
	var si SimpleInterface = &Implementor{}
	var ai AnotherInterface = &Implementor{}
	var gaps StructWithGaps
	var compact CompactStruct

	useInterface(s)
	useInterface(arr)
	useInterface(sl)
	useInterface(mp)
	useInterface(si)
	useInterface(ai)
	useInterface(gaps)
	useInterface(compact)

	_ = gaps
	_ = compact

	useType(reflect.TypeOf(s))
	useType(reflect.TypeOf(arr))
	useType(reflect.TypeOf(sl))
	useType(reflect.TypeOf(mp))
	useType(reflect.TypeOf(si))
	useType(reflect.TypeOf(ai))
	useType(reflect.TypeOf(gaps))
	useType(reflect.TypeOf(compact))

	fmt.Println("Done")
}
