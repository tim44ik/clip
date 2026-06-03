# clip Scripting Language Documentation

This document describes the syntax and features of the embedded scripting language used in clip modules.

## Basic Concepts

- **Variables**: All variables begin with `%` (e.g., `%x`, `%result`).  
  Variables are dynamically typed and can hold numbers, strings, booleans, and arrays.
- **Comments**: Start with `#` and continue to the end of the line.
- **Output**: Use the built-in `print(...)` function. Multiple arguments can be passed separated by commas: `print(%a, %b, "text")`.

## Data Types

| Type    | Examples                          | Notes                              |
|---------|-----------------------------------|------------------------------------|
| Number  | `10`, `-5`, `0`                   | Integer (int)                      |
| String  | `"hello"`, `"123"`                | Double quotes required             |
| Boolean | `true`, `false`                   | Results of comparisons and logic   |
| Array   | `[1, "two", true]`                | Mixed types allowed                |

## Variables

Assignment:
```
%name = "John"
%age = 30
%data = [1, 2, 3]
%flag = true
```

Usage in expressions:
```
%sum = %a + %b
%greeting = "Hello, " + %name
```

## Operators

### Arithmetic

| Operator | Description               | Example     |
|----------|---------------------------|-------------|
| `+`      | Addition / concatenation  | `%a + %b`   |
| `-`      | Subtraction               | `%a - %b`   |
| `*`      | Multiplication            | `%a * %b`   |
| `/`      | Division                  | `%a / %b`   |
| `%`      | Modulo (remainder)        | `%a % %b`   |

> Note: The `%` operator is also used for variable prefix; the lexer distinguishes based on context.

### Comparisons

| Operator | Description        | Example      |
|----------|--------------------|--------------|
| `==`     | Equal              | `%x == 10`   |
| `!=`     | Not equal          | `%x != 5`    |
| `<`      | Less than          | `%i < 10`    |
| `>`      | Greater than       | `%i > 0`     |
| `<=`     | Less or equal      | `%i <= 5`    |
| `>=`     | Greater or equal   | `%i >= 0`    |

Comparison results are boolean (`true` or `false`).

### Logical Operators

| Operator | Description                     | Example                  |
|----------|---------------------------------|--------------------------|
| `and`    | Logical AND (short-circuit)     | `%a > 0 and %b < 10`     |
| `or`     | Logical OR (short-circuit)      | `%x == 0 or %y == 0`     |
| `not`    | Logical NOT (unary)             | `not %flag`              |

Logical operators work on booleans. In conditional contexts, any non‑zero number and non‑empty string evaluate to `true`.

## Control Flow

### Conditional `if`

```
if condition then
    # body
else
    # alternative (optional)
end
```

Example:
```
if %x > 0 then
    print("positive")
else
    print("non-positive")
end
```

### `for` Loop

C‑style syntax:
```
for initialization; condition; post do
    # body
end
```

All three sections are optional. Examples:

```
# Standard loop
for %i = 0; %i < 10; %i = %i + 1 do
    print(%i)
end

# Infinite loop
for ; ; do
    # ...
end

# Loop with empty init and post
%i = 0
for ; %i < 5; do
    %i = %i + 1
end
```

### `break` and `continue`

- `break` – immediately exits the loop.
- `continue` – jumps to the next iteration (post‑step is executed).

Example:
```
for %i = 0; %i < 10; %i = %i + 1 do
    if %i == 5 then
        break
    end
    if %i == 2 then
        continue
    end
    print(%i)
end
```

## Built‑in Functions

