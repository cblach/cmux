// Copyright 2024 Christian Thorseth Blach. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package cmux
import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "math"
    "net/http"
    "net/http/httptest"
    "reflect"
    "strings"
    "testing"
)

func rBody(r io.Reader) string {
    rawBody, _ := io.ReadAll(r)
    return string(rawBody)
}

type EmptyType struct{}

func TestMethods(t *testing.T) {
    testMethod := func(method, body string) {
        t.Run("test " + method, func(t *testing.T) {
            m := Mux{}
            m.HandleFunc("/", &EmptyType{},
                Delete(func(req *Request[EmptyBody, *EmptyType]) error {
                    if method != "DELETE" {
                        t.Errorf("unexpected method %s", method)
                    }
                    return nil
                }, nil),
                Get(func(req *Request[EmptyBody, *EmptyType]) error {
                    if method != "GET" {
                        t.Errorf("unexpected method %s", method)
                    }
                    return nil
                }, nil),
                Head(func(req *Request[EmptyBody, *EmptyType]) error {
                    if method != "HEAD" {
                        t.Errorf("unexpected method %s", method)
                    }
                    return nil
                }, nil),
                Options(func(req *Request[EmptyBody, *EmptyType]) error {
                    if method != "OPTIONS" {
                        t.Errorf("unexpected method %s", method)
                    }
                    return nil
                }, nil),
                Post(func(req *Request[EmptyBody, *EmptyType]) error {
                    if method != "POST" {
                        t.Errorf("unexpected method %s", method)
                    }
                    return nil
                }, nil),
                Put(func(req *Request[EmptyBody, *EmptyType]) error {
                    if method != "PUT" {
                        t.Errorf("unexpected method %s", method)
                    }
                    return nil
                }, nil),
                Trace(func(req *Request[EmptyBody, *EmptyType]) error {
                    if method != "TRACE" {
                        t.Errorf("unexpected method %s", method)
                    }
                    return nil
                }, nil),
            )
            req, err := http.NewRequest(method, "/", strings.NewReader(body))
            if err != nil {
                t.Errorf("http.NewRequest failed: %v", err)
                return
            }
            rec := httptest.NewRecorder()
            m.ServeHTTP(rec, req)
            if rec.Code != 200 {
                t.Errorf("unexpected response code %d, expected %d: %s", rec.Code, 200, rBody(rec.Body))
                return
            }
        })
    }
    testMethod("DELETE", "")
    testMethod("GET", "")
    testMethod("HEAD", "")
    testMethod("OPTIONS", "")
    testMethod("POST", "{}")
    testMethod("PUT", "{}")
    testMethod("TRACE", "")
}

