⚠️ This tool is DEPRECATED!
> 📝 The tool support **host-fasthttp** v0.1.x and older.
>
> 🐾 Use [gen-host-fasthttp-request](../gen-host-fasthttp-request) for **host-fasthttp** v0.2.x and newer.


gen-host-fasthttp-resource
================
A tool based on package [github.com/Bofry/host-fasthttp](https://github.com/Bofry/host-fasthttp) to genreating incipient request go files.


## **Synopsis**
⠿ Generating *xxxResource.go* with `go:generate` directive.

  - Step 1: Put `//go:generate gen-host-fasthttp-resource` in your go file.
    ```go
    //go:generate gen-host-fasthttp-resource
    type ResourceManager struct {
        *YourResource `url:"/your_path"`
    }
    ```
  - Step 2: Run command `gen-host-fasthttp-resource`.
    ```bash
    $ gen-host-fasthttp-resource
    ```

⠿ Generating *xxxResource.go* specified target go file.

  - Step 1: The demo go file.
    ```go
    type ResourceManager struct {
        *YourResource `url:"/your_path"`
    }
    ```
  - Step 2: Run command `gen-host-fasthttp-resource` with one of the following operations.
    ```bash
    $ export GOFILE=your_app.go
    $ gen-host-fasthttp-resource
    ```

