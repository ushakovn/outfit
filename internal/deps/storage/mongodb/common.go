package mongodb

import (
  "reflect"

  "go.mongodb.org/mongo-driver/bson"
)

func makeBsonDUpdates(document any) bson.D {
  updates := bson.D{}

  typ := reflect.TypeOf(document)
  value := reflect.ValueOf(document)

  if typ.Kind() == reflect.Ptr {
    typ = typ.Elem()
    value = value.Elem()
  }

  for i := 1; i < typ.NumField(); i++ {
    field := typ.Field(i)

    val := value.Field(i)
    tag := field.Tag.Get("bson")

    if !isZeroType(val) {
      update := bson.E{
        Key:   tag,
        Value: val.Interface(),
      }
      updates = append(updates, update)
    }
  }

  return bson.D{{
    Key:   "$set",
    Value: updates,
  }}
}

func makeBsonDFilters(kv map[string]any) bson.D {
  out := bson.D{}

  for key, value := range kv {
    out = append(out, bson.E{
      Key:   key,
      Value: value,
    })
  }

  return out
}

func isZeroType(value reflect.Value) bool {
  zero := reflect.Zero(value.Type()).Interface()

  switch value.Kind() {
  case reflect.Slice, reflect.Array, reflect.Chan, reflect.Map:
    return value.Len() == 0
  default:
    return reflect.DeepEqual(zero, value.Interface())
  }
}
