package swagger

import (
    "reflect"
    "strings"
)

const (
    swagTagKey   = "swag"
    swagSep      = ";"
    swagIdentSep = ":"
    swagValueSep = ","
)

type parsedStructTag struct {
    Name           string
    Optional       bool
    PossibleValues []string
}

func getParsedStructTag(tag reflect.StructTag, name string) parsedStructTag {
    pst := parsedStructTag{}
    nameTag := tag.Get(name)
    if nameTag == "" {
        return pst
    }

    // This is a hack to remove omitempty from tags
    fNameParts := strings.Split(nameTag, swagValueSep)
    if len(fNameParts) > 0 {
        pst.Name = strings.TrimSpace(fNameParts[0])
    }

    swagTag := tag.Get(swagTagKey)
    parts := strings.Split(swagTag, swagSep)
    for _, p := range parts {
        x := strings.TrimSpace(strings.ToLower(p))
        switch {
        case x == "optional":
            pst.Optional = true
        case strings.HasPrefix(x, "enum:"):
            xx := strings.SplitN(p, swagIdentSep, 2)
            if len(xx) == 2 {
                xx = strings.Split(xx[1], swagValueSep)
                for _, v := range xx {
                    pst.PossibleValues = append(pst.PossibleValues, strings.TrimSpace(v))
                }
            }
        }
    }

    return pst
}
