package reflection

import "reflect"

func IsZeroType(value reflect.Value) bool {
  zero := reflect.Zero(value.Type()).Interface()

  switch value.Kind() {
  case reflect.Slice, reflect.Array, reflect.Chan, reflect.Map:
    return value.Len() == 0
  default:
    return reflect.DeepEqual(zero, value.Interface())
  }
}
