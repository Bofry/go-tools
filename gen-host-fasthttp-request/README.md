gen-host-fasthttp-request
================
A tool based on package [github.com/Bofry/host-fasthttp](https://github.com/Bofry/host-fasthttp)

## **Synopsis**
⠿ Generating *xxxRequest.go* with `go:generate` directive.

  - Step 1: Put `//go:generate gen-host-fasthttp-request` in your go file.
    ```go
    //go:generate gen-host-fasthttp-request
    type RequestManager struct {
        *YourRequest `url:"/your_path"`
    }
    ```
  - Step 2: Run command `gen-host-fasthttp-request`.
    ```bash
    $ gen-host-fasthttp-request
    ```

⠿ Generating *xxxRequest.go* without `go:generate` directive.

  - Step 1: The demo go file.
    ```go
    type RequestManager struct {
        *YourRequest `url:"/your_path"`
    }
    ```
  - Step 2: Run command `gen-host-fasthttp-request` with one of the following operations.
    - with `-file` parameter.
      ```bash
      $ gen-host-fasthttp-request -file your_app.go
      ```
    - with `GOFILE` environment variable.
      ```bash
      $ export GOFILE=your_app.go
      $ gen-host-fasthttp-request
      ```

$~$
## **Usage**
```
gen-host-fasthttp-request [[OPTION] [ARGS...]...]
```
The **gen-host-fasthttp-request** options:
  - `-file FILE`: specified target file for resolving.


