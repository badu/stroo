package halp

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

const (
	bracket      = "{"
	closeBracket = "}"
	comma        = ","
	colon        = ":"
	semicolon    = ";"
	space        = " "
	invalid      = "<invalid>"
	nihil        = "nil"
	newline      = "\n"
	ptrsign      = "&"
)

var (
	packageRegexp   = regexp.MustCompile("\\b[a-zA-Z_]+[a-zA-Z_0-9]+\\.")
	shortnameRegexp = regexp.MustCompile(`\s*([,;{}()])\s*`)
	printerType     = reflect.TypeOf((*Printer)(nil)).Elem()
	DefaultOpts     = PrintOpts{
		SkipPackage: true,
		SkipPrivate: true,
		SkipZero:    true,
		Excludes:    []*regexp.Regexp{regexp.MustCompile(`^(XXX_.*)$`)},
		Separator:   " ",
	}
)

type (
	Printer interface {
		Print(w io.Writer)
	}
	PrintOpts struct {
		MainPackage string
		Separator   string
		WillCompact bool
		WithFuncs   bool
		SkipPackage bool
		SkipPrivate bool
		SkipZero    bool
		Excludes    []*regexp.Regexp
		FieldFilter func(reflect.StructField, reflect.Value) bool
	}
	ptrsMap struct {
		ptrs      []uintptr
		cachedPtr []uintptr
	}
	visitor struct {
		io.Writer
		ptrsMap
		curDepth      int
		opts          *PrintOpts
		currPtrName   string
		packageRegexp *regexp.Regexp
	}
)

func (s *visitor) Write(content string) {
	_, _ = s.Writer.Write([]byte(content))
}

func (s *visitor) indent() {
	if !s.opts.WillCompact {
		s.Write(strings.Repeat("  ", s.curDepth))
	}
}

func (s *visitor) addNewline() {
	if s.currPtrName != "" {
		if s.opts.WillCompact {
			s.Write(fmt.Sprintf("/*%s*/", s.currPtrName))
		} else {
			s.Write(fmt.Sprintf(" // %s\n", s.currPtrName))
		}
		s.currPtrName = ""
		return
	}
	if !s.opts.WillCompact {
		s.Write(newline)
	}
}

func (s *visitor) visitType(value reflect.Value) {
	typeName := value.Type().String()
	if s.opts.SkipPackage {
		typeName = packageRegexp.ReplaceAllLiteralString(typeName, "")
	} else if s.packageRegexp != nil {
		typeName = s.packageRegexp.ReplaceAllLiteralString(typeName, "")
	}
	if s.opts.WillCompact {
		typeName = shortnameRegexp.ReplaceAllString(typeName, "$1")
	}
	s.Write(typeName)
}

func (s *visitor) visitSlice(value reflect.Value) {
	s.visitType(value)
	if value.Len() == 0 {
		s.Write(bracket + closeBracket)
		if s.opts.WillCompact {
			s.Write(semicolon)
		}
		s.addNewline()
		return
	}
	s.Write(bracket)
	s.addNewline()
	s.curDepth++
	for i := 0; i < value.Len(); i++ {
		s.indent()
		s.visit(value.Index(i))
		if !s.opts.WillCompact || i < value.Len()-1 {
			s.Write(comma)
		}
		s.addNewline()
	}
	s.curDepth--
	s.indent()
	s.Write(closeBracket)
}

