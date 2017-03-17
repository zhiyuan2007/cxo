package skyobject

import (
	"reflect"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

var RLEN = len(Reference{}) // 32

//
// set of keys
//

// keys set
type Set map[Reference]struct{}

func (s Set) Add(k Reference) {
	s[k] = struct{}{}
}

//
// find missing objects
//

// missing objects
func (c *Container) Want() (set Set, err error) {
	if c.root == nil {
		return // don't want anything (has no root object)
	}
	set = make(Set)
	err = c.wantKeys(c.root.Schema, c.root.Object, set)
	return
}

// want by schema key and object key
func (c *Container) wantKeys(sk, ok Reference, set Set) (err error) {
	var sd []byte // shcema data and object data
	var ex bool   // exist
	if sd, ex = c.get(sk); !ex {
		set.Add(sk)
		if _, ex = c.get(ok); ex {
			set.Add(ok)
		}
		return
	}
	var s Schema
	if err = encoder.DeserializeRaw(sd, &s); err != nil {
		return
	}
	err = c.wantSchemaObjKey(&s, ok, set)
	return
}

// by schema and object key
func (c *Container) wantSchemaObjKey(s *Schema,
	ok Reference, set Set) (err error) {

	var od []byte // object data
	var ex bool   // exist
	if _, ex = c.get(ok); !ex {
		set.Add(ok)
		return
	}

	_, err = c.wantSchemaObjData(s, od, set)
	return
}

// by schema and object data
func (c *Container) wantSchemaObjData(s *Schema,
	od []byte, set Set) (n int, err error) {

	switch s.Kind() {
	case reflect.Bool, reflect.Int8, reflect.Uint8:
		n += 1
	case reflect.Int16, reflect.Uint16:
		n += 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		n += 4
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		n += 8
	case reflect.String:
		var l int
		if l, err = getLength(od); err != nil {
			return
		}
		n += 4 + l
	case reflect.Array:
		// it is not a field and we can't see tags and treat it as cipher.SHA256
		var elem *Schema
		if elem, err = s.Elem(); err != nil {
			return
		}
		var l int = s.Len()
		if kind := elem.Kind(); isBasic(kind) {
			n = l * basicSize(kind)
			return
		} else {
			var m int
			for i := 0; i < l; i++ {
				if m, err = c.wantSchemaObjData(elem, od[n:], set); err != nil {
					return
				}
				n += m
			}
		}
	case reflect.Slice:
		var elem *Schema
		if elem, err = s.Elem(); err != nil {
			return
		}
		var l int
		if l, err = getLength(od); err != nil {
			return
		}
		n += 4 // length
		if kind := elem.Kind(); isBasic(kind) {
			n += l * basicSize(kind)
			return
		} else {
			var m int
			for i := 0; i < l; i++ {
				if m, err = c.wantSchemaObjData(elem, od[n:], set); err != nil {
					return
				}
				n += m
			}
		}
	case reflect.Struct:
		if s.Name() == dynamicRef { // not a field -> not a reference
			n = RLEN * 2
		} else {
			var m int
			for _, sf := range s.Fields() {
				if m, err = c.wantField(&sf, od[n:], set); err != nil {
					return
				}
				n += m
			}
		}
	default:
		err = ErrInvalidSchema
	}

	return
}

func (c *Container) wantField(f *Field, od []byte, set Set) (n int, err error) {

	var s *Schema

	if s, err = f.Schema(); err != nil {
		return
	}

	switch s.Kind() {
	case reflect.Bool, reflect.Int8, reflect.Uint8:
		n += 1
	case reflect.Int16, reflect.Uint16:
		n += 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		n += 4
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		n += 8
	case reflect.String:
		var l int
		if l, err = getLength(od); err != nil {
			return
		}
		n += 4 + l
	case reflect.Array:
		if s.Name() == singleRef { // Reference (cipher.SHA256)
			var ref Reference
			if err = encoder.DeserializeRaw(od[:RLEN], &ref); err != nil {
				return
			}
			n += 32
			var sr *Schema
			if sr, err = f.SchemaOfReference(); err != nil {
				return
			}
			if err = c.wantSchemaObjKey(sr, ref, set); err != nil {
				return
			}
		} else { // not a reference
			var elem *Schema
			if elem, err = s.Elem(); err != nil {
				return
			}
			var l int = s.Len()
			if kind := elem.Kind(); isBasic(kind) {
				n = l * basicSize(kind)
				return
			} else {
				var m int
				for i := 0; i < l; i++ {
					m, err = c.wantSchemaObjData(elem, od[n:], set)
					if err != nil {
						return
					}
					n += m
				}
			}
		}
	case reflect.Slice:
		var l int
		if l, err = getLength(od); err != nil {
			return
		}
		n += 4
		if s.Name() == arrayRef { // Reference (cipher.SHA256)
			var refs References
			if err = encoder.DeserializeRaw(od, &refs); err != nil {
				return
			}
			var sr *Schema
			if sr, err = f.SchemaOfReference(); err != nil {
				return
			}
			for _, ok := range refs {
				if err = c.wantSchemaObjKey(sr, ok, set); err != nil {
					return
				}
			}
		} else { // not a reference
			var elem *Schema
			if elem, err = s.Elem(); err != nil {
				return
			}
			if isFlat(elem.Kind()) { // can't contain references
				n = 4 + l
				return
			} else {
				var m, k int = 4, 0
				for m < 4+l {
					k, err = c.wantSchemaObjData(elem, od[m:], set)
					if err != nil {
						return
					}
					m += k
				}
			}
		}
		n = 4 + l
	case reflect.Struct:
		if s.Name() == dynamicRef { // dynamic refernce
			n = RLEN * 2 // len(cipher.SHA256{}) * 2
			var dh Dynamic
			if err = encoder.DeserializeRaw(od, &dh); err != nil {
				return
			}
			err = c.wantKeys(dh.Schema, dh.Object, set)
		} else {
			var m int
			for _, sf := range s.Fields() {
				if m, err = c.wantField(&sf, od[n:], set); err != nil {
					return
				}
				n += m
			}
		}
	default:
		err = ErrInvalidSchema
	}

	return

}

func getLength(p []byte) (l int, err error) {
	var u uint32
	err = encoder.DeserializeRaw(p, &u)
	l = int(u)
	return
}

func basicSize(kind reflect.Kind) (n int) {
	switch kind {
	case reflect.Bool, reflect.Int8, reflect.Uint8:
		n = 1
	case reflect.Int16, reflect.Uint16:
		n = 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		n = 4
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		n = 8
	}
	return
}
