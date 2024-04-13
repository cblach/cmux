// Copyright 2024 Christian Thorseth Blach. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package cmux
import(
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log"
    "net/http"
    "net/http/httputil"
    "os"
    "reflect"
    "strings"
    "sync"
    "time"
    "unsafe"
)

var DefaultMux = &Mux{}

type Mux struct {
    Before          func(http.ResponseWriter, *http.Request, any, any) error

    parent          *Mux
    methodHandlers  map[string]*MethodHandler

    metadata        any
    metadataRaw     []byte
    metadataType     reflect.Type

    servesDir       bool /* Does the handlefunc serve a dir? (i.e. ends with '/') */
    debugTimings    bool
    debug           bool
    dfltContentType string

    /* Directly mapped muxes */
    m            map[string]*Mux

    /* Linearly mapped muxes */
    matchers    []fmtMatcher

    sync.RWMutex
}

var methodHandlerType = reflect.TypeOf(MethodHandler{})

/* Fmt stuff */

type fmtMatcher struct {
    Mux      *Mux
    Prefix   string
    Suffix   string
    FieldParser pathFieldParser

    /* for parsing only */
    Label    string
    Type     reflect.Type
    Size     uintptr
}

/* Actual routing */

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Body == nil {
        r.Body = io.NopCloser(bytes.NewReader([]byte{}))
    }
    if mux.debug {
        rawReq, err := httputil.DumpRequest(r, true)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Failed to dump request: %s", err.Error())
        } else {
            fmt.Fprintf(os.Stderr, "Request = {\n%s\n}\n", string(rawReq))
        }
    }
    if r.URL.Path[0] != '/' {
        http.NotFound(w, r)
        return
    }
    dirs := strings.Split(r.URL.Path, "/")[1:]
    mux.RLock()
    match, fallback, patches := mux.matchDir(dirs)
    mux.RUnlock()
    if match == nil {
        match = fallback
        if match == nil {
            http.NotFound(w, r)
            return
        }
    }
    var mh *MethodHandler
    if mh = match.methodHandlers[r.Method]; mh == nil {
        http.Error(w, "", http.StatusMethodNotAllowed)
        return
    }
    if mux.dfltContentType != "" {
        w.Header().Set("Content-Type", mux.dfltContentType)
    }
    var mdIf any = nil
    mdRaw := make([]byte, len(match.metadataRaw))
    if match.metadata != nil {
        copy(mdRaw, match.metadataRaw)
        mdPtr := unsafe.Pointer(unsafe.SliceData(mdRaw))
        for _, patch := range patches {
            dst := unsafe.Slice((*byte)(unsafe.Add(mdPtr, patch.Offset)), patch.Size)
            src := unsafe.Slice((*byte)(patch.Source), patch.Size)
            copy(dst, src)
        }
        mdIf = reflect.NewAt(match.metadataType.Elem(), mdPtr).Interface()
    }
    if mux.Before != nil {
        if err := mux.Before(w, r, mdIf, mh.Data); err != nil {
            mux.handleErr(w, r, err)
            return
        }
    }
    var t0, t1 time.Time
    if mux.debugTimings { t0 = time.Now() }
    if err := mh.Func(w, r, mdIf); err != nil {
        mux.handleErr(w, r, err)
    }
    if mux.debugTimings {
        t1 = time.Now()
        log.Println(t1.Sub(t0), r.URL.Path)
    }
}

/* Note that fnName exists for debugging purposes */
func (mux *Mux) mkRoute(path string, metadata any, methodHandlers map[string]*MethodHandler) {
    mux.Lock()
    if mux.m == nil { mux.m = map[string]*Mux{} }
    defer mux.Unlock()
    if path[0] != '/' { log.Fatalln("path must start with slash", path) }
    dirs := strings.Split(path, "/")[1:]

    servesDir := false
    if dirs[len(dirs) - 1] == "" {
        dirs = dirs[:len(dirs) - 1]
        servesDir = true
    }
    for _, dir := range dirs {
        preBracket, postBracket, found := strings.Cut(dir, "{")
        if strings.Contains(preBracket, "}") {
            log.Fatalln("unexpected end bracket not closing expresison")
        }
        if found {
            /* found variable bracket: */
            pathVar, rem, found := strings.Cut(postBracket, "}")
            if !found {
                log.Fatalln("missing end bracket")
            }
            if strings.Contains(pathVar, "{") {
                log.Fatalln("nested brackets not allowed in expressions")
            }
            if metadata == nil {
                log.Fatalln("metadata cannot be nil when using labels")
            }
            parserMap := parseStruct(metadata)
            p, ok := parserMap[pathVar]
            if !ok {
                log.Fatalf("struct for %s does not contain field %s",
                           path, pathVar)
            }
            matcher := fmtMatcher{
                Mux: &Mux {
                    parent: mux,
                    m: map[string]*Mux{},
                },
                Prefix: preBracket,
                Suffix: rem,
                FieldParser: p,
                Label: pathVar,
                Size:  p.Size,
            }
            var mIdx int
            var m fmtMatcher
            for mIdx, m = range mux.matchers{
                if m.Prefix == matcher.Prefix &&
                   m.Suffix == matcher.Suffix &&
                   m.FieldParser.Type == matcher.FieldParser.Type &&
                   m.Label == matcher.Label &&
                   m.Size == matcher.Size {
                    break
                }
            }
            if mIdx < len(mux.matchers) {
                mux = mux.matchers[mIdx].Mux
            } else {
                mux.matchers = append(mux.matchers, matcher)
                mux = matcher.Mux
            }
        } else {
            /* did not find variable bracket */
            if dir == "" { log.Fatalln("empty dir name not permittede", path) }
            nmux, ok := mux.m[dir]
            if !ok {
                mux.m[dir] = &Mux{
                    parent: mux,
                    m: map[string]*Mux{},
                }
                mux = mux.m[dir]
            } else { mux = nmux }
        }
    }
    mux.servesDir = servesDir
    if mux.metadata = metadata; mux.metadata != nil {
        mux.metadataType = reflect.TypeOf(mux.metadata)
        rv := reflect.ValueOf(mux.metadata)
        mux.metadataRaw = unsafe.Slice((*byte)(rv.UnsafePointer()), mux.metadataType.Elem().Size())
    }
    mux.methodHandlers = methodHandlers
}

