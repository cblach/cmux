# cmux
HTTP Router for creating JSON APIs in Golang. Featuring built-in JSON handling and typesafe path variables etc.

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
        cmux.Get(func(w http.ResponseWriter, req *cmux.Request[cmux.EmptyBody, *Md]) error {
            return nil
        }, nil),
        cmux.Post(func(w http.ResponseWriter, req *cmux.Request[PostData, *Md]) error {
            fmt.Println("Received post value:", req.Body.SomeValue)
            return nil
        }, nil),
    )
    http.ListenAndServe("localhost:8080", &m)
}
```

# Using Path Variables
Define path variables using curly brackets in the path and retrieve values by passing a struct to HandleFunc.
The field tag "cmux" can be used to specify which path variable the field represents. Alternately path variables are saved to field names matching the path variable (case-insensitive).
Path variables can have prefixes or suffixes, note only one variable is supported per path section (i.e. between a pair of '/').

```go
func main() {
    m := cmux.Mux{}
    type Md struct{
        City       string
        StreetName string `cmux:"street"`
    }
    m.HandleFunc("/city-{city}/{street}", &Md{},
        cmux.Get(func(w http.ResponseWriter, req *cmux.Request[cmux.EmptyBody, *Md]) error {
            fmt.Println("city:", req.Metadata.City, "street:", req.Metadata.StreetName)
            return nil
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
        cmux.Get(func(w http.ResponseWriter, req *cmux.Request[cmux.EmptyBody, *Md]) error {
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
