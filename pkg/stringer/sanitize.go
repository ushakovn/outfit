package stringer

import (
  "fmt"
  "html"
  "regexp"
  "strconv"
  "strings"

  "github.com/microcosm-cc/bluemonday"
  "golang.org/x/text/cases"
  "golang.org/x/text/language"
)

const (
  SeparatorSpace      = " "
  SeparatorDash       = "-"
  SeparatorUnderscore = "_"
)

var (
  policy         = bluemonday.StrictPolicy()
  RegexSepSet    = regexp.MustCompile(`\s`)
  RegexNonDigit  = regexp.MustCompile(`[^0-9]`)
  RegexNonFloat  = regexp.MustCompile(`[^0-9.,]`)
  RegexRepeatSep = regexp.MustCompile(`\s{2,}`)
)

func StripTags(s string) string {
  return strings.TrimSpace(policy.Sanitize(s))
}

func Strip(s string) string {
  return strings.TrimSpace(s)
}

func IsEmptyStr(s string) bool {
  return Strip(s) == ""
}

func TrimRepeatSeparators(s string, repl string) string {
  return RegexRepeatSep.ReplaceAllString(Strip(s), repl)
}

func ExtractDigit(s string) string {
  s = RegexNonFloat.ReplaceAllString(s, "")
  return strings.ReplaceAll(s, ",", ".")
}

func ToTitle(s string, lang ...language.Tag) string {
  lTag := language.Und
  for _, l := range lang {
    lTag = l
    break
  }
  return cases.Title(lTag, cases.NoLower).String(s)
}

func ContainsStrings(s string, parts ...string) bool {
  for _, part := range parts {
    if !strings.Contains(s, part) {
      return false
    }
  }
  return true
}

func SanitizeString(s string) string {
  s = RegexRepeatSep.ReplaceAllLiteralString(s, " ")
  s = html.UnescapeString(s)
  s = strings.TrimSpace(s)
  return s
}

func NormalizeFloatStr(s string) string {
  const (
    sepComma      = ","
    sepPoint      = "."
    zeroAmountStr = "0"
    replaceFirst  = 1
  )
  var frac string
  s = strings.Replace(s, sepComma, sepPoint, replaceFirst)
  s = RegexNonFloat.ReplaceAllString(s, "")
  parts := strings.Split(s, sepPoint)
  count := len(parts)
  if count == 0 || s == "" {
    return zeroAmountStr
  }
  if count > 1 {
    frac = parts[count-1]
    if frac == "" {
      return zeroAmountStr
    }
    s = strings.Join(parts[:count-1], "")
    s = fmt.Sprint(s, sepPoint, frac)
  }
  return s
}

func NormalizeIntStr(s string) string {
  return RegexNonDigit.ReplaceAllLiteralString(s, "")
}

func ParseIntStr(s string) int {
  s = NormalizeIntStr(s)
  v, _ := strconv.Atoi(s)
  return v
}

func ToString(v any) string {
  return fmt.Sprintf("%v", v)
}

func StripSpaces(s string) string {
  RegexSepSet.ReplaceAllLiteralString(s, "")
  return Strip(s)
}

func ToStringWithOptions(options ...func(s string) string) func(v any) string {
  return func(v any) string {
    s := fmt.Sprintf("%v", v)
    for _, option := range options {
      s = option(s)
    }
    return s
  }
}

func MergeStrings(sep string, elems ...string) string {
  return strings.Join(elems, sep)
}