type HTTPResponder interface {
    HTTPRespond() (any, error)
}

type HTTPErrorResponder interface {
    HTTPError()(int, any)
}

func (mux *Mux) handleErr(w http.ResponseWriter, r *http.Request, err error) {
    var her HTTPErrorResponder
    var hr HTTPResponder
    code := 200
    var out any
    if errors.As(err, &her) {
        code, out = her.HTTPError()
    } else if errors.As(err, &hr) {
        out, err = hr.HTTPRespond()
        if err != nil {
            if errors.As(err, &her) {
                code, out = her.HTTPError()
            } else {
                code = http.StatusInternalServerError
                out = &struct{Error string `json:"error"`}{"internal server error"}
            }
            log.Printf("Encountered unexpected error at %s: %s", r.URL, err.Error())
        }
    } else {
        code = http.StatusInternalServerError
        out = &struct{Error string `json:"error"`}{"internal server error"}
        log.Printf("Encountered unexpected error at %s: %s", r.URL, err.Error())
    }

    w.WriteHeader(code)
    json.NewEncoder(w).Encode(out)
    if mux.debug {
        res := http.Response {
            StatusCode: code,
            Proto:      "HTTP/1.1",
            Header:     w.Header(),
        }
        rawRes, err := httputil.DumpResponse(&res, false)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Failed to dump request: %s", err.Error())
        } else {
            fmt.Fprintf(os.Stderr, "Response = {\n%s", string(rawRes))
        }
        json.NewEncoder(os.Stderr).Encode(out)
        fmt.Fprintf(os.Stderr, "\n}\n")
    }
}

type codeResponder struct{
    code int
    error
}

func (r *codeResponder) HTTPError() (int, any) {
    var str string
    if str = r.error.Error(); str == "" {
        str = "unknown error"
    }
    return r.code, struct{Error string `json:"error"`}{str}
}

func (r *codeResponder) Unwrap() error {
    return r.error
}

func WrapError(err error, code int) error {
    if err.Error() == "" {
        err = errors.New(http.StatusText(code))
    }
    return &codeResponder{
        code: code,
        error: err,
    }
}

func HTTPError(err string, code int) error {
    if err == "" {
        err = http.StatusText(code)
    }
    return &codeResponder{
        code: code,
        error: errors.New(err),
    }
}

type whitelistedData struct{
    data any
}

func Whitelist(res any) whitelistedData {
    return whitelistedData{res}
}

func (wd whitelistedData) HTTPRespond()(any, error) {
    return wd.data, nil
}

func (wd whitelistedData) Error() string {
    return "whitelisted data not working"
}

/*
 * Attempt to match a dir
 * If no exact match is found, it will attempt to fallback to a served dir,
 * i.e. a folder served with / at the end, e.g. 'folder/'
 */

func (mux *Mux) matchDir(dirs []string) (*Mux, *Mux, []mdPatch) {
    if len(dirs) == 0 {
        return mux, nil, []mdPatch{}
    }

    dir := dirs[0]
    dirs = dirs[1:]
    var fallback *Mux
    var fbPatches []mdPatch
    /* Check for exact string matches */
    nmux, ok := mux.m[dir]
    if ok {
        if match, fb, patches := nmux.matchDir(dirs); match != nil {
            return match, nil, patches
        } else {
            fallback = fb
            fbPatches = patches
        }
    }
    /* Loop through the parsers, and see if they match */
    for _, matcher := range mux.matchers {
        if !strings.HasPrefix(dir, matcher.Prefix) ||
           !strings.HasSuffix(dir[len(matcher.Prefix):], matcher.Suffix) {
            continue
        }
        src, err := matcher.FieldParser.Fn(dir[len(matcher.Prefix):len(dir) - len(matcher.Suffix)])
        if err != nil { continue }
        patch := mdPatch{
            Offset: matcher.FieldParser.Offset,
            Source: src,
            Size:   matcher.FieldParser.Size,
        }
        if match, fb, patches := matcher.Mux.matchDir(dirs); match != nil {
            /* Prepend to argList */
            patches = append([]mdPatch{patch}, patches...)
            return match, nil, patches
        } else if fallback == nil {
            fallback = fb
            fbPatches = append([]mdPatch{patch}, patches...)
        }
    }

    if fallback == nil && mux.servesDir {
        return nil, mux, []mdPatch{}
    }
    return nil, fallback, fbPatches
}
