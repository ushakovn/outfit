package ridestep

type ParsedProduct struct {
  Name  string
  Brand string
  Image string
  Skus  []ParsedProductSku
}

type ParsedProductView struct {
  Name     string `json:"name"`
  Id       int    `json:"id"`
  Category string `json:"category"`
  Price    int    `json:"price"`
  Brand    string `json:"brand"`
}

type ParsedProductTracking struct {
  Name  string
  Image string
}

type ParsedProductSku struct {
  Id                  string           `json:"id"`
  ProductId           string           `json:"product_id"`
  Sku                 string           `json:"sku"`
  Sort                string           `json:"sort"`
  Name                string           `json:"name"`
  ImageId             any              `json:"image_id"`
  Price               string           `json:"price"`
  Count               int64            `json:"count"`
  Available           string           `json:"available"`
  StockBaseRatio      any              `json:"stock_base_ratio"`
  OrderCountMin       any              `json:"order_count_min"`
  OrderCountStep      any              `json:"order_count_step"`
  Status              string           `json:"status"`
  DimensionId         any              `json:"dimension_id"`
  FileName            string           `json:"file_name"`
  FileSize            string           `json:"file_size"`
  FileDescription     any              `json:"file_description"`
  Virtual             string           `json:"virtual"`
  BoMinPrice          any              `json:"bo_min_price"`
  Stock               map[string]int64 `json:"stock"`
  UnconvertedCurrency string           `json:"unconverted_currency"`
  Currency            string           `json:"currency"`
}