func (s *visitor) visitStruct(value reflect.Value) {
	first := false
	for i := 0; i < value.NumField(); i++ {
		field := value.Type().Field(i)
		if s.opts.SkipPrivate && field.PkgPath != "" {
			continue
		}
		willContinue := false
		for _, regex := range s.opts.Excludes {
			if regex.MatchString(field.Name) {
				willContinue = true
				break
			}
		}
		if willContinue {
			continue
		}
		if s.opts.FieldFilter != nil && !s.opts.FieldFilter(field, value.Field(i)) {
			continue
		}
		if s.opts.SkipZero && isZero(value.Field(i)) {
			continue
		}

		if !first {
			s.visitType(value)
			s.Write(bracket)
			s.addNewline()
			s.curDepth++
			first = true
		}
		s.indent()
		s.Write(field.Name)
		if s.opts.WillCompact {
			s.Write(colon)
		} else {
			s.Write(colon + space)
		}
		s.visit(value.Field(i))
		if !s.opts.WillCompact || i < value.NumField()-1 {
			s.Write(comma)
		}
		s.addNewline()
	}
	if first {
		s.curDepth--
		s.indent()
		s.Write(closeBracket)
		return
	}
	s.visitType(value)
	s.Write(bracket + closeBracket)
}

func (s *visitor) visitMap(value reflect.Value) {
	s.visitType(value)
	s.Write(bracket)
	s.addNewline()
	s.curDepth++
	keys := value.MapKeys()
	sort.Sort(sortKeys{keys: keys, options: s.opts})
	for idx, key := range keys {
		s.indent()
		s.visit(key)
		if s.opts.WillCompact {
			s.Write(colon)
		} else {
			s.Write(colon + space)
		}
		s.visit(value.MapIndex(key))
		if !s.opts.WillCompact || idx < len(keys)-1 {
			s.Write(comma)
		}
		s.addNewline()
	}
	s.curDepth--
	s.indent()
	s.Write(closeBracket)
}

func (s *visitor) visitFunc(v reflect.Value) {
	parts := strings.Split(runtime.FuncForPC(v.Pointer()).Name(), "/")
	name := parts[len(parts)-1]
	if strings.Count(name, ".") > 1 {
		s.visitType(v)
		return
	}
	if s.opts.SkipPackage {
		name = packageRegexp.ReplaceAllLiteralString(name, "")
	} else if s.packageRegexp != nil {
		name = s.packageRegexp.ReplaceAllLiteralString(name, "")
	}
	if s.opts.WillCompact {
		name = shortnameRegexp.ReplaceAllString(name, "$1")
	}
	s.Write(name)
}

func (s *visitor) visitValue(value reflect.Value) {
	out := new(bytes.Buffer)
	printFunc := value.MethodByName("Print")
	printFunc.Call([]reflect.Value{reflect.ValueOf(out)})
	s.visitType(value)
	if s.opts.WillCompact {
		s.Write(out.String())
		return
	}
	var err error
	firstLine := true
	for err == nil {
		var lineBytes []byte
		lineBytes, err = out.ReadBytes('\n')
		line := strings.TrimRight(string(lineBytes), space+newline)
		if err != nil && err != io.EOF {
			break
		}
		if firstLine {
			firstLine = false
		} else {
			s.indent()
		}
		s.Write(line)
		if err == io.EOF {
			return
		}
		s.addNewline()
	}
	panic(err)
}

func (s *visitor) print(value interface{}) {
	if value == nil {
		s.Write(nihil)
		return
	}
	v := reflect.ValueOf(value)
	s.visit(v)
}

func (s *visitor) visitPtr(value reflect.Value) bool {
	pointerName, firstTime := s.ptrNamed(value)
	if pointerName == "" {
		return true
	}
	if firstTime {
		s.currPtrName = pointerName
		return true
	}
	s.Write(pointerName)
	return false
}

