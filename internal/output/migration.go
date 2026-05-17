package output

import "bytes"

// spliceMigration applies the §1.8 object-merge / non-object-wrap policy to
// an already-marshaled response body, returning compact JSON. The pending
// block itself lives in internal/migrationsink (a leaf package, so keychain
// can record into it without an output<->keychain import cycle).
//
// Policy (template inherited from slck B1):
//   - object responses  -> "_migration" merged as the first top-level field,
//     original fields preserved verbatim and in order.
//   - non-object responses (arrays from list endpoints) -> wrapped as
//     {"_migration": ..., "data": <original>}.
func spliceMigration(body, mig []byte) []byte {
	t := bytes.TrimSpace(body)
	if len(t) > 0 && t[0] == '{' && t[len(t)-1] == '}' {
		inner := bytes.TrimSpace(t[1 : len(t)-1])
		if len(inner) == 0 {
			return []byte(`{"_migration":` + string(mig) + `}`)
		}
		return []byte(`{"_migration":` + string(mig) + `,` + string(inner) + `}`)
	}
	return []byte(`{"_migration":` + string(mig) + `,"data":` + string(t) + `}`)
}
