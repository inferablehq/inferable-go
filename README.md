# Inferable Go Client

Inferable Go Client is a Go package that provides a client for interacting with the Inferable API. It allows you to register your go services against the Inferable control plane.

## Installation

To install the Inferable Go Client, use the following command:

```
go get github.com/inferablehq/inferable-go
```

## Usage

### Initializing the Client

To create a new Inferable client, use the `New` function:

```go
import "github.com/inferablehq/inferable-go/inferable"

client, err := inferable.New("your-api-secret", "https://api.inferable.ai")
if err != nil {
    // Handle error
}
```

If you don't provide an API endpoint, it will use the default endpoint: `https://api.inferable.ai`.

### Registering a Service

To register a service:

```go
service, err := client.RegisterService("MyService")
if err != nil {
    // Handle error
}
```

### Registering a Function

After registering a service, you can register functions within that service:

```go
type MyInput struct {
    Message string `json:"message"`
}

myFunc := func(input MyInput) string {
    return "Hello, " + input.Message
}

err := service.RegisterFunc(inferable.Function{
    Func:        myFunc,
    Name:        "MyFunction",
    Description: "A simple greeting function",
})
if err != nil {
    // Handle error
}
```

### Starting the Service

To start the service and begin listening for incoming requests:

```go
err := service.Start()
if err != nil {
    // Handle error
}
```

### Stopping the Service

To stop the service:

```go
service.Stop()
```

### Checking Server Health

To check if the Inferable server is healthy:

```go
err := client.ServerOk()
if err != nil {
    // Handle error
}
```

## Contributing

Contributions to the Inferable Go Client are welcome. Please ensure that your code adheres to the existing style and includes appropriate tests.

## Support

For support or questions, please [create an issue in the repository](https://github.com/inferablehq/inferable-go/issues).