func (s *visitor) visit(value reflect.Value) {
	if value.Kind() == reflect.Ptr && value.IsNil() {
		s.Write(nihil)
		return
	}
	realValue := underlying(value)
	if realValue.Type().Implements(printerType) {
		if s.visitPtr(realValue) {
			s.visitValue(realValue)
		}
		return
	}

	switch realValue.Kind() {
	case reflect.Invalid:
		s.Write(invalid)
	case reflect.Bool:
		boolValue := "false"
		if realValue.Bool() {
			boolValue = "true"
		}
		s.Write(boolValue)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		s.Write(strconv.FormatInt(realValue.Int(), 10))
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		s.Write(strconv.FormatUint(realValue.Uint(), 10))
	case reflect.Float32:
		s.Write(strconv.FormatFloat(realValue.Float(), 'g', -1, 32))
	case reflect.Float64:
		s.Write(strconv.FormatFloat(realValue.Float(), 'g', -1, 64))
	case reflect.Complex64, reflect.Complex128:
		precision := 32
		if realValue.Kind() == reflect.Complex128 {
			precision = 64
		}
		s.Write("complex")
		s.Write(strconv.FormatInt(int64(precision*2), 10))
		r := real(realValue.Complex())
		s.Write("(")
		s.Write(strconv.FormatFloat(r, 'g', -1, precision))
		i := imag(realValue.Complex())
		if i >= 0 {
			s.Write("+")
		}
		s.Write(strconv.FormatFloat(i, 'g', -1, precision))
		s.Write("i)")
	case reflect.String:
		s.Write(strconv.Quote(realValue.String()))
	case reflect.Slice:
		if realValue.IsNil() {
			s.Write(nihil)
			break
		}
		fallthrough
	case reflect.Array:
		if s.visitPtr(realValue) {
			s.visitSlice(realValue)
		}
	case reflect.Interface:
		if realValue.IsNil() {
			s.Write(nihil)
		}
	case reflect.Ptr:
		if s.visitPtr(realValue) {
			s.Write(ptrsign)
			s.visit(realValue.Elem())
		}
	case reflect.Map:
		if s.visitPtr(realValue) {
			s.visitMap(realValue)
		}
	case reflect.Struct:
		s.visitStruct(realValue)
	case reflect.Func:
		if s.opts.WithFuncs {
			s.visitFunc(realValue)
		}
	default:
		if realValue.CanInterface() {
			_, _ = fmt.Fprintf(s.Writer, "%v", realValue.Interface())
		} else {
			_, _ = fmt.Fprintf(s.Writer, "%v", realValue.String())
		}
	}
}

func (s *visitor) hasCached(ptr uintptr) bool {
	for _, addr := range s.cachedPtr {
		if addr == ptr {
			return false
		}
	}
	s.cachedPtr = append(s.cachedPtr, ptr)
	return true
}

func (s *visitor) ptrNamed(value reflect.Value) (string, bool) {
	if !isPtr(value) {
		return "", false
	}
	for idx, ptr := range s.ptrs {
		if value.Pointer() == ptr {
			return fmt.Sprintf("p%d", idx), s.hasCached(value.Pointer())
		}
	}
	return "", false
}

func newVisitor(value interface{}, options *PrintOpts, writer io.Writer) *visitor {
	result := &visitor{
		opts:    options,
		ptrsMap: ptrsMap{ptrs: MapReusedPointers(reflect.ValueOf(value))},
		Writer:  writer,
	}
	if options.MainPackage != "" {
		result.packageRegexp = regexp.MustCompile(fmt.Sprintf("\\b%s\\.", options.MainPackage))
	}

	return result
}

func Print(value ...interface{}) {
	(&DefaultOpts).Print(value...)
}

func SPrint(value ...interface{}) string {
	return (&DefaultOpts).SPrint(value...)
}

func (o PrintOpts) Print(values ...interface{}) {
	for i, value := range values {
		state := newVisitor(value, &o, os.Stdout)
		if i > 0 {
			state.Write(o.Separator)
		}
		state.print(value)
	}
	_, _ = os.Stdout.Write([]byte(newline))
}

func (o PrintOpts) SPrint(values ...interface{}) string {
	out := new(bytes.Buffer)
	for i, value := range values {
		if i > 0 {
			out.Write([]byte(o.Separator))
		}
		state := newVisitor(value, &o, out)
		state.print(value)
	}
	return out.String()
}

type sortKeys struct {
	keys    []reflect.Value
	options *PrintOpts
}

