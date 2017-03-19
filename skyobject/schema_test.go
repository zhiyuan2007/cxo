package skyobject

import (
	"reflect"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

func shouldPanic(t *testing.T) {
	if recover() == nil {
		t.Error("missing panicing")
	}
}

func TestNewRegistery(t *testing.T) {
	t.Run("nil db", func(t *testing.T) {
		defer shouldPanic(t)
		NewRegistery(nil)
	})
	t.Run("norm", func(t *testing.T) {
		db := data.NewDB()
		r := NewRegistery(db)
		if r.db != db {
			t.Error("wrong db in registery")
		}
		if r.nmr == nil {
			t.Error("nil-map for registered types")
		}
		if r.reg == nil {
			t.Error("nil-map registery")
		}
	})
}

func TestRegistry_Register(t *testing.T) {
	t.Run("unnamed", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		defer shouldPanic(t)
		r.Register("Unnamed", []User{})
	})
	t.Run("named", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		r.Register("User", User{})
		tn := typeName(reflect.TypeOf(User{}))
		if x, ok := r.nmr["User"]; !ok {
			t.Error("missing registered type")
		} else if x != tn {
			t.Error("registered with wrong type name")
		} else if ch, ok := r.reg[x]; !ok {
			t.Error("name registered, but type is not registered")
		} else if s, err := r.SchemaByReference(Reference(ch)); err != nil {
			t.Error("unexpected error: ", err)
		} else if s.Name() != tn {
			// TODO
			t.Error("registered type has wrong name: ", s.Name())
		}
	})
}

func TestRegistry_SaveSchema(t *testing.T) {
	t.Run("invalid type", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		var x interface{}
		defer shouldPanic(t)
		r.SaveSchema(x)
	})
	t.Run("valid", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		ur := r.SaveSchema(User{})
		if ur == (Reference{}) {
			t.Error("empty reference to saved type")
		}
		if _, ok := r.db.Get(cipher.SHA256(ur)); !ok {
			t.Error("saved schema missing in db")
		}
		typ := reflect.TypeOf(User{})
		if ch, ok := r.reg[typeName(typ)]; !ok {
			t.Error("saved schema missing in registery")
		} else if Reference(ch) != ur {
			t.Error("wrong reference for saved schema")
		}
	})
	t.Run("recursive", func(t *testing.T) {
		type Recur struct {
			Name   string
			Len    uint32
			Nested []Recur
		}
		r := NewRegistery(data.NewDB())
		ur := r.SaveSchema(Recur{})
		if ur == (Reference{}) {
			t.Error("empty reference to saved type")
		}
		if _, ok := r.db.Get(cipher.SHA256(ur)); !ok {
			t.Error("saved schema missing in db")
		}
		typ := reflect.TypeOf(Recur{})
		if ch, ok := r.reg[typeName(typ)]; !ok {
			t.Error("saved schema missing in registery")
		} else if Reference(ch) != ur {
			t.Error("wrong reference for saved schema")
		}
	})
}

func TestRegistry_SchemaByTypeName(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		_, err := r.SchemaByTypeName("missing")
		if err == nil {
			t.Error("missing error")
		}
	})
	t.Run("saved", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		r.SaveSchema(User{})
		tn := typeName(reflect.TypeOf(User{}))
		s, err := r.SchemaByTypeName(tn)
		if err != nil {
			t.Error(err)
			return
		}
		// TODO
		t.Log("Schema: ", s)
	})
}

func TestRegistry_SchemaByReference(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		_, err := r.SchemaByReference(Reference{1, 2, 3})
		if err == nil {
			t.Error("missing error")
		}
	})
	t.Run("saved", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		ur := r.SaveSchema(User{})
		s, err := r.SchemaByReference(ur)
		if err != nil {
			t.Error(err)
			return
		}
		// TODO
		t.Log("Schema: ", s)
	})
}

