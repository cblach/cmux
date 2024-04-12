// Copyright 2024 Christian Thorseth Blach. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package cmux
import (
    "io"
    "net/http"
    "net/http/httptest"
    "sync/atomic"
    "strings"
    "strconv"
    "testing"
)

var testId atomic.Uint64

func mkTestId() string {
    return strconv.FormatUint(testId.Add(1), 101)
}

func rBody(r io.Reader) string {
    rawBody, _ := io.ReadAll(r)
    return string(rawBody)
}

type EmptyType struct{}

func TestMethods(t *testing.T) {
    test := func(desc, method, body string) {
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
    }
    test("test DELETE", "DELETE", "")
    test("test GET", "GET", "")
    test("test HEAD", "HEAD", "")
    test("test OPTIONS", "OPTIONS", "")
    test("test POST", "POST", "{}")
    test("test PUT", "PUT", "{}")
    test("test Trace", "TRACE", "")
}

func TestPath(t *testing.T) {
    type D struct {
        Var1     string `cmux:"xyz_var1"`
        OtherVar string
    }
    test := func(desc, handlePath, requestPath string, expData D) {
        t.Run(desc, func(t *testing.T) {
            m := Mux{}
            m.HandleFunc(handlePath, &D{},
                Get(func(req *Request[EmptyBody, *D]) error {
                    if expData != *req.Metadata {
                        t.Errorf("expected variable do not match captured request variables %v != %v",
                                 expData, *req.Metadata)
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
    test("var", "/{xyz_var1}", "/somevar1", D{Var1: "somevar1"})
    test("prefix", "/prefix{xyz_var1}", "/prefixabc", D{Var1: "abc"})
    test("suffix", "/{xyz_var1}suffix", "/z1yxsuffix", D{Var1: "z1yx"})
    test("prefix and suffix", "/prefix{xyz_var1}suffix", "/prefixz1yxsuffix", D{Var1: "z1yx"})
    test("multiple vars", "/prefix{xyz_var1}/{othervar}", "/prefixabc/x", D{Var1: "abc", OtherVar: "x"})
    test("multiple of same var", "/{othervar}a/b{othervar}", "/abca/bx", D{OtherVar: "x"})
    test("empty variable", "/prefix{xyz_var1}/{othervar}", "/prefix/x", D{Var1: "", OtherVar: "x"})
    test("deeply nested", "/aaa/bbb/ccc/ddd/eee/fff{othervar}", "/aaa/bbb/ccc/ddd/eee/fffx", D{Var1: "", OtherVar: "x"})
}