func (s sortKeys) Len() int {
	return len(s.keys)
}

func (s sortKeys) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
}

func (s sortKeys) Less(i, j int) bool {
	outI := new(bytes.Buffer)
	outJ := new(bytes.Buffer)
	newVisitor(s.keys[i], s.options, outI).visit(s.keys[i])
	newVisitor(s.keys[j], s.options, outJ).visit(s.keys[j])
	return outI.String() < outJ.String()
}

func MapReusedPointers(value reflect.Value) []uintptr {
	pMap := &ptrsMap{}
	pMap.visit(value)
	return pMap.cachedPtr
}

func (m *ptrsMap) visit(value reflect.Value) {
	if value.Kind() == reflect.Invalid {
		return
	}
	if isPtr(value) && value.Pointer() != 0 {
		if m.isReused(value.Pointer()) {
			return
		}
	}

	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		numEntries := value.Len()
		for i := 0; i < numEntries; i++ {
			m.visit(value.Index(i))
		}
	case reflect.Interface:
		m.visit(value.Elem())
	case reflect.Ptr:
		m.visit(value.Elem())
	case reflect.Map:
		keys := value.MapKeys()
		sort.Sort(sortKeys{keys: keys, options: &DefaultOpts})
		for _, key := range keys {
			m.visit(value.MapIndex(key))
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			m.visit(value.Field(i))
		}
	}
}

func (m *ptrsMap) isReused(ptr uintptr) bool {
	for _, have := range m.cachedPtr {
		if ptr == have {
			return true
		}
	}
	for _, seen := range m.ptrs {
		if ptr == seen {
			m.cachedPtr = append(m.cachedPtr, ptr)
			return true
		}
	}
	m.ptrs = append(m.ptrs, ptr)
	return false
}

func underlying(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	return v
}

func isPtr(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return true
	}
	return false
}

func isZero(value reflect.Value) bool {
	return (isPtr(value) && value.IsNil()) ||
		(value.IsValid() && value.CanInterface() && reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface()))
}

type OptName string

const (
	FloatPrecision   OptName = "FloatPrecision"
	MaxDiff          OptName = "MaxDiff"
	MaxDepth         OptName = "MaxDepth"
	LogErrors        OptName = "LogErrors"
	LookupUnexported OptName = "LookupUnexported"
	IncludeNilSlices OptName = "IncludeNilSlices"

	DefaultFloatPrecision int = 10
	DefaultMaxDiff        int = 10
)