func TestPath(t *testing.T) {
    type MD struct {
        Var1     string `cmux:"xyz_var1"`
        OtherVar string
        Uintvar  uint
        U64var   uint64
        U32var   uint32
        U16var   uint16
        U8var    uint8
        Intvar   int
        I64var   int64
        I32var   int32
        I16var   int16
        I8var    int8
    }
    testPath := func(desc, handlePath, requestPath string, expMetadata MD) {
        t.Run(desc, func(t *testing.T) {
            m := Mux{}
            m.HandleFunc(handlePath, &MD{},
                Get(func(req *Request[EmptyBody, *MD]) error {
                    if expMetadata != *req.Metadata {
                        t.Errorf("expected variable do not match captured request variables %v != %v",
                                 expMetadata, *req.Metadata)
                    }
                    return nil
                }, ""),
            )
            req, err := http.NewRequest("GET", requestPath, nil)
            if err != nil {
                t.Errorf("http.NewRequest failed: %v", err)
                return
            }
            rec := httptest.NewRecorder()
            m.ServeHTTP(rec, req)
            if rec.Code != 200 {
                t.Errorf("unexpected response code %d, expected %d", rec.Code, 200)
                return
            }
        })
    }
    testPath("string var", "/{xyz_var1}", "/somevar1", MD{Var1: "somevar1"})

    testPath("uint var", "/{uintvar}", fmt.Sprintf("/%d", uint(math.MaxUint)), MD{Uintvar: math.MaxUint})
    testPath("uint64 var", "/{u64var}", fmt.Sprintf("/%d", uint64(math.MaxUint64)), MD{U64var: math.MaxUint64})
    testPath("uint32 var", "/{u32var}", fmt.Sprintf("/%d", uint32(math.MaxUint32)), MD{U32var: math.MaxUint32})
    testPath("uint16 var", "/{u16var}", fmt.Sprintf("/%d", uint16(math.MaxUint16)), MD{U16var: math.MaxUint16})
    testPath("uint8 var", "/{u8var}", fmt.Sprintf("/%d", uint8(math.MaxUint8)), MD{U8var: math.MaxUint8})

    testPath("int var", "/{intvar}", fmt.Sprintf("/%d", math.MaxInt), MD{Intvar: math.MaxInt})
    testPath("int64 var", "/{i64var}", fmt.Sprintf("/%d", int64(math.MaxInt64)), MD{I64var: math.MaxInt64})
    testPath("int32 var", "/{i32var}", fmt.Sprintf("/%d", int32(math.MaxInt32)), MD{I32var: math.MaxInt32})
    testPath("int16 var", "/{i16var}", fmt.Sprintf("/%d", int16(math.MaxInt16)), MD{I16var: math.MaxInt16})
    testPath("int8 var", "/{i8var}", fmt.Sprintf("/%d", int8(math.MaxInt8)), MD{I8var: math.MaxInt8})

    testPath("negative int var", "/{intvar}", fmt.Sprintf("/%d", math.MinInt), MD{Intvar: math.MinInt})
    testPath("negative int64 var", "/{i64var}", fmt.Sprintf("/%d", int64(math.MinInt64)), MD{I64var: math.MinInt64})
    testPath("negative int32 var", "/{i32var}", fmt.Sprintf("/%d", int32(math.MinInt32)), MD{I32var: math.MinInt32})
    testPath("negative int16 var", "/{i16var}", fmt.Sprintf("/%d", int16(math.MinInt16)), MD{I16var: math.MinInt16})
    testPath("negative int8 var", "/{i8var}", fmt.Sprintf("/%d", int8(math.MinInt8)), MD{I8var: math.MinInt8})


    testPath("prefix", "/prefix{xyz_var1}", "/prefixabc", MD{Var1: "abc"})
    testPath("suffix", "/{xyz_var1}suffix", "/z1yxsuffix", MD{Var1: "z1yx"})
    testPath("prefix and suffix", "/prefix{xyz_var1}suffix", "/prefixz1yxsuffix", MD{Var1: "z1yx"})
    testPath("multiple vars", "/prefix{xyz_var1}/{othervar}", "/prefixabc/x", MD{Var1: "abc", OtherVar: "x"})
    testPath("multiple of same var", "/{othervar}a/b{othervar}", "/abca/bx", MD{OtherVar: "x"})
    testPath("empty variable", "/prefix{xyz_var1}/{othervar}", "/prefix/x", MD{Var1: "", OtherVar: "x"})
    testPath("deeply nested", "/aaa/bbb/ccc/ddd/eee/fff{othervar}", "/aaa/bbb/ccc/ddd/eee/fffx", MD{Var1: "", OtherVar: "x"})
}

func testPost[T any](t *testing.T, desc string, data any) {
    t.Run(desc, func(t *testing.T) {
        m := Mux{}
        type MD struct{}
        m.HandleFunc("/", &MD{},
            Post(func(req *Request[T, *MD]) error {
                if !reflect.DeepEqual(data, req.Body) {
                    t.Errorf("body mismatch %v != %v",
                             data, req.Body)
                }
                return nil
            }, ""),
        )
        var sentBody []byte
        if b, ok := data.([]byte); ok {
            sentBody = b
        } else {
            var err error
            sentBody, err = json.Marshal(data)
            if err != nil {
                t.Errorf("json.Marshal failed: %v", err)
                return
            }
        }
        req, err := http.NewRequest("POST", "/", bytes.NewReader(sentBody))
        if err != nil {
            t.Errorf("http.NewRequest failed: %v", err)
            return
        }
        rec := httptest.NewRecorder()
        m.ServeHTTP(rec, req)
        if rec.Code != 200 {
            t.Errorf("unexpected response code %d, expected %d", rec.Code, 200)
            return
        }
    })
}

