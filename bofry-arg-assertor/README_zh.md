<div style="text-align: right">
<a href="README.md">EN</a>
</div>

bofry-arg-assertor
================
提供基於 [github.com/Bofry/arg](https://github.com/Bofry/arg) 套件產生參數驗證的程式碼工具。

## **使用方式**
⠿ 複製下面提供的程式碼，並保存到 **protagonistArgv.go** 檔案。樣本提供要製作使用的 **ProtagonistArgv** 型別，稍後介紹如何使用工具產生 **ProtagonistArgvAssertor** 型別於 **ProtagonistArgvAssertor_gen.go** 檔案，以及如何在 **ProtagonistArgv** 撰寫驗證。
```go
// protagonistArgv.go
package test

type ProtagonistArgv struct {
    Name   string   `json:"id"`
    Age    int      `json:"age"`
}
```

### **操作步驟**
- **步驟一**： 修改型別的定義，加入標籤指示標記內容 `/* tag=json */` 到型別名稱與型別定義之間，如下所示：
    ```go
    type ProtagonistArgv struct /* tag=json */ {
        Name   string   `json:"id"`
        Age    int      `json:"age"`
    }
    ```
- **步驟二**： 在終端機使用下面的命令。產生 **ProtagonistArgvAssertor** 的型別與 **protagonistArgvAssertor_gen.go** 檔案。
    ```bash
    $ bofry-arg-assertor -path protagonistArgv.go
    ```
- **步驟三**： 步驟二完成後，我們可以在 **ProtagonistArgv** 中使用 **ProtagonistArgvAssertor** 型別提供的參數驗證方法。使用前需先匯入套件引用指示 `"github.com/Bofry/arg"`，同時建立要驗證的方法函式如下：
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
    > 📝 **ProtagonistArgvAssertor** 會產生一組方法，提供 **ProtagonistArgv** 驗證操作。其方法名稱即 **ProtagonistArgv** 的欄位名稱。
    >
    > 🐾 參數型別與 arg.ValueAssertion 的型別對照表參考：[*golang* 型別與支援的 *arg.ValueAssertion* 類型](#golang_vs_arg.ValueAssertion)。

## **技術規格**
1. **來源檔案**：
   1. 指定的來源**型別名稱**後綴必須為 **Argv**。 例如： **XxxxArgv**。
2. **目的檔案與目的型別**：
   1. 工具會依據來源型別，產生後綴為 **Assertor** 的型別。
   2. 目的檔案會以來源檔案名稱添加後綴 **Assertor_gen.go** 來命名。
3. **tag 指示標記**：
   > 💬 tag 標記的作用是顯示在參數驗證失敗時，參數名稱會顯示在抛出的異常訊息內容中。
   1. **作用於指定型別內的所有欄位**：
      
      ⠿ 語法 `tag=<your tag name>` 指定在下面所示的位置上。
        ```go
        type XxxxArgv struct /* tag=<your tag name> */ {
            ...
        }
        ```
        **範例**：下面範例使用 json 作為 tag 標記，因此 `id` 會被用於 **JsonArgv.ID** 欄位。
        ```go
        type JsonArgv struct /* tag=json */ {
            ID  string  `json:"id"   query:"_id"`
        }
        ```
   2. **作用於個別欄位**：

        ⠿ 在欄位的 tag 內使用 `^` 標記指定要使用的標記名稱。
        ```go
        type XxxxArgv struct {
            XxxxField  <fieldtype>  `...    ^:"<your tag name>"`
        }
        ```
        **範例**: ID 欄位指定要使用的標記名稱 `json`。因此 `id` 會被用於 **JsonArgv.ID** 欄位。
        ```go
        type JsonArgv struct {
            ID  string  `json:"id"   query:"_id"   ^:"json"`
        }
        ```
4. ***golang* 型別與支援的 *arg.ValueAssertion* 類型** <a id="golang_vs_arg.ValueAssertion"></a>
   | arg.ValueAssertion   | golang 型別 |
   |:---------------------|:-------------|
   | --                   | `bool`
   | **StringAssertion**  | `string`
   | **IntAssertion**     | `int` `int8` `int16` `int32` `int64`
   | **UIntAssertion**    | `uint` `uint8` `uint16` `uint32` `uint64`
   | **FloatAssertion**   | `float32` `float64`
   | **IPAssertion**      | `net.IP` -or- `arg.IP`
   | **NumberAssertion**  | `encoding/json.Number` -or- `arg.Number`
   | **ValueAssertion**   | *除上面列出的所有型別*

