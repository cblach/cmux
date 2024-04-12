// Copyright 2024 Christian Thorseth Blach. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package cmux
import(
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "reflect"
    "runtime"
)

const(
    inputTypeNone = iota
    inputTypeBytes
    inputTypeStruct
)

type MethodHandler struct {
    Method   string
    Func     func(http.ResponseWriter, *http.Request, any) error
    Data     any
    Mux      *Mux /* the leaf-node mux respponisble for the handler */

    /* for debug purposes: */
    FuncName string
}

type EmptyBody struct{}

type Request[T any, M any] struct {
    Body T
    Metadata M

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
    if inputType == inputTypeNone {
        t := reflect.TypeOf(input)
        if t.Kind() == reflect.Struct {
            inputType = inputTypeStruct
        } else {
            panic("cmux: cannot handle type " + t.String())
        }

    }

    return func(w http.ResponseWriter, httpReq *http.Request, md any) error {
        req := Request[I, M]{
            HTTPReq: httpReq,
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
        } else if inputType == inputTypeStruct {
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

func Delete[I EmptyBody, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        Method: "DELETE",
        Func: getEmptyBodyHandler(fn, data),
        Data: data,
    }
}

func Get[I EmptyBody, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        Method: "GET",
        Func: getEmptyBodyHandler(fn, data),
        Data: data,
    }
}

func Head[I EmptyBody, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        Method: "HEAD",
        Func: getEmptyBodyHandler(fn, data),
        Data: data,
    }
}

func Options[I EmptyBody, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        Method: "OPTIONS",
        Func: getEmptyBodyHandler(fn, data),
        Data: data,
    }
}

func Patch[I any, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        Method: "PATCH",
        Func: getHandler(fn, data),
        Data: data,
    }
}

func Post[I any, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        Method: "POST",
        Func: getHandler(fn, data),
        Data: data,
    }
}

func Put[I any, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        Method: "PUT",
        Func: getHandler(fn, data),
        Data: data,
    }
}

func Trace[I EmptyBody, M any] (fn func(*Request[I, M]) error, data any) MethodHandler {
    return MethodHandler{
        Method: "TRACE",
        Func: getEmptyBodyHandler(fn, data),
        Data: data,
    }
}

func (mux *Mux) HandleFunc(path string, metadata any, mhs ...MethodHandler) {
    if reflect.TypeOf(metadata) == methodHandlerType {
        panic("missing metadata argument")
    }
    methodHandlers := map[string]*MethodHandler{}
    for i, mh := range mhs {
        mh.FuncName = runtime.FuncForPC(reflect.ValueOf(mh.Func).Pointer()).Name()
        methodHandlers[mh.Method] = &mhs[i]
    }
    mux.mkRoute(path, metadata, methodHandlers)
}

func HandleFunc(path string, metadata any, mhs ...MethodHandler) {
    DefaultMux.HandleFunc(path, metadata, mhs...)
}

func (mux *Mux) SetDefaultContentType(ctype string) {
    mux.dfltContentType = ctype
}
