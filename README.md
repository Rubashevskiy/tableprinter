# Go Struct Table Printer

A lightweight, high-performance, type-safe Go library that automatically formats slices of structures into clean, customizable text tables using struct tags and generics.

[![Go Reference](https://go.dev)](https://go.dev)
[![License: MIT](https://shields.io)](https://opensource.org)

## Features

- **Type-Safe Generics**: Leverages Go generics (`[T any]`) to guarantee compile-time type safety for your data rows.
- **Fluent API**: Chaining-friendly `AddRow` and `AddRows` methods for elegant data loading.
- **Declarative Configuration**: Control column alignment, time formats, custom null text, and boolean labels directly via `table` struct tags.
- **Smart Precision & `StringFixed` Support**: Seamlessly formats floating-point numbers. Automatically detects and utilizes `StringFixed(prec int32) string` methods on custom types (e.g., decimal libraries).
- **Text Truncation**: Easily limit maximum cell character length using the `limit` parameter.
- **Figure Space Support**: Optionally uses Figure Space (`\u2007`) instead of standard spaces to preserve alignment in specific typographic or terminal environments.
- **Zero Dependencies**: Pure Go standard library implementation.

## Installation

```bash
go get github.com/Rubashevskiy/tableprinter
```

## Quick Start

Define your configuration using struct tags, load data using fluent methods, and render your table.

```go
package main

import (
	"fmt"
	"time"

	"://github.com"
)

type User struct {
	ID        int       `table:"l"`                       // Left align
	Name      string    `table:"l; limit:10"`             // Left align, max 10 chars
	Balance   float64   `table:"r; prec:2"`               // Right align, 2 decimal places
	IsActive  bool      `table:"l; bool:Active/Inactive"` // Custom boolean labels
	CreatedAt time.Time `table:"l; time:2006-01-02"`      // Custom time format
	Comment   *string   `table:"l; nil:N/A"`              // Custom pointer nil representation
}

func main() {
	// Initialize a new table with a title and 4-space column spacing
	table, err := tableprinter.NewTable[User]("Active Users Report", 4)
	if err != nil {
		panic(err)
	}

	// 1. Add a single row using the fluent API
	table.AddRow(User{
		ID:        1,
		Name:      "Alexander Prince", // Will be truncated to "Alexander " due to limit:10
		Balance:   1250.501,
		IsActive:  true,
		CreatedAt: time.Now(),
		Comment:   nil,
	})

	// 2. Add multiple rows at once
	moreUsers := []User{
		{ID: 2, Name: "Alice", Balance: 42.00, IsActive: false, CreatedAt: time.Now()},
		{ID: 3, Name: "Bob", Balance: 0.00, IsActive: true, CreatedAt: time.Now()},
	}
	table.AddRows(moreUsers)

	// 3. Render to string
	// Set to true to use Figure Space (\u2007) for fixed-width alignment, or false for standard space
	output := table.Render(false)
	
	fmt.Print(output)
}
```

### Output Example
```text
------------------------------------------------------------
Active Users Report
------------------------------------------------------------
1       Alexander      1250.50    Active      2026-08-13    N/A
2       Alice            42.00    Inactive    2026-08-13    nil
3       Bob               0.00    Active      2026-08-13    nil
```

## Struct Tag Syntax

The `table` tag configuration syntax looks as follows:
```go
`table:"<alignment>; param1:value1; param2:value2"`
```

### Supported Parameters

| Parameter | Required | Allowed Values | Default Value | Description |
| :-----:| :-:| :--- | :--- | :--- |
| **Alignment** | **Yes** | `l` (left), `r` (right) | — | Sets column text alignment. Must be the first parameter. |
| `time` | No | Any Go time layout | `2006-01-02 15:04:05` | Custom layout for `time.Time` fields. |
| `nil`  | No | Any string | `nil` | Custom text to display when a pointer/nilable field is `nil`. |
| `bool` | No | `True/False` or `True` | `True/False` | Custom labels for boolean values. Split by `/`. |
| `limit`| No | Integer $\ge 0$ | `0` (no limit) | Truncates cell text if it exceeds this length (measured in runes). |
| `prec` | No | Integer $\ge 0$ | `-1` (auto) | Floating-point precision (decimal places). Passed into `StringFixed(prec)` if available. |

## Advanced Features

### Custom Decimal / Type Support
If a field type implements `StringFixed(prec int32) string` (like many arbitrary-precision decimal libraries), the library will automatically invoke it instead of standard string formatting. The `prec` parameter value from your struct tag will be safely passed into this method.

### Safe Pointer & Interface Handling
The renderer automatically unwraps pointers and handles nilable Go types (`chan`, `func`, `interface`, `map`, `pointer`, `slice`). If the value is `nil`, it gracefully substitutes it with your defined `nil` parameter string.

## Validation Errors

The `NewTable` constructor validates your structures at runtime and returns an error if:
1. The passed generic type `T` is not a `struct` (or a pointer to a struct).
2. The struct has zero exported fields tagged with `table`.
3. The alignment tag is neither `l` nor `r`.
4. Numeric parameters (`limit`, `prec`) fail to parse or are negative.
5. Parameter syntax is malformed (missing key/value separator).

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
