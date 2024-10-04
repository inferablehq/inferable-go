# Inferable Go Client

Inferable Go Client is a Go package that provides a client for interacting with the Inferable API. It allows you to register your go functions against the Inferable control plane.

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

### Registering a Function

Register functions within Inferable using the default service.

```go
type MyInput struct {
    Message string `json:"message"`
}

err := client.Default.RegisterFunc(inferable.Function{
    Func:        myFunc,
    Name:        "MyFunction",
    Description: "A simple greeting function",
})

if err != nil {
    // Handle error
}
```

<details>

<summary>👉 The Golang SDK for Inferable reflects the types from the input struct of the function.</summary>

Unlike the TypeScript schema, the Golang SDK for Inferable reflects the types from the input struct of the function. It uses the [invopop/jsonschema](https://pkg.go.dev/github.com/invopop/jsonschema) library under the hood to generate JSON schemas from Go types through reflection.

If the input struct defines jsonschema properties using struct tags, the SDK will use those in the generated schema. This allows for fine-grained control over the schema generation.

Here's an example to illustrate this:

```go
import (
    "github.com/inferablehq/inferable-go/inferable"
    "time"
)

type UserInput struct {
    ID        int       `json:"id" jsonschema:"required"`
    Name      string    `json:"name" jsonschema:"minLength=2,maxLength=50"`
    Email     string    `json:"email" jsonschema:"format=email"`
    BirthDate time.Time `json:"birth_date" jsonschema:"format=date"`
    Tags      []string  `json:"tags" jsonschema:"uniqueItems=true"`
}

func createUser(input UserInput) string {
    // Function implementation
}

service, _ := client.RegisterService("UserService")

err := service.RegisterFunc(inferable.Function{
    Func:        createUser,
    Name:        "CreateUser",
    Description: "Creates a new user",
})

if err != nil {
    // Handle error
}
```

In this example, the UserInput struct uses jsonschema tags to define additional properties for the schema:

- The id field is marked as required.
- The name field has minimum and maximum length constraints.
- The email field is specified to be in email format.
- The birth_date field is set to date format.
- The tags field is defined as an array with unique items.

When this function is registered, the Inferable Go SDK will use these jsonschema tags to generate a more detailed and constrained JSON schema for the input.

The [invopop/jsonschema library](https://pkg.go.dev/github.com/invopop/jsonschema) provides many more options for schema customization, including support for enums, pattern validation, numeric ranges, and more.

</details>

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