type CompareOptions struct {
	Name  OptName
	Value interface{}
}
type cmpResult struct {
	diff             []string
	buff             []string
	floatFormat      string
	FloatPrecision   int
	MaxDiff          int
	MaxDepth         int
	LogErrors        bool
	LookupUnexported bool
	IncludeNilSlices bool
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()

func Equal(actual, expected interface{}, opts ...CompareOptions) []string {
	actualValue := reflect.ValueOf(actual)
	expectedValue := reflect.ValueOf(expected)
	result := &cmpResult{
		diff: []string{},
		buff: []string{},
	}
	for _, opt := range opts {
		switch opt.Name {
		case FloatPrecision:
			if fv, ok := opt.Value.(int); ok {
				result.FloatPrecision = fv
			}
		case MaxDiff:
			if fv, ok := opt.Value.(int); ok {
				result.MaxDiff = fv
			}
		case MaxDepth:
			if fv, ok := opt.Value.(int); ok {
				result.MaxDepth = fv
			}
		case LogErrors:
			if fv, ok := opt.Value.(bool); ok {
				result.LogErrors = fv
			}
		case LookupUnexported:
			if fv, ok := opt.Value.(bool); ok {
				result.LookupUnexported = fv
			}
		case IncludeNilSlices:
			if fv, ok := opt.Value.(bool); ok {
				result.IncludeNilSlices = fv
			}
		}
	}
	if result.FloatPrecision <= 0 {
		result.FloatPrecision = DefaultFloatPrecision
	}
	if result.MaxDiff <= 0 {
		result.MaxDiff = DefaultMaxDiff
	}
	result.floatFormat = fmt.Sprintf("%%.%df", result.FloatPrecision)
	if actual == nil && expected == nil {
		return nil
	} else if actual == nil && expected != nil {
		result.keep("<nil pointer>", expected)
	} else if actual != nil && expected == nil {
		result.keep(actual, "<nil pointer>")
	}
	if len(result.diff) > 0 {
		return result.diff
	}
	result.equals(actualValue, expectedValue, 0)
	if len(result.diff) > 0 {
		return result.diff
	}
	return nil
}

func (r *cmpResult) equals(actual, expected reflect.Value, level int) {
	if r.MaxDepth > 0 && level > r.MaxDepth {
		logError(r, errors.New("recurred to MaxDepth"))
		return
	}

	if !actual.IsValid() || !expected.IsValid() {
		if actual.IsValid() && !expected.IsValid() {
			r.keep(actual.Type(), "<nil pointer>")
		} else if !actual.IsValid() && expected.IsValid() {
			r.keep("<nil pointer>", expected.Type())
		}
		return
	}

	actualType := actual.Type()
	expectedType := expected.Type()
	if actualType != expectedType {
		r.keep(actualType, expectedType)
		logError(r, errors.New("cannot compare types (different reflect.Type)"))
		return
	}

	actualKind := actual.Kind()
	expectedKind := expected.Kind()

	actualDeref := actualKind == reflect.Ptr || actualKind == reflect.Interface
	expectedDeref := expectedKind == reflect.Ptr || expectedKind == reflect.Interface

	if actualType.Implements(errorType) && expectedType.Implements(errorType) {
		if (!actualDeref || !actual.IsNil()) && (!expectedDeref || !expected.IsNil()) {
			actualErrorMethod := actual.MethodByName("Error").Call(nil)[0].String()
			expectedErrorMethod := expected.MethodByName("Error").Call(nil)[0].String()
			if actualErrorMethod != expectedErrorMethod {
				r.keep(actualErrorMethod, expectedErrorMethod)
				return
			}
		}
	}

	if actualDeref || expectedDeref {
		if actualDeref {
			actual = actual.Elem()
		}
		if expectedDeref {
			expected = expected.Elem()
		}
		r.equals(actual, expected, level+1)
		return
	}

	switch actualKind {

	case reflect.Struct:
		if actualEqualMethod := actual.MethodByName("Equal"); actualEqualMethod.IsValid() && actualEqualMethod.CanInterface() {
			funcType := actualEqualMethod.Type()
			if funcType.NumIn() == 1 && funcType.In(0) == expectedType {
				equalCallResults := actualEqualMethod.Call([]reflect.Value{expected})
				if !equalCallResults[0].Bool() {
					r.keep(actual, expected)
				}
				return
			}
		}
		for i := 0; i < actual.NumField(); i++ {
			if actualType.Field(i).PkgPath != "" && !r.LookupUnexported {
				continue
			}
			if actualType.Field(i).Tag.Get("skipEqual") == "-" {
				continue
			}

			r.push(actualType.Field(i).Name)
			actualField := actual.Field(i)
			expectedField := expected.Field(i)
			r.equals(actualField, expectedField, level+1)
			r.pop()
			if len(r.diff) >= r.MaxDiff {
				break
			}
		}
	case reflect.Map:
		if actual.IsNil() || expected.IsNil() {
			if actual.IsNil() && !expected.IsNil() {
				r.keep("<nil map>", expected)
			} else if !actual.IsNil() && expected.IsNil() {
				r.keep(actual, "<nil map>")
			}
			return
		}
		if actual.Pointer() == expected.Pointer() {
			return
		}

		for _, key := range actual.MapKeys() {
			r.push(fmt.Sprintf("map[%s]", key))
			actualMapIndex := actual.MapIndex(key)
			expectedMapIndex := expected.MapIndex(key)
			if expectedMapIndex.IsValid() {
				r.equals(actualMapIndex, expectedMapIndex, level+1)
			} else {
				r.keep(actualMapIndex, "<does not have key>")
			}
			r.pop()
			if len(r.diff) >= r.MaxDiff {
				return
			}
		}
		for _, key := range expected.MapKeys() {
			if actualMapIndex := actual.MapIndex(key); actualMapIndex.IsValid() {
				continue
			}
			r.push(fmt.Sprintf("map[%s]", key))
			r.keep("<does not have key>", expected.MapIndex(key))
			r.pop()
			if len(r.diff) >= r.MaxDiff {
				return
			}
		}
	case reflect.Array:
		size := actual.Len()
		for i := 0; i < size; i++ {
			r.push(fmt.Sprintf("array[%d]", i))
			r.equals(actual.Index(i), expected.Index(i), level+1)
			r.pop()
			if len(r.diff) >= r.MaxDiff {
				break
			}
		}
	case reflect.Slice:
		if r.IncludeNilSlices {
			if actual.IsNil() && expected.Len() != 0 {
				r.keep("<nil slice>", expected)
				return
			} else if actual.Len() != 0 && expected.IsNil() {
				r.keep(actual, "<nil slice>")
				return
			}
		} else {
			if actual.IsNil() && !expected.IsNil() {
				r.keep("<nil slice>", expected)
				return
			} else if !actual.IsNil() && expected.IsNil() {
				r.keep(actual, "<nil slice>")
				return
			}
		}
		actualSize := actual.Len()
		expectedSize := expected.Len()
		if actual.Pointer() == expected.Pointer() && actualSize == expectedSize {
			return
		}
		correctedSize := actualSize
		if expectedSize > actualSize {
			correctedSize = expectedSize
		}
		for i := 0; i < correctedSize; i++ {
			r.push(fmt.Sprintf("slice[%d]", i))
			if i < actualSize && i < expectedSize {
				r.equals(actual.Index(i), expected.Index(i), level+1)
			} else if i < actualSize {
				r.keep(actual.Index(i), "<no value>")
			} else {
				r.keep("<no value>", expected.Index(i))
			}
			r.pop()
			if len(r.diff) >= r.MaxDiff {
				break
			}
		}
	case reflect.Float32, reflect.Float64:
		actualFloat := fmt.Sprintf(r.floatFormat, actual.Float())
		expectedFloat := fmt.Sprintf(r.floatFormat, expected.Float())
		if actualFloat != expectedFloat {
			r.keep(actual.Float(), expected.Float())
		}
	case reflect.Bool:
		if actual.Bool() != expected.Bool() {
			r.keep(actual.Bool(), expected.Bool())
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if actual.Int() != expected.Int() {
			r.keep(actual.Int(), expected.Int())
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if actual.Uint() != expected.Uint() {
			r.keep(actual.Uint(), expected.Uint())
		}
	case reflect.String:
		if actual.String() != expected.String() {
			r.keep(actual.String(), expected.String())
		}
	default:
		logError(r, fmt.Errorf("cannot compare the reflect.Kind : unknown condition %v", actualKind))
	}
}

func (r *cmpResult) push(name string) {
	r.buff = append(r.buff, name)
}

func (r *cmpResult) pop() {
	if len(r.buff) > 0 {
		r.buff = r.buff[0 : len(r.buff)-1]
	}
}

func (r *cmpResult) keep(actual, expected interface{}) {
	if len(r.buff) > 0 {
		varName := strings.Join(r.buff, ".")
		r.diff = append(r.diff, fmt.Sprintf("%s: %v != %v", varName, actual, expected))
	} else {
		r.diff = append(r.diff, fmt.Sprintf("%v != %v", actual, expected))
	}
}

func logError(r *cmpResult, err error) {
	if r.LogErrors {
		log.Println(err)
	}
}
