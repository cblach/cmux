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

func (mux *Mux) ToggleDebugTimings(toggle bool) {
    mux.debugTimings = toggle
}

func (mux *Mux) ToggleDebug(toggle bool) {
    mux.debug = true
}

func getFunctionName(mh *MethodHandler) string {
    if mh.FuncName != "" { return mh.FuncName }
    return runtime.FuncForPC(reflect.ValueOf(mh.Func).Pointer()).Name()
}

func (mux *Mux) Print(w io.Writer, indentA ...string) {
    mux.RLock()
    defer mux.RUnlock()
    const stdindent = "    "
    indent := ""
    if len(indentA) != 0 { indent = indentA[0] }

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
            fmt.Fprintln(w, indent + "/" + k + " (" + method +  ")->" + mh.FuncName + "()")
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
            fmt.Fprintln(w, indent + "/" + v.Prefix + v.Label+ " (" + method +  ")->" + mh.FuncName + "()")
        }
        if !hasMethod {
            fmt.Fprintln(w, indent + "/" + v.Prefix + v.Label)
        }

        v.Mux.Print(w, indent + stdindent)
    }
}
