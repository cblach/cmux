// Copyright 2024 Christian Thorseth Blach. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package cmux
import(
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "reflect"
    "runtime"
)

const(
    inputTypeAny = iota
    inputTypeBytes
)

// MethodHandlers each handles a specific HTTP Method. They are returned
// by the functions Delete, Get, Head, Options, Patch, Post, Put, Trace.
type MethodHandler struct {
    method string
    fn     func(http.ResponseWriter, *http.Request, any) error
    data   any
    mux    *Mux /* the leaf-node mux respponisble for the handler */

    /* for debug purposes: */
    fnName string
}

type EmptyBody struct{}

// Request stores incoming request data.
// Body contains the unmarshaled body of the request
// Metadata contains custom data that is passed to the
// HandleFunc and can be mutated by the mux Before Method.
// It also provides access to the underlying http.Request and
// http.ResponseWriter
type Request[T any, M any] struct {
    Body T
    Metadata M
    Context context.Context


    /* Underlying native golang request / responsewriter: */
    HTTPReq *http.Request
    ResponseWriter http.ResponseWriter
}

type handleFnType func (w http.ResponseWriter, httpReq *http.Request, md any) error

func getEmptyBodyHandler[I EmptyBody, M any](fn func(*Request[I, M]) error,
                                             data any) handleFnType {
    return func (w http.ResponseWriter, httpReq *http.Request, md any) error {
        req := Request[I, M]{
            Body:          I{},
            Context:       httpReq.Context(),
            HTTPReq:       httpReq,
            ResponseWriter: w,
        }
        if md != nil {
            var ok bool
            if req.Metadata, ok = md.(M); !ok {
                return &codeResponder{
                    code:  http.StatusInternalServerError,
                    error: errors.New("unexpected metadata type"),
                }
            }
        }
        return fn(&req)
    }
}

func getHandler[I any, M any](fn func(*Request[I, M]) error,
                              data any) handleFnType {
    var inputType int
    var input I
    switch any(input).(type) {
    case []byte:
        inputType = inputTypeBytes
    }

    return func(w http.ResponseWriter, httpReq *http.Request, md any) error {
        req := Request[I, M]{
            Context:        httpReq.Context(),
            HTTPReq:        httpReq,
            ResponseWriter: w,
        }
        if md != nil {
            var ok bool
            if req.Metadata, ok = md.(M); !ok {
                return &codeResponder{
                    code:  http.StatusInternalServerError,
                    error: errors.New("unexpected metadata type"),
                }
            }
        }
        if inputType == inputTypeBytes {
            b, ok := (any(&req.Body)).(*[]byte)
            if !ok {
                panic("impossible case")
            }
            barr, err := io.ReadAll(httpReq.Body)
            if err != nil {
                return &codeResponder{
                    code:  http.StatusBadRequest,
                    error: fmt.Errorf("io.ReadAll failed: %w", err),
                }
            }
            *b = barr
        } else if inputType == inputTypeAny {
            if err := json.NewDecoder(httpReq.Body).Decode(&req.Body); err != nil {
                return &codeResponder{
                    code:  http.StatusBadRequest,
                    error: fmt.Errorf("json decoding failed: %w", err),
                }
            }
        } else {
            panic("impossible case")
        }
        return fn(&req)
    }
}

// Handle DELETE HTTP method requests.
func Delete[I EmptyBody, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        method: "DELETE",
        fn: getEmptyBodyHandler(fn, data),
        data: data,
    }
}

// Handle GET HTTP method requests.
func Get[I EmptyBody, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        method: "GET",
        fn:     getEmptyBodyHandler(fn, data),
        data:    data,
    }
}

// Handle HEAD HTTP method requests.
func Head[I EmptyBody, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        method: "HEAD",
        fn:     getEmptyBodyHandler(fn, data),
        data:   data,
    }
}

// Handle OPTIONS HTTP method requests.
func Options[I EmptyBody, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        method: "OPTIONS",
        fn:     getEmptyBodyHandler(fn, data),
        data:   data,
    }
}

// Handle PATCH HTTP method requests.
func Patch[I any, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        method: "PATCH",
        fn:     getHandler(fn, data),
        data:   data,
    }
}

// Handle POST HTTP method requests.
func Post[I any, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        method: "POST",
        fn:     getHandler(fn, data),
        data:   data,
    }
}

// Handle PUT HTTP method requests.
func Put[I any, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        method: "PUT",
        fn:     getHandler(fn, data),
        data:   data,
    }
}

// Handle TRACE HTTP method requests.
func Trace[I EmptyBody, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        method: "TRACE",
        fn:     getEmptyBodyHandler(fn, data),
        data:   data,
    }
}

// HandleFunc handles requests matching the specified path in the speciified MethodHandlers.
// The metadata is copied for each new incoming request and can be mutated by the Mux.Before
// method before being available in the MethodHandler functions.
func (mux *Mux) HandleFunc(path string, metadata any, mhs ...MethodHandler) {
    if reflect.TypeOf(metadata) == methodHandlerType {
        panic("missing metadata argument")
    }
    methodHandlers := map[string]*MethodHandler{}
    for i, mh := range mhs {
        mh.fnName = runtime.FuncForPC(reflect.ValueOf(mh.fn).Pointer()).Name()
        methodHandlers[mh.method] = &mhs[i]
    }
    mux.mkRoute(path, metadata, methodHandlers)
}

func HandleFunc(path string, metadata any, mhs ...MethodHandler) {
    DefaultMux.HandleFunc(path, metadata, mhs...)
}

func (mux *Mux) SetDefaultContentType(ctype string) {
    mux.dfltContentType = ctype
}
