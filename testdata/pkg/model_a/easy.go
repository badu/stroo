package model_a

import (
	"github.com/badu/stroo/testdata/pkg/model_b"
	"time"
)

type ItemPrice struct {
	Value   float32 `json:"dollars"`
	TaxType int32   `json:"type"`
}

type CartItem struct {
	ItemName      string         `json:"name"`
	Quantity      float64        `json:"howMuch"`
	Price         ItemPrice      `json:"price"`
	Tax           *ItemPrice     `json:"tax"`
	ProductImages *ProductImages `json:"images"` // pointer to slice of pointer to alias
}

type Taxes []ItemPrice         // classic slice
type CartItems []*CartItem     // slice of pointers
type ImageURL string           // string alias
type ImagesPtr *string         // pointer to string alias
type Timer time.Ticker         // external alias
type Timer2 *time.Ticker       // external alias
type ProductImages []*ImageURL // slice of pointer to alias
// times comment @time.Time
type Times []*time.Time // slice of pointer to external

type UserData struct {
	Name          string    `json:"buyer"`
	Phone         *string   `json:"mobile"`
	Timer         Timer     `json:"timer"`
	ImportedField time.Time `json:"importedField"`
}

type UserDataPtr struct {
	Name          string     `json:"customer"`
	Phone         *string    `json:"mobile"`
	AnotherTimer  Timer2     `json:"timer"`
	ImportedField *time.Time `json:"importedField"`
}

type Pot struct {
	Name string `json:"potName"`
}

//go:generate stroo -type=ShopCart -output=easy_json_gen.go -template=./../../templates/json_marshal.tmpl
type ShopCart struct {
	string                            // embedded field
	UserData     `json:"usr"`         // embedded struct
	*UserDataPtr `json:"anotherData"` // embedded pointer
	Times                             // embedded array
	unexported   string               // unexported field
	// basic fields
	Bool                     bool                             `json:"boolean_value"`
	BoolPtr                  *bool                            `json:"a_BoolPtr"`
	Int                      int                              `json:"int_value"`
	IntPtr                   *int                             `json:"a_IntPtr"`
	Int8                     int8                             `json:"int8_value"`
	Int8Ptr                  *int8                            `json:"a_Int8Ptr"`
	Int16                    int16                            `json:"int16_value"`
	Int16Ptr                 *int16                           `json:"a_Int16Ptr"`
	Int32                    int32                            `json:"int32_value"`
	Int32Ptr                 *int32                           `json:"a_Int32Ptr"`
	Int64                    int64                            `json:"int64_value"`
	Int64Ptr                 *int64                           `json:"a_Int64Ptr"`
	Uint                     uint                             `json:"uint_value"`
	UintPtr                  *uint                            `json:"a_UintPtr"`
	Uint8                    uint8                            `json:"uint8_value"`
	Uint8Ptr                 *uint8                           `json:"a_Uint8Ptr"`
	Uint16                   uint16                           `json:"uint16_value"`
	Uint16Ptr                *uint16                          `json:"a_Uint16Ptr"`
	Uint32                   uint32                           `json:"uint32_value"`
	Uint32Ptr                *uint32                          `json:"a_Uint32Ptr"`
	Uint64                   uint64                           `json:"uint64_value"`
	Uint64Ptr                *uint64                          `json:"a_Uint64Ptr"`
	Float32                  float32                          `json:"float32_value"`
	Float32Ptr               *float32                         `json:"a_Float32Ptr"`
	Float64                  float64                          `json:"float64_value"`
	Float64Ptr               *float64                         `json:"a_Float64Ptr"`
	StringField              string                           `json:"string_value"`
	StringPtr                *string                          `json:"a_StringPtr"`
	Pot                      Pot                              `json:"-"`                        // normal struct
	StructFromAnotherPackage model_b.StructFromAnotherPackage `json:"structFromAnotherPackage"` // field from another package
	PurchaseDate             time.Time                        `json:"purchaseDate"`             // time field
	DeliveryDate             *time.Time                       `json:"emptyTime"`                // pointer to time
	Items                    CartItems                        `json:"items"`                    // normal slice
	Taxes                    *Taxes                           `json:"appliedTaxes"`             // pointer to slice
	ArrayOfPointerToImport   Times                            `json:"arrayOfPtrToTimeDotTime"`  // slice of fields from another package
}