func TestRegistry_getSchema(t *testing.T) {
	t.Run("flat", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		for _, i := range []interface{}{
			false,
			int8(0), int16(0), int32(0), int64(0),
			uint8(0), uint16(0), uint32(0), uint64(0),
			float32(0), float64(0),
			"empty",
		} {
			typ := reflect.TypeOf(i)
			sch := r.getSchema(typ)
			if sch == nil {
				t.Error("got nil-schema")
				continue
			}
			if sch.Kind() != typ.Kind() {
				t.Errorf("wrong kind: want %s, got %s",
					typ.Kind().String(),
					sch.Kind().String())
			}
			if sch.Name() != "" {
				t.Error("non-empty name for unnamed type: ", sch.Name())
			}
		}
	})
	t.Run("invalid", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		for _, i := range []interface{}{
			nil,
			make(chan struct{}),
			func() {},
			int(0),
			uint(0),
		} {
			typ := reflect.TypeOf(i)
			func() {
				defer shouldPanic(t)
				r.getSchema(typ)
			}()
		}
	})
	t.Run("flat named", func(t *testing.T) {
		type Bool bool
		type Int8 int8
		type Int16 int16
		type Int32 int32
		type Int64 int64
		type Uint8 uint8
		type Uint16 uint16
		type Uint32 uint32
		type Uint64 uint64
		type Float32 float32
		type Float64 float64
		type String string
		r := NewRegistery(data.NewDB())
		for _, i := range []interface{}{
			Bool(false),
			Int8(0), Int16(0), Int32(0), Int64(0),
			Uint8(0), Uint16(0), Uint32(0), Uint64(0),
			Float32(0), Float64(0),
			String("empty"),
		} {
			typ := reflect.TypeOf(i)
			sch := r.getSchema(typ)
			if sch == nil {
				t.Error("got nil-schema")
				continue
			}
			if sch.Kind() != typ.Kind() {
				t.Errorf("wrong kind: want %s, got %s",
					typ.Kind().String(),
					sch.Kind().String())
			}
			if sch.Name() != typeName(typ) {
				t.Error("non-empty name for unnamed type: ", sch.Name())
			}
		}
	})
	t.Run("complex unnamed", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		for _, i := range []interface{}{
			[]bool{},
			[1]int8{}, [2]int16{}, [3]int32{}, [4]int64{},
			[]uint8{}, [16]uint16{}, [32]uint32{}, []uint64{},
			[]float32{}, []float64{},
			struct{ Name string }{"empty"},
		} {
			typ := reflect.TypeOf(i)
			sch := r.getSchema(typ)
			if sch == nil {
				t.Error("got nil-schema")
				continue
			}
			if sch.Kind() != typ.Kind() {
				t.Errorf("wrong kind: want %s, got %s",
					typ.Kind().String(),
					sch.Kind().String())
			}
			if sch.Name() != typeName(typ) {
				t.Error("non-empty name for unnamed type: ", sch.Name())
			}
			switch typ.Kind() {
			case reflect.Array:
				if sch.Len() != typ.Len() {
					t.Errorf("wrong length of array: want %d, got %d",
						typ.Len(), sch.Len())
				}
				fallthrough
			case reflect.Slice:
				el, err := sch.Elem()
				if err != nil {
					t.Error(err)
					continue
				}
				if el.Kind() != typ.Elem().Kind() {
					t.Errorf("invalid kind of element: want %s, got %s",
						typ.Elem().Kind().String(),
						el.Kind().String())
				}
			case reflect.Struct:
				if len(sch.Fields()) != typ.NumField() {
					t.Errorf("wrong fields count: want %d, got %d",
						typ.NumField(), len(sch.Fields()))
				}
			FieldsRange:
				for i := 0; i < typ.NumField(); i++ {
					fl := typ.Field(i)
					for _, sf := range sch.Fields() {
						if sf.Name() == fl.Name {
							if sf.Kind() != fl.Type.Kind() {
								t.Error("wrong kind of field: ", sf.Kind())
							}
							continue FieldsRange
						}
						t.Error("can't find field: ", fl.Name)
					}
				}
			}

		}
	})
	t.Run("references", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		for _, i := range []interface{}{
			Reference{},
			References{},
			Dynamic{},
		} {
			typ := reflect.TypeOf(i)
			sch := r.getSchema(typ)
			// must not be saved
			if _, ok := r.reg[sch.Name()]; ok {
				t.Error("saved reference scehma: ", sch.Name())
			}
		}
	})
	t.Run("complex named", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		r.Register("User", User{})
		for _, i := range []interface{}{
			Group{},
			User{},
			List{},
			Man{},
		} {
			typ := reflect.TypeOf(i)
			sch := r.getSchema(typ)
			// must be saved
			if _, ok := r.reg[sch.Name()]; !ok {
				t.Error("schema of named type was not saved: ", sch.Name())
			}
		}
	})
	// TODO: complex named recursive
	// TODO: skip fields
	// TODO: elem of references
}

func TestRegistry_getField(t *testing.T) {
	r := NewRegistery(data.NewDB())
	r.Register("User", User{})
	//
}

func TestSchema_Name(t *testing.T) {
	//
}

func TestSchema_Kind(t *testing.T) {
	//
}

func TestSchema_Elem(t *testing.T) {
	//
}

func TestSchema_Len(t *testing.T) {
	//
}

func TestSchema_Fields(t *testing.T) {
	//
}

func TestSchema_setElem(t *testing.T) {
	//
}

func TestSchema_isNamed(t *testing.T) {
	//
}

func TestSchema_isSaved(t *testing.T) {
	//
}

func TestSchema_load(t *testing.T) {
	//
}

func TestSchema_String(t *testing.T) {
	//
}

func TestField_Kind(t *testing.T) {
	//
}

func TestField_TypeName(t *testing.T) {
	//
}

func TestField_Name(t *testing.T) {
	//
}

func TestField_Schema(t *testing.T) {
	//
}

func TestField_Tag(t *testing.T) {
	//
}

func TestField_tagSchemaName(t *testing.T) {
	//
}

func TestField_isReference(t *testing.T) {
	//
}

func TestField_String(t *testing.T) {
	//
}

func TestSchema_reset(t *testing.T) {
	//
}

func TestSchema_Encode(t *testing.T) {
	//
}

func TestSchema_Decode(t *testing.T) {
	//
}

func Test_typeName(t *testing.T) {
	typ := reflect.Indirect(reflect.ValueOf(User{})).Type()
	got := typeName(typ)
	t.Log("typeName: ", got)
	if want := typ.PkgPath() + "." + typ.Name(); want != got {
		t.Errorf("wrong type name: want %q, got %q", want, got)
	}
}

func Test_isFlat(t *testing.T) {
	t.Run("flat", func(t *testing.T) {
		for _, k := range []reflect.Kind{
			reflect.Bool,
			reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64,
			reflect.String,
		} {
			if !isFlat(k) {
				t.Error("flat type treated as not flat: ", k)
			}
		}
	})
	t.Run("complex", func(t *testing.T) {
		for _, k := range []reflect.Kind{
			reflect.Slice,
			reflect.Array,
			reflect.Struct,
			reflect.Ptr,
		} {
			if isFlat(k) {
				t.Error("complex type treated as flat: ", k)
			}
		}
	})

}

func Test_mustGetSchemaOfTag(t *testing.T) {
	//
}
