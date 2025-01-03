package money

import "github.com/leekchan/accounting"

var acc = accounting.Accounting{
  Symbol:    "₽ ",
  Precision: 2,
  Thousand:  " ",
  Decimal:   ".",
}

func String(value int64) string {
  return acc.FormatMoney(value)
}
