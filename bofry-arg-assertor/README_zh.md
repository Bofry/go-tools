bofry-arg-assertor
================
針對指定的參數型別自動產生 Assertor 並為參數型別提供合適的驗證操作指令。

## **使用方式**
- **步驟一**： 準備要製作 Assertor 的 Argv 型別，此處樣本提供 protagonistArgv.go 檔案，並包含要製作使用的 ProtagonistArgv 型別，其內容如下：
    ```go
    // protagonistArgv.go
    package test

    type ProtagonistArgv struct {
        Name   string   `json:"id"`
        Age    int      `json:"age"`
    }
    ```
- **步驟二**： 修改型別的定義，加入標籤指示標記內容 `/* tag=json */` 到型別名稱與型別定義之間，如下所示：
    ```go
    type ProtagonistArgv struct /* tag=json */ {
        Name   string   `json:"id"`
        Age    int      `json:"age"`
    }
    ```
- **步驟三**： 在終端機使用下面的命令。產生 Assertor 的型別與原始碼檔案。
    ```bash
    $ bofry-arg-assertor -path protagonistArgv.go
    ```
    📝 將會產生一個 protagonistArgvAssertor_gen.go 的檔案。
- **步驟四**： 步驟三完成後，我們可以在 protagonistArgv.go 中使用 Assertor 的功能驗證參數。使用前需先匯入套件引用指示 `"github.com/Bofry/arg"`，同時建立要驗證的方法函式如下：
    ```go
    import (
        "github.com/Bofry/arg"
    )
    ```
    ```go
    func (argv *ProtagonistArgv) Validate() error {
        assertor := argv.Assertor()
        err := arg.Assert(
            assertor.Name(
                arg.Strings.NonEmpty,
            ),
            assertor.Age(
                arg.Ints.NonNegativeInteger,
                arg.Ints.NonZero,
            ),
        )

        return err
    }
    ```

## **技術規格**
1. **來源檔案**：
   1. 指定的來源**型別名稱**後綴必須為 **Argv**。 例如： **XxxxArgv**。
2. **目的檔案**：
   1. 工具會依據來源型別，產生後綴為 **ArgvAssertor** 的型別，並產生以來源檔案名稱添加後綴 **Assertor_gen.go** 的檔名。
3. **tag 指示標記**：
   > 💬 tag 標記的作用是在參數驗證失敗時，參數名稱會顯示在抛出的異常訊息內容中。
   1. 作用於指定型別內的所有欄位：
      
      ⠿ 語法 `tag=<your tag name>` 指定在下面所示的位置上。
        ```go
        type XxxxArgv struct /* tag=<your tag name> */ {
            ...
        }
        ```
        **範例 1**：下面範例使用 json 作為 tag 標記，因此 `id` 會被用於 **JsonArgv.ID** 欄位。
        ```go
        type JsonArgv struct /* tag=json */ {
            ID  string  `json:"id"   query:"_id"`
        }
        ```
   2. 作用於個別欄位：

        ⠿ 在欄位的 tag 內使用 `^` 標記指定要使用的標記名稱。
        ```go
        type XxxxArgv struct {
            XxxxField  <fieldtype>  `...    ^:"<your tag name>"`
        }
        ```
        **範例 1**: ID 欄位指定要使用的標記名稱 `json`。因此 `id` 會被用於 **JsonArgv.ID** 欄位。
        ```go
        type JsonArgv struct {
            ID  string  `json:"id"   query:"_id"   ^:"json"`
        }
        ```
4. golang 型別與支援的 **arg.ValueAssertion** 類型 
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

