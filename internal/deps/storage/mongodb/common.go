package mongodb

import (
  "reflect"

  "github.com/ushakovn/outfit/pkg/reflection"
  "go.mongodb.org/mongo-driver/bson"
)

func makeBsonBsonDSort(params []SortParams) bson.D {
  bsonD := bson.D{}

  for _, sort := range params {
    bsonD = append(bsonD, bson.E{
      Key:   sort.Field,
      Value: int(sort.Order),
    })
  }

  return bsonD
}

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

    if !reflection.IsZeroType(val) {
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
