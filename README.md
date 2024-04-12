# cmux
HTTP router for creating JSON APIs in Golang. Featuring built-in JSON handling and typesafe path variables etc.

## Basic HTTP Request Handling
Handle methods like Get or Post. Methods with a request body like POST, must specify the data structure which the JSON should be encoded to, as a function parameter (e.g. `cmux.Request[cmux.SomeRequestBody, *Metadatastruct] error {...`). This data is then accessed from the Body field of cmux.Request.
```go
func main() {
    m := cmux.Mux{}
    type PostData struct{
        SomeValue string `json:"some_value"`
    }
    type Md struct{}
    m.HandleFunc("/", &Md{},
        cmux.Get(func(req *cmux.Request[cmux.EmptyBody, *Md]) error {
            return nil
        }, nil),
        cmux.Post(func(req *cmux.Request[PostData, *Md]) error {
            fmt.Println("Received post value:", req.Body.SomeValue)
            return nil
        }, nil),
    )
    http.ListenAndServe("localhost:8080", &m)
}
```

## Using Path Variables
Define path variables using curly brackets in the path and retrieve values by passing a struct to HandleFunc.
The field tag "cmux" can be used to specify which path variable the field represents. Alternately path variables are saved to field names matching the path variable (case-insensitive).
Path variables can have prefixes or suffixes. Note only one variable is supported per path section (i.e. between a pair of '/').

```go
func main() {
    m := cmux.Mux{}
    type Md struct{
        City       string
        StreetName string `cmux:"street"`
    }
    m.HandleFunc("/city-{city}/{street}", &Md{},
        cmux.Get(func(req *cmux.Request[cmux.EmptyBody, *Md]) error {
            fmt.Println("city:", req.Metadata.City, "street:", req.Metadata.StreetName)
            return nil
        }, nil),
    )
    http.ListenAndServe("localhost:8080", &m)
}
```
## Responses and filtering
When a MethodHandler returns a type that implements the HTTPResponder interface (and the error interface), the HTTPRespond() method is called and the response is encoded as JSON (unless an error is returned). This can used to filter secret fields.

```go
type ResData struct {
    PublicData  string `json:"public_data"`
    PrivateData string `json:"private_data,omitempty"`
}

func (r *ResData) HTTPRespond() (any, error) {
    return &ResData{
        PublicData: r.PublicData,
    }, nil
}

func (r *ResData) Error() string {
    return "not filtered"
}

func main() {
    m := cmux.Mux{}
    type Md struct{}
    m.HandleFunc("/info", &Md{},
        cmux.Get(func(req *cmux.Request[cmux.EmptyBody, *Md]) error {
            return &ResData{
                PublicData: "some public data",
                PrivateData: "some private data",
            }
        }, nil),
    )
    http.ListenAndServe("localhost:8080", &m)
}

```
## Returning errors
HTTP errors can be returned directly using `cmux.HTTPError(err string, code int) error` or `cmux.WrapError(err error, code int) error` or by returning a type satisfying the HTTPErrorResponder interface.
```go
type CustomError struct{}

func (ce *CustomError) HTTPError()(int, any) {
    return 400, struct{
        AlternateError string `json:"alternate_error"`
    }{
        AlternateError: "not good",
    }
}

func (ce *CustomError) Error() string {
    return "did not respond correctly to error"
}

func main() {
    m := cmux.Mux{}
    type PostData struct{
        SomeValue string `json:"some_value"`
    }
    type Md struct{}
    m.HandleFunc("/", &Md{},
        cmux.Get(func(req *cmux.Request[cmux.EmptyBody, *Md]) error {
            return cmux.HTTPError("", http.StatusNotFound)
        }, nil),
        cmux.Post(func(req *cmux.Request[PostData, *Md]) error {
            return cmux.WrapError(errors.New("something bad happened"), http.StatusInternalServerError)
        }, nil),
        cmux.Put(func(req *cmux.Request[PostData, *Md]) error {
            return &CustomError{}
        }, nil),
    )
    http.ListenAndServe("localhost:8080", &m)
}
```

## Before and Method Handler Data
Each Method Handler can be passed a custom data argument, which is processed by the Mux's Before method. This could be a simple string for access control list or permissions handling or a struct containing more complex data. Here we require requests to (`"http://localhost:8080/cities/{city}"`) to have pass a `"{city}_mayor"` token in the Token HTTP request header:

```go
func main() {
    type Md struct{
        City string
    }
    m := cmux.Mux{
        Before: func(res http.ResponseWriter, req *http.Request, metadata, methodData any) error {
            switch v := metadata.(type) {
            case *Md:
                permission, ok := methodData.(string)
                if !ok {
                    panic("expected a permission string, got something else")
                }
                if req.Header.Get("token") != v.City + "_" + permission {
                    return cmux.HTTPError("", http.StatusForbidden)
                }
            default:
                return fmt.Errorf("unexpected metadata type %T", v)
            }
            return nil
        },
    }
    m.HandleFunc("/cities/{city}", &Md{},
        cmux.Get(func(req *cmux.Request[cmux.EmptyBody, *Md]) error {
            return nil
        }, "mayor"),
    )
    http.ListenAndServe("localhost:8080", &m)
}
```
Performing the following curl command in a terminal will then yield a 200 response code:
```console
curl -v localhost:8080/cities/london -H 'Token: london_mayor'
```
