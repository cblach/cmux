// Copyright 2024 Christian Thorseth Blach. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package cmux
import(
    "log"
    "reflect"
    "strconv"
    "strings"
    "unsafe"
)

type PathParser interface {
    ParsePath()
}

type pathFieldParser struct {
    Fn              func(string) (unsafe.Pointer, error)
    Type            reflect.Type
    Offset          uintptr
    Size            uintptr
}

type mdPatch struct {
    Source  unsafe.Pointer
    Offset  uintptr /* offset in metatdata struct */
    Size    uintptr
}

func parseString(str string) (unsafe.Pointer, error) {
    return unsafe.Pointer(&str), nil
}

func getParseInt(bitSize int) func (string) (unsafe.Pointer, error) {
    return func (str string) (unsafe.Pointer, error) {
        i, err := strconv.ParseInt(str, 10, bitSize)
        if err != nil {
            return nil, err
        }
        return unsafe.Pointer(&i), nil
    }
}

func getParseUint(bitSize int) func (string) (unsafe.Pointer, error) {
    return func (str string) (unsafe.Pointer, error) {
        i, err := strconv.ParseUint(str, 10, bitSize)
        if err != nil {
            return nil, err
        }
        return unsafe.Pointer(&i), nil
    }
}

var mdTypeMap = map[reflect.Type]map[string]pathFieldParser{}

func parseStruct(md any) map[string]pathFieldParser {
    mdType := reflect.TypeOf(md)
    if p, ok := mdTypeMap[mdType]; ok {
        return p
    }
    if mdType.Kind() != reflect.Pointer {
        panic(mdType.Name() + " is not a pointer")
    }
    mdType = mdType.Elem()
    if mdType.Kind() != reflect.Struct {
        panic(mdType.Name() + " is not a struct pointer")
    }
    p := map[string]pathFieldParser{}
    for _, f := range reflect.VisibleFields(mdType) {
        tag := f.Tag.Get("cmux")
        if tag == "-" {
            continue
        } else if tag == "" {
            if tag = strings.ToLower(f.Name); tag == "" {
                continue
            }
        }
        var fn func(string)(unsafe.Pointer, error)
        switch f.Type.Kind() {
        case reflect.String:
            fn = parseString
        case reflect.Uint:
            fn = getParseUint(0)
        case reflect.Uint64:
            fn = getParseUint(64)
        case reflect.Uint32:
            fn = getParseUint(32)
        case reflect.Uint16:
            fn = getParseUint(16)
        case reflect.Uint8:
            fn = getParseUint(8)
        case reflect.Int:
            fn = getParseInt(0)
        case reflect.Int64:
            fn = getParseInt(64)
        case reflect.Int32:
            fn = getParseInt(32)
        case reflect.Int16:
            fn = getParseInt(16)
        case reflect.Int8:
            fn = getParseInt(8)
        default:
            log.Fatalln("unsupported kind: " + f.Type.Kind().String())
        }
        if p[tag].Fn != nil  {
            log.Fatalln("multiple struct fields matching path variable \"" + tag + "\" in struct " + mdType.String())
        }
        p[tag] = pathFieldParser{
            Fn:     fn,
            Offset: f.Offset,
            Size:   f.Type.Size(),
        }
    }
    return p
}
