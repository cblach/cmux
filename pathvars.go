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

func parseUint64(str string) (unsafe.Pointer, error) {
    i, err := strconv.ParseUint(str, 10, 64)
    if err != nil {
        return nil, err
    }
    return unsafe.Pointer(&i), nil
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
        tag := f.Tag.Get("mux")
        if tag == "" {
            if tag = strings.ToLower(f.Name); tag == "" {
                continue
            }
        }
        var fn func(string)(unsafe.Pointer, error)
        switch f.Type.Kind() {
        case reflect.String:
            fn = parseString
        case reflect.Uint64:
            fn = parseUint64
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