### `print(...)`
Prints values to standard output (appears in the module's output area in the GUI).  
Accepts any number of comma‑separated arguments.

```
print("Hello")
print(%x, %y, "result")
```

### `len(arr_or_str)`
Returns the length of an array (number of elements) or the length of a string in **runes** (characters).

```
%arr = [1,2,3]
print(len(%arr))   # 3
%s = "hello"
print(len(%s))     # 5 (five runes)
```

### `append(array, elem1, elem2, ...)`
Adds one or more elements to a **copy** of the array (creates a new array) and returns the new array.  
The original array is unchanged.

```
%arr = [1,2]
%new = append(%arr, 3, 4)
print(%new)    # [1, 2, 3, 4]
```

### `split(string, separator)`
Splits a string into an array of substrings by the given separator. Separator is a string.

```
%words = split("one,two,three", ",")
print(%words)   # [one, two, three]
```

### `fields(string)`
Splits a string into an array of words on whitespace (like `strings.Fields` in Go).  
Extra whitespace is removed.

```
%parts = fields("  foo bar  baz")
print(%parts)   # [foo, bar, baz]
```

### `contains(str_or_arr, value)`

- **For strings**: returns `true` if the string contains the substring.
- **For arrays**: returns `true` if the array contains the exact value (using equality comparison).

```
%text = "hello world"
print(contains(%text, "world"))   # true

%arr = [1, "two", 3]
print(contains(%arr, "two"))      # true
print(contains(%arr, 5))          # false
```

### `replace(str_or_arr, old, new)`

- **For strings**: replaces all occurrences of `old` with `new` and returns the new string.
- **For arrays**: returns a new array where every element equal to `old` is replaced by `new`. The original array is unchanged.

```
%text = "cat dog cat"
%newText = replace(%text, "cat", "mouse")
print(%newText)   # "mouse dog mouse"

%arr = [1, 2, 3, 2]
%newArr = replace(%arr, 2, 99)
print(%newArr)    # [1, 99, 3, 99]
```

### `run(command)`
Executes an external command in a **long‑lived** bash process.  
Context (working directory, environment variables) is preserved between calls within the same module.  
Returns the combined stdout+stderr of the command as a string.  
If the command exits with a non‑zero code, a runtime error occurs (script stops).

```
%out = run("ls -la")
print(%out)

run("cd /tmp")
%pwd = run("pwd")
print(%pwd)   # /tmp
```

### `runIsolated(command)`
Executes an external command in a **one‑shot** process (a new process each call).  
Context is **not** preserved between calls. Useful for commands that should not affect the environment or exit with a non-zero code.  
Returns the command’s stdout (stderr is included in the error on non‑zero exit).

```
%date = runIsolated("date")
print(%date)
```

### `process(dbType, data1, data2, ...)`
Queries the vulnerability database (CVE/CPE) and returns an array of strings – results of processing each argument.  
`dbType` is a string indicating the database type (usually `"NVD"`).  
The remaining arguments are strings containing text to analyse (e.g., command output).  
The function uses a pre‑filled PostgreSQL database to find CVEs for products and versions.

```
%res = process("NVD", "nginx 1.20", "CVE-2023-1234")
print(%res)
```

### `report(fileType, ...)`
Generates a report (e.g., PDF) from the content accumulated by previous calls to `addToReport` or `report` itself.  
The first argument is the report type (e.g., `".pdf"`).  
Subsequent arguments are strings that are appended to the current report content.  
Returns the current report content for module.  
If no report exists, it creates a new one.

```
%reportContent = report(".pdf", "Title", "Line 1") 
```

## Arrays and Array Operations

### Array literal
```
%arr = [1, "hello", true, [2,3]]
```

### Indexing
```
%elem = %arr[1]        # "hello"
%arr[2] = false        # assignment
```

### Slicing
Syntax: `%arr[start:end]` (both indices optional).  
Works for arrays and strings (strings are indexed by runes).

```
%arr = [10,20,30,40]
%sub = %arr[1:3]       # [20,30]
%s = "hello world"
%substr = %s[0:5]      # "hello"
```

### Built‑in functions for arrays
- `len(%arr)` – array length.
- `append(%arr, elem1, ...)` – returns a new array.
- `%arr[%i]` – read/write element.

### Functions for type conversion

- `int` - to convert string to integer, panics if the value isn't an integer.
- `str` - to convert integer to string.

## Example module

```
%sum = 0
for %i = 1; %i <= 5; %i = %i + 1 do
    %sum = %sum + %i
end
print("Sum from 1 to 5 =", %sum)

%text = "the cat in the hat"
if contains(%text, "cat") then
    %newText = replace(%text, "cat", "dog")
    print(%newText)
end

%out = run("echo Hello from shell")
print(%out)
```

## Execution notes

- Scripts run in isolated environments (each module has its own interpreter instance).
- The functions `process`, `run`, `runIsolated` require the container to be running with appropriate privileges and a configured database.
- Long‑running `run` commands may block the module’s execution – use with care.
- On runtime errors (division by zero, out‑of‑bounds index, undefined variable), the script is aborted and the error message is printed to the module’s log.