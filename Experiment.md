## Experimentation ##

Introducing symbol **"!"** as short panic and **"?"** as short return
syntax to improve error handling and various part of the language.

### Error Handling ###

Both **!** and **?** can be use to shorten error detection in case
there is no need wrap error with the new message.

For example, use **!** to panic immediately if error is not nil

```go
name, err! := findStudent()
```

Equivalent to

```go
name, err := findStudent()
if err != nil {
    panic(err)
}
```

use **?** to return immediately if error is not nil

```go
// the compile will generate return statement appropriate to
// the function signature
name, err? := findStudent()
```

Equivalent to

```go
name, err := findStudent()
if err != nil {
    return err
}
```

To wrap error with extra message, it's recommend to use the same old
scenario.

```go
name, err := findStudent()
if err != nil {
    return fmt.Errorf("%w: additional message", err)
}
```

### Boolean handling ###

The syntax also work with boolean variable for example:

A response from a function:
```go
name, existed? := findPatient()
```

Equivalent to

```go
name, existed := findPatient()
if !existed {
    return errors.New(fmt.Sprintf("findPatient return false"))
}
```

A variable define by Map or Cast:
```go
name, ok? := dict["not-existed"]
```

Equivalent to

```go
name, ok := dict["not-existed"]
if !ok {
    return errors.New(fmt.Sprintf("key not-existed doest not exist in map dict"))
}
```

A select case channel:
```go
select {
case name, ok? := <- studentChan:
}
```

Equivalent to

```go
select {
case name, ok := <- studentChan:
    if !ok {
        return
    }
}
```

### Improvement ##

**Need to provide addition message out of the box.**