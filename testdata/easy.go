package testdata

import (
	"github.com/badu/stroo/testdata/other"
	"time"
)

type T0 []int
type T1 []*int
type T2 [][]int
type T2_1 []*[]int
type T3 []map[string]string
type S struct{ Name string }

type T4 []map[S]string
type S2 struct{ Name string }

type T5 []*map[S2]string
type T6 []struct{ Name string }
type T7 []chan string
type T8 []*chan string
type T9 []chan struct{}
type T10 []*chan *struct{}
type T11 []*struct{ Name string }
type S3 struct{ Name string }

type T12 []S3
type T13 []S13

type S13 struct {
	Name  string
	Email string
}
type S4 struct{ Name string }

type T14 []*S4
type T15 struct {
	EmbeddedS          // embedded
	*EmbeddedS2        // embedded pointer
	error              // embedded error
	Name        string `json:"name"`
	PtrName     *string
	unexported  string
}
type EmbeddedS struct {
	Name  string `json:"s.name"`
	Email string `json:"mail"`
}
type EmbeddedS2 struct {
	Name string `json:"s2.name"`
}
type T16 struct {
	S      S5
	Itemz  Items  `json:"itmz"`
	Pricez Prices `json:"prcz"`
}
type S5 struct{}
type Prices []Price2
type Items []*Item
type Item struct {
	Name  string `json:"name"`
	Stock float64
}
type Price2 struct {
	Name  string `json:"name"`
	Value float64
}
type T17 struct {
	Items
}
type T18 struct {
	Child *T18 `json:"ptr_child"`
}

//go:generate stroo -type=TestData -output=easy_json_gen.go -template=./../../templates/json_marshal.tmpl
type TestData struct {
	// basic fields
	Bool                         bool                           `json:"boolean_value"`
	BoolPtr                      *bool                          `json:"a_BoolPtr"`
	Int                          int                            `json:"int_value"`
	IntPtr                       *int                           `json:"a_IntPtr"`
	Int8                         int8                           `json:"int8_value"`
	Int8Ptr                      *int8                          `json:"a_Int8Ptr"`
	Int16                        int16                          `json:"int16_value"`
	Int16Ptr                     *int16                         `json:"a_Int16Ptr"`
	Int32                        int32                          `json:"int32_value"`
	Int32Ptr                     *int32                         `json:"a_Int32Ptr"`
	Int64                        int64                          `json:"int64_value"`
	Int64Ptr                     *int64                         `json:"a_Int64Ptr"`
	Uint                         uint                           `json:"uint_value"`
	UintPtr                      *uint                          `json:"a_UintPtr"`
	Uint8                        uint8                          `json:"uint8_value"`
	Uint8Ptr                     *uint8                         `json:"a_Uint8Ptr"`
	Uint16                       uint16                         `json:"uint16_value"`
	Uint16Ptr                    *uint16                        `json:"a_Uint16Ptr"`
	Uint32                       uint32                         `json:"uint32_value"`
	Uint32Ptr                    *uint32                        `json:"a_Uint32Ptr"`
	Uint64                       uint64                         `json:"uint64_value"`
	Uint64Ptr                    *uint64                        `json:"a_Uint64Ptr"`
	Float32                      float32                        `json:"float32_value"`
	Float32Ptr                   *float32                       `json:"a_Float32Ptr"`
	Float64                      float64                        `json:"float64_value"`
	Float64Ptr                   *float64                       `json:"a_Float64Ptr"`
	StringField                  string                         `json:"string_value"`
	StringPtr                    *string                        `json:"a_StringPtr"`
	StructField                  NormalStruct                   `json:"-"`                       // normal struct
	TimeField                    time.Time                      `json:"purchaseDate"`            // time field
	PtrTimeField                 *time.Time                     `json:"emptyTime"`               // pointer to time
	SliceOfPointersField         SliceOfPointers                `json:"items"`                   // normal slice
	PtrToClassicSliceField       *ClassicSlice                  `json:"appliedTaxes"`            // pointer to slice
	ExternalSliceOfPointersField ExternalSliceOfPointers        `json:"arrayOfPtrToTimeDotTime"` // slice of fields from another package
	EmbeddedField                                               // embedded struct
	*EmbeddedPtrField                                           // pointer to embedded struct
	StructFromAnotherPackage     other.StructFromAnotherPackage `json:"structFromAnotherPackage"` // field from another package
}

type Price struct {
	Value   float32 `json:"dollars"`
	TaxType int32   `json:"type"`
}

type CartItem struct {
	ItemName string                  `json:"name"`
	Quantity float64                 `json:"howMuch"`
	PriceTag Price                   `json:"price"`
	Tax      *Price                  `json:"tax"`
	Images   *SliceOfPointersToAlias `json:"images"` // pointer to slice of pointer to alias
}

type ClassicSlice []Price                 // classic slice
type SliceOfPointers []*CartItem          // slice of pointers
type BasicAlias string                    // string alias
type BasicPtrAlias *string                // pointer to string alias
type ExternalAlias time.Ticker            // external alias
type ExternalPtrAlias *time.Ticker        // external alias
type SliceOfPointersToAlias []*BasicAlias // slice of pointer to alias
// times comment @time.Time
type ExternalSliceOfPointers []*time.Time // slice of pointer to external

type EmbeddedField struct {
	Name          string        `json:"buyer"`
	Phone         *string       `json:"mobile"`
	AliasField    ExternalAlias `json:"timer"`
	ImportedField time.Time     `json:"importedField"`
}

type EmbeddedPtrField struct {
	Name          string           `json:"customer"`
	Phone         *string          `json:"mobile"`
	AliasField    ExternalPtrAlias `json:"timer"`
	ImportedField *time.Time       `json:"importedField"`
}

type NormalStruct struct {
	Name string `json:"potName"`
}

type Service interface {
	Store(*TestData) (*TestData, error)
}