func TestPost(t *testing.T) {
    type BasicPost struct {
        A string `json:"a"`
        B string `json:"b"`
    }
    testPost[BasicPost](t, "struct", BasicPost{A: "A str", B: "b123 & 123"})
    testPost[string](t, "string", "abc")
    testPost[uint](t, "uint", uint(10))
    testPost[int](t, "int", 10)
    testPost[[]byte](t, "bytes", []byte{'a', 'b', 'c'})
}

type ResA struct {
    A      string `json:"stra,omitempty"`
    B      string `json:"strb,omitempty"`
    Secret string `json:"secret,omitempty"`

    Fn   func(any) (any, error) `json:"-"`
}

func (ra *ResA) HTTPRespond() (any, error) {
    return ra.Fn(ra)
}

func (ra *ResA) Error() string {
    return "httprespond not called"
}

type Card struct {
    CardNumber string `json:"card_number"`
    Expiry     string `json:"expiry"`
    CVC        string `json:"cvc"`
}

func (c *Card) HTTPRespond() (any, error) {
    if len(c.CardNumber) == 0 {
        return nil, errors.New("empty card_number")
    }
    typ := "unknown"
    if c.CardNumber[0] == '4' {
        typ = "visa"
    }
    return struct{
        CardType string `json:"card_type"`
        Expiry   string `json:"expiry"`
    }{
        CardType: typ,
        Expiry:   c.Expiry,
    }, nil
}

func (c *Card) Error() string {
    return "httprespond not called"
}

func TestResponse(t *testing.T) {
    testRes := func(desc string, returnData error, expBody string) {
        t.Run(desc, func(t *testing.T) {
            m := Mux{}
            type MD struct {}
            m.HandleFunc("/", &MD{},
                Get(func(req *Request[EmptyBody, *MD]) error {
                    return returnData
                }, ""),
            )
            req, err := http.NewRequest("GET", "/", nil)
            if err != nil {
                t.Errorf("http.NewRequest failed: %v", err)
                return
            }
            rec := httptest.NewRecorder()
            m.ServeHTTP(rec, req)
            if rec.Code != 200 {
                t.Errorf("unexpected response code %d, expected %d", rec.Code, 200)
                return
            }
            recvdBody := strings.TrimSpace(rBody(rec.Body))
            if recvdBody != expBody {
                t.Errorf("unexpected data, got: %s", recvdBody)
                return
            }
        })
    }
    testRes("basic (no mutation)",
            &ResA{A: "somestr", B: "xyz", Fn: func(d any) (any, error) { return d, nil }},
           `{"stra":"somestr","strb":"xyz"}`)
    testRes("bypass", Bypass(&struct{A uint}{1203}), `{"A":1203}`)
    testRes("filter",
            &ResA{A: "astr", B: "a_23$", Secret: "somesecret",
                  Fn: func(d any) (any, error) {
                    return ResA{
                        A: d.(*ResA).A,
                        B: d.(*ResA).B,
                    }, nil
                },
            }, `{"stra":"astr","strb":"a_23$"}`)
    testRes("transform",
            &Card{CardNumber: "4111 1111 1111 1111", Expiry: "11/2030", CVC: "123"},
            `{"card_type":"visa","expiry":"11/2030"}`)
    testRes("byte response",
            &ResA{A: "a",
                  Fn: func(d any) (any, error) {
                      return []byte{'a', 'b', 'c'}, nil
                  },
            }, `abc`)
}
/*
func testErrors(t *testing.T) {

}
*/
/*
func testBefore(t *testing.T) {
    type MD struct {
        Var1     string `cmux:"xyz_var1"`
        OtherVar string
    }
    testBefore := func(desc string, expMetadata MD) {
        t.Run(desc, func(t *testing.T) {
            m := Mux{}
            m.HandleFunc(handlePath, &MD{},
                Get(func(req *Request[EmptyBody, *MD]) error {
                    if expMetadata != *req.Metadata {
                        t.Errorf("expected variable do not match captured request variables %v != %v",
                                 expMetadata, *req.Metadata)
                    }
                    return nil
                }, ""),
            )
            req, err := http.NewRequest("GET", requestPath, nil)
            if err != nil {
                t.Errorf("http.NewRequest failed: %v", err)
                return
            }
            rec := httptest.NewRecorder()
            m.ServeHTTP(rec, req)
            if rec.Code != 200 {
                t.Errorf("unexpected response code %d, expected %d", rec.Code, 200)
                return
            }
        })
    }
}
*/