// Copyright 2024 Christian Thorseth Blach. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package cmux
import(
    "fmt"
    "io"
    "reflect"
    "runtime"
    "sort"
)

func (mux *Mux) EnableDebugTimings(enable bool) {
    mux.debugTimings = enable
}

func (mux *Mux) EnableDebug(enable bool) {
    mux.debug = enable
}

func getFunctionName(mh *MethodHandler) string {
    if mh.fnName != "" { return mh.fnName }
    return runtime.FuncForPC(reflect.ValueOf(mh.fn).Pointer()).Name()
}

func (mux *Mux) Print(w io.Writer, indent string) {
    mux.mutex.RLock()
    defer mux.mutex.RUnlock()
    const stdindent = "    "

    keys := make([]string, 0, len(mux.m))
    for k := range mux.m {
        keys = append(keys, k)
    }
    sort.Strings(keys)
    for _, k := range keys {
        v := mux.m[k]
        hasMethod := false
        for method, mh := range v.methodHandlers {
            hasMethod = true
            fmt.Fprintln(w, indent + "/" + k + " (" + method +
                            ")->" + mh.fnName + "()")
        }
        if !hasMethod {
            fmt.Fprintln(w, indent + "/" + k)
        }
        v.Print(w, indent + stdindent)
    }
    for _, v := range mux.matchers {
        hasMethod := false
        for method, mh := range v.Mux.methodHandlers {
            hasMethod = true
            fmt.Fprintln(w, indent + "/" + v.Prefix + v.Label+ " (" +
                            method +  ")->" + mh.fnName + "()")
        }
        if !hasMethod {
            fmt.Fprintln(w, indent + "/" + v.Prefix + v.Label)
        }

        v.Mux.Print(w, indent + stdindent)
    }
}
