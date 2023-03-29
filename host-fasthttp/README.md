host-fasthttp
================
A tool based on package [github.com/Bofry/host-fasthttp](https://github.com/Bofry/host-fasthttp) to genreating incipient web API projects.

## **Synopsis**
⠿ Generating an incipient new web API project *mywebapi*.
```bash
$ ./host-fasthttp init mywebapi
```

⠿ Generating an incipient new web API project under folder *mywebapi*. Using `.` to instead project folder name as application module name.
```bash
$ mkdir mywebapi
$ cd mywebapi
$ ./host-fasthttp init .
```

⠿ Generating an incipient new web API project using existing `go.mod` file.
```bash
$ go mod init mywebapi
$ ./host-fasthttp init
```

$~$
## **Usage**
```
host-fasthttp COMMAND [ARGS...] [OPTIONS...]
```

  - `init` : create new host-fasthttp project.
    > **usage:**
    > ```
    > http-fasthttp init [MODULE_NAME] [OPTIONS...]
    > ```
    > **arguments:**
    > - `MODULE_NAME`:  the go module name for the application.<br/>
    **NOTE:** The `MODULE_NAME` can use `.` period symbol to apply current working directory name.
    >
    > **options:**
    > - `-v VERSION`: the host-fasthttp version.
    >
  - `help` : show usage.





