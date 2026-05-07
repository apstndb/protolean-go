# protolean-go

**ProtoJSON for [LEAN](https://github.com/fiialkod/lean-format)** — a Go package that converts protocol buffer messages to the LEAN format.

> **proto + lean = protolean**

LEAN (LLM-Efficient Adaptive Notation) is a token-optimized serialization format that uses tab delimiters, single-char keywords (`T`/`F`/`_`), and tabular arrays to achieve 28% fewer tokens than JSON.

This package is modeled after [ProtoJSON](https://protobuf.dev/programming-guides/json/), the canonical JSON mapping for Protocol Buffers. Just as ProtoJSON defines how `proto.Message` values are serialized as JSON, `protolean` defines how they are serialized as LEAN.

## Features

- **Direct conversion**: Uses `google.golang.org/protobuf/reflect/protoreflect` to inspect proto messages and build LEAN data structures directly.
- **Tabular optimization**: Repeated proto messages with uniform schemas are automatically encoded in LEAN's compact tabular form (`key[N]:field1 field2` with tab-delimited rows).
- **Semi-tabular optimization**: Mixed-schema arrays use LEAN's `~` marker to factor shared keys into columns while keeping extra keys inline.
- **Well-known types**: Supports all 17 `google.protobuf` well-known types (Timestamp, Duration, Struct, Value, ListValue, FieldMask, Any, wrapper types, Empty).
- **Per-type default emission**: Make specific repeated row types tabular without adding noise to the entire protobuf tree.

## Installation

```bash
go get github.com/apstndb/protolean-go
```

## Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/apstndb/protolean-go/protolean"
)

func main() {
    p := &Person{Name: "Alice", Age: 30, Active: true}

    lean, err := protolean.Marshal(p)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(lean)
    // Output:
    // name:Alice
    // age:30
    // active:T
}
```

### Tabular arrays

Repeated messages with uniform schemas become tabular:

```go
company := &Company{
    Employees: []*Person{
        {Name: "Alice", Age: 30, Active: true},
        {Name: "Bob", Age: 25, Active: false},
    },
}

lean, _ := protolean.Marshal(company)
// employees[2]:name	age	active
//   Alice	30	T
//   Bob	25	F
```

### Options

```go
// Emit default values for specific types (enables tabular form)
protolean.MarshalOptions{
    EmitDefaultValuesForTypes: []protoreflect.FullName{"myapp.Person"},
}.Marshal(msg)
```

## How it works

1. `protolean` walks the `proto.Message` using `protoreflect.Message.Range`.
2. Each field value is mapped to plain Go values (`string`, `int64`, `float64`, `bool`, `[]any`, `map[string]any`).
3. The resulting structure is passed to `lean.Encode`, which applies LEAN-specific optimizations such as tabular arrays, dot-flattening, and semi-tabular encoding.

## License

MIT
