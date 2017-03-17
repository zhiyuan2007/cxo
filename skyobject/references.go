package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
)

//
// use special named types for references
//

// A Reference type represents reference to another object
type Reference cipher.SHA256

// A References type represents references to array of another objects
type References []Reference

// A Dynamic represents dynamic reference to any object and reference to its
// schema
type Dynamic struct {
	Schema Reference
	Object Reference
}
