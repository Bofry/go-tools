[ZH](README_zh.md)

bofry-arg-assertor
================
A tool based on package [github.com/Bofry/arg](https://github.com/Bofry/arg) to generate soure code for given files.

## **Usage**
⠿ Copy the following sample code and save as file named **protagonistArgv.go**. The sample provides a type **ProtagonistArgv** that we will explain how to generate a **ProtagonistArgvAssertor** type and **ProtagonistArgvAssertor_gen.go** file with the tool. And how to compose validating codes under **ProtagonistArgv**.
```go
// protagonistArgv.go
package test

type ProtagonistArgv struct {
    Name   string   `json:"id"`
    Age    int      `json:"age"`
}
```

### **Quick Start**
- **Step 1**: Places the tag annotation `/* tag=json */` among struct name and stuct field list definition. As following:
    ```go
    type ProtagonistArgv struct /* tag=json */ {
        Name   string   `json:"id"`
        Age    int      `json:"age"`
    }
    ```
- **Step 2**: Executes the command on your terminal. The command will generate type **ProtagonistArgvAssertor** and file **protagonistArgvAssertor_gen.go** for later use.
    ```bash
    $ bofry-arg-assertor -path protagonistArgv.go
    ```
- **Step 3**: After the completion of Step 2, we can compose validating code with the methods of **ProtagonistArgvAssertor** under the type **ProtagonistArgv**. Before starting, the package `"github.com/Bofry/arg"` should be imported first. The validating sample code as following:
    ```go
    import (
        "github.com/Bofry/arg"
    )
    ```
    ```go
    func (argv *ProtagonistArgv) Validate() error {
        // get ProtagonistArgvAssertor instance
        assertor := argv.Assertor()
        err := arg.Assert(
            // validate ProtagonistArgv.Name
            assertor.Name(
                arg.Strings.NonEmpty,
            ),
            // validate ProtagonistArgv.Age
            assertor.Age(
                arg.Ints.NonNegativeInteger,
                arg.Ints.NonZero,
            ),
        )

        return err
    }
    ```
    > 📝 The **ProtagonistArgvAssertor** will generate a set of methods that provide the operations of validation under **ProtagonistArgv**. All name of methods are consistent with name of fields in **ProtagonistArgv**.
    >
    > 🐾 The table describes the relation about the supported arg.ValueAssertion and golang types. see [The *golang types* and supported *arg.ValueAssertion*](#golang_vs_arg.ValueAssertion)。


## **Specification**
1. **source files**:
   1. The source type name must contain suffix **Argv**. e.g:  **XxxxArgv**.
2. **destanation files and destanation types**:
   1. The tool will generate source code for specified type and the name is the suffix **Assertor** appended to source type name.
   2. The destanation file will save name as the suffix **Assertor_gen.go** appended to the original filename.
3. **tag annotation**:
   > 💬 The tag annotation provides the argument name on the error message when the given value is invalid.
   1. **applied to all fields in the entire type**:

        ⠿ Use syntax `tag=<your tag name>` place among struct name and struct field list definition. As following sample:  
        ```go
        type XxxxArgv struct /* tag=<your tag name> */ {
            ...
        }
        ```
        **example**: the tag value `id` wiil be applied to field **JsonArgv.ID**.  
        ```go
        type JsonArgv struct /* tag=json */ {
            ID  string  `json:"id"   query:"_id"`
        }
        ```
   2. **applied to specifed field**:

        ⠿ Use the tag `^` specified field tag name for applying.
        ```go
        type XxxxArgv struct {
            XxxxField  <fieldtype>  `...    ^:"<your tag name>"`
        }
        ```
        **example**: The tag `^` spcified field tag name with `json`. Therefore the tag value `id` wiil be applied to field **JsonArgv.ID**.  
        ```go
        type JsonArgv struct {
            ID  string  `json:"id"   query:"_id"   ^:"json"`
        }
        ```
4. **The *golang types* and supported *arg.ValueAssertion***  <a id="golang_vs_arg.ValueAssertion"></a>
   | arg.ValueAssertion   | golang types |
   |:---------------------|:-------------|
   | --                   | `bool`
   | **StringAssertion**  | `string`
   | **IntAssertion**     | `int` `int8` `int16` `int32` `int64`
   | **UIntAssertion**    | `uint` `uint8` `uint16` `uint32` `uint64`
   | **FloatAssertion**   | `float32` `float64`
   | **IPAssertion**      | `net.IP` -or- `arg.IP`
   | **NumberAssertion**  | `encoding/json.Number` -or- `arg.Number`
   | **Value**            | *all types except above*
