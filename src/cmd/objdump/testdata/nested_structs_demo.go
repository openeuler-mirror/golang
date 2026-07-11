package main

import (
	"fmt"
	"reflect"
)

type Inner struct {
	ID    int
	Name  string
	Value float64
}

type Outer struct {
	Regular   int
	Nested    Inner
	Anonymous struct {
		DeepField string
		Count     int
	}
}

var globalOuter Outer
var globalInner Inner

type CustomReader interface {
	Read(p []byte) (n int, err error)
}

type CustomWriter interface {
	Write(p []byte) (n int, err error)
}

type CustomCloser interface {
	Close() error
}

type ReadWriter interface {
	CustomReader
	CustomWriter
}

type ReadWriteCloser interface {
	ReadWriter
	CustomCloser
}

type Formatter interface {
	Format(fmt string, args ...interface{}) string
}

type Stringer interface {
	String() string
}

type myReader struct{}

func (r *myReader) Read(p []byte) (n int, err error) {
	return 0, nil
}

type myWriter struct{}

func (w *myWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

type myCloser struct{}

func (c *myCloser) Close() error {
	return nil
}

type myRWCloser struct {
	*myReader
	*myWriter
	*myCloser
}

func (rwc *myRWCloser) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (rwc *myRWCloser) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (rwc *myRWCloser) Close() error {
	return nil
}

type myFormatter struct{}

func (f *myFormatter) Format(fmtStr string, args ...interface{}) string {
	return fmt.Sprintf(fmtStr, args...)
}

type myStringer struct{}

func (s *myStringer) String() string {
	return "myStringer"
}

type myError struct{}

func (e *myError) Error() string {
	return "myError"
}

var (
	globalReader    CustomReader    = &myReader{}
	globalWriter    CustomWriter    = &myWriter{}
	globalCloser    CustomCloser    = &myCloser{}
	globalRWCloser  ReadWriteCloser = &myRWCloser{}
	globalFormatter Formatter       = &myFormatter{}
	globalStringer  Stringer        = &myStringer{}
	globalErr       error           = &myError{}
)

func useInterface(v interface{}) {}

func useType(v reflect.Type) {}

func main() {
	useInterface(struct{ A, B int }{1, 2})
	useInterface(struct{ X, Y float64 }{1.0, 2.0})
	useInterface(globalOuter)
	useInterface(globalInner)
	useInterface(fmt.Sprintf("test"))

	useInterface(globalReader)
	useInterface(globalWriter)
	useInterface(globalCloser)
	useInterface(globalRWCloser)
	useInterface(globalFormatter)
	useInterface(globalStringer)
	useInterface(globalErr)

	useType(reflect.TypeOf(globalReader))
	useType(reflect.TypeOf(globalWriter))
	useType(reflect.TypeOf(globalCloser))
	useType(reflect.TypeOf(globalRWCloser))
	useType(reflect.TypeOf(globalFormatter))
	useType(reflect.TypeOf(globalStringer))
	useType(reflect.TypeOf(globalErr))

	var i interface{} = globalRWCloser
	switch v := i.(type) {
	case CustomReader:
		if _, err := v.Read(nil); err != nil {
			fmt.Println("read error:", err)
		}
	case CustomWriter:
		if _, err := v.Write(nil); err != nil {
			fmt.Println("write error:", err)
		}
	case CustomCloser:
		if err := v.Close(); err != nil {
			fmt.Println("close error:", err)
		}
	default:
		fmt.Println("unexpected type")
	}

	_ = globalOuter.Nested.ID
	_ = globalInner.Name
}
