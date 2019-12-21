package model_a

import (
	"github.com/badu/stroo/testdata/pkg/model_b"
	"time"
)

type ItemPrice struct {
	Value   float32
	TaxType int32
}

type Taxes []ItemPrice // classic slice

type CartItem struct {
	ItemName      string
	Quantity      float64
	Price         ItemPrice
	Tax           *ItemPrice
	ProductImages *ProductImages // pointer to slice of pointer to alias
}

type CartItems []*CartItem // slice of pointers

type ImageURL string // string alias

type ProductImages []*ImageURL // slice of pointer to alias

type Times []*time.Time

type UserData struct {
	Name  string
	Phone *string
}

//go:generate stroo -type=ShopCart -output=easy_json_gen.go -template=./../../templates/json_marshal.tmpl
type ShopCart struct {
	// embedded field
	UserData
	// unexported field
	unexported string
	// basic fields
	Bool       bool
	BoolPtr    *bool
	Int        int
	IntPtr     *int
	Int8       int8
	Int8Ptr    *int8
	Int16      int16
	Int16Ptr   *int16
	Int32      int32
	Int32Ptr   *int32
	Int64      int64
	Int64Ptr   *int64
	Uint       uint
	UintPtr    *uint
	Uint8      uint8
	Uint8Ptr   *uint8
	Uint16     uint16
	Uint16Ptr  *uint16
	Uint32     uint32
	Uint32Ptr  *uint32
	Uint64     uint64
	Uint64Ptr  *uint64
	Float32    float32
	Float32Ptr *float32
	Float64    float64
	Float64Ptr *float64
	String     string
	StringPtr  *string
	// field from another package
	StructFromAnotherPackage model_b.StructFromAnotherPackage
	// time field
	PurchaseDate time.Time
	// pointer to time
	DeliveryDate *time.Time
	// normal slice
	Items CartItems
	// pointer to slice
	Taxes *Taxes
	// slice of fields from another package
	Times Times
}
