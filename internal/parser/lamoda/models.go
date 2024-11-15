package lamoda

import "time"

type ParsedProduct struct {
  Product struct {
    Brand struct {
      ThumbnailImage string `json:"thumbnail_image"`
      ID             string `json:"id"`
      IsBeauty       bool   `json:"is_beauty"`
      IsKids         bool   `json:"is_kids"`
      IsPremium      bool   `json:"is_premium"`
      IsSport        bool   `json:"is_sport"`
      SeoTail        string `json:"seo_tail"`
      Title          string `json:"title"`
    } `json:"brand"`
    Breadcrumbs []struct {
      GenderSegment string `json:"gender_segment"`
      ID            int    `json:"id"`
      ImagePath     string `json:"image_path"`
      Label         string `json:"label"`
      Name          string `json:"name"`
      URL           string `json:"url"`
      SizeTableID   int    `json:"size_table_id,omitempty"`
      ImageStub     string `json:"image_stub,omitempty"`
    } `json:"breadcrumbs"`
    CatalogLinks []struct {
      Deeplink       string `json:"deeplink"`
      Description    string `json:"description"`
      Link           string `json:"link"`
      ThumbnailImage string `json:"thumbnail_image"`
      Title          string `json:"title"`
      Type           string `json:"type"`
    } `json:"catalog_links"`
    Installments []struct {
      Description   string `json:"description"`
      PageInfoURL   string `json:"page_info_url"`
      PaymentMethod string `json:"payment_method"`
      Price         string `json:"price"`
      ShowBanner    bool   `json:"show_banner"`
      Title         string `json:"title"`
      PriceRaw      int    `json:"price_raw,omitempty"`
    } `json:"installments"`
    IsHybrid      bool `json:"is_hybrid"`
    IsReturnable  bool `json:"is_returnable"`
    OtherCategory struct {
      ThumbnailImage string `json:"thumbnail_image"`
    } `json:"other_category"`
    Seasons []struct {
      ID    string `json:"id"`
      Title string `json:"title"`
    } `json:"seasons"`
    SeoTail       string `json:"seo_tail"`
    SizeID        string `json:"size_id"`
    SizeScaleCode string `json:"size_scale_code"`
    Sizes         []struct {
      ShortSku           string      `json:"short_sku"`
      BrandID            string      `json:"brand_id"`
      BrandSizeSystem    string      `json:"brand_size_system"`
      BrandTitle         string      `json:"brand_title"`
      ID                 string      `json:"id"`
      IsUniversal        bool        `json:"is_universal"`
      PrimaryDescription string      `json:"primary_description"`
      ShipmentType       interface{} `json:"shipment_type"`
      SizeSystem         string      `json:"size_system"`
      Sku                string      `json:"sku"`
      StockQuantity      int64       `json:"stock_quantity"`
      TaxVat             int         `json:"tax_vat"`
      Title              string      `json:"title"`
    } `json:"sizes"`
    SkuSupplier string        `json:"sku_supplier"`
    SubsetID    int           `json:"subset_id"`
    Actions     []interface{} `json:"actions"`
    Attributes  []struct {
      Key   string `json:"key"`
      Title string `json:"title"`
      Value string `json:"value"`
    } `json:"attributes"`
    Badges       []interface{} `json:"badges"`
    BestCategory struct {
      GenderSegment string `json:"gender_segment"`
      ID            string `json:"id"`
      ImageURL      string `json:"image_url"`
      SeoTail       string `json:"seo_tail"`
      SizeTableID   string `json:"size_table_id"`
      Title         string `json:"title"`
    } `json:"best_category"`
    BestPriceInfo struct {
      Loyalty struct {
        Discount             int  `json:"discount"`
        DiscountAmount       int  `json:"discount_amount"`
        IsDiscountRestricted bool `json:"is_discount_restricted"`
        Price                int  `json:"price"`
      } `json:"loyalty"`
      Type string `json:"type"`
    } `json:"best_price_info"`
    CategoryLeaves  []int  `json:"category_leaves"`
    CategoryType    string `json:"category_type"`
    Collection      string `json:"collection"`
    ColoredProducts []struct {
      Colors []struct {
        ID    string `json:"id"`
        Title string `json:"title"`
      } `json:"colors"`
      Gallery    []string `json:"gallery"`
      IsInStock  bool     `json:"is_in_stock"`
      IsSellable bool     `json:"is_sellable"`
      Name       string   `json:"name"`
      Price      int      `json:"price"`
      Sizes      []struct {
        BrandID       string      `json:"brand_id"`
        BrandTitle    string      `json:"brand_title"`
        ID            string      `json:"id"`
        IsUniversal   bool        `json:"is_universal"`
        ShipmentType  interface{} `json:"shipment_type"`
        SizeSystem    string      `json:"size_system"`
        Sku           string      `json:"sku"`
        StockQuantity int         `json:"stock_quantity"`
        TaxVat        int         `json:"tax_vat"`
        Title         string      `json:"title"`
      } `json:"sizes"`
      Sku       string `json:"sku"`
      Thumbnail string `json:"thumbnail"`
      Type      string `json:"type"`
    } `json:"colored_products"`
    Colors []struct {
      ID    string `json:"id"`
      Title string `json:"title"`
    } `json:"colors"`
    Counters struct {
      BuyPerDay    int `json:"buy_per_day"`
      BuyPerWeek   int `json:"buy_per_week"`
      Photoreviews int `json:"photoreviews"`
      Questions    int `json:"questions"`
      Reviews      int `json:"reviews"`
      TotalPhotos  int `json:"total_photos"`
      ViewPerDay   int `json:"view_per_day"`
      ViewPerWeek  int `json:"view_per_week"`
    } `json:"counters"`
    CustomBadges interface{} `json:"custom_badges"`
    CustomTags   interface{} `json:"custom_tags"`
    Delivery     struct {
      BestDeliveryInfo struct {
        BestDeliveryType string `json:"best_delivery_type"`
        BestTerms        struct {
          DateMax time.Time `json:"date_max"`
          DateMin time.Time `json:"date_min"`
        } `json:"best_terms"`
        DeliveryData []struct {
          Courier struct {
            DateMax time.Time `json:"date_max"`
            DateMin time.Time `json:"date_min"`
            Info    struct {
              Title string `json:"title"`
            } `json:"info"`
            IsSpecialConditions bool   `json:"is_special_conditions"`
            IsTryonAble         bool   `json:"is_tryon_able"`
            IsTryonAllowed      bool   `json:"is_tryon_allowed"`
            PriceType           string `json:"price_type"`
          } `json:"courier,omitempty"`
          Type   string `json:"type"`
          Pickup struct {
            DateMax time.Time `json:"date_max"`
            DateMin time.Time `json:"date_min"`
            Info    struct {
              Title string `json:"title"`
            } `json:"info"`
            IsSpecialConditions bool   `json:"is_special_conditions"`
            IsTryonAble         bool   `json:"is_tryon_able"`
            IsTryonAllowed      bool   `json:"is_tryon_allowed"`
            Price               int    `json:"price"`
            PriceType           string `json:"price_type"`
          } `json:"pickup,omitempty"`
          Post struct {
            DateMax time.Time `json:"date_max"`
            DateMin time.Time `json:"date_min"`
            Info    struct {
              Title string `json:"title"`
            } `json:"info"`
            IsSpecialConditions bool   `json:"is_special_conditions"`
            Price               int    `json:"price"`
            PriceType           string `json:"price_type"`
          } `json:"post,omitempty"`
        } `json:"delivery_data"`
        Description string `json:"description"`
        ShowTryon   struct {
          Content struct {
            Blocks []struct {
              Image string `json:"image"`
              Text  string `json:"text"`
              Title string `json:"title"`
            } `json:"blocks"`
            Title string `json:"title"`
          } `json:"content"`
          LabelAllowed bool   `json:"label_allowed"`
          LabelTitle   string `json:"label_title"`
          LabelType    string `json:"label_type"`
        } `json:"show_tryon"`
      } `json:"best_delivery_info"`
      Type string `json:"type"`
    } `json:"delivery"`
    DiscountLamodaAndLoyaltyAndAction int `json:"discount_lamoda_and_loyalty_and_action"`
    Features                          struct {
      ShowColor  bool `json:"show_color"`
      ShowRating bool `json:"show_rating"`
      ShowSize   bool `json:"show_size"`
      ShowVolume bool `json:"show_volume"`
    } `json:"features"`
    FeedbackID int `json:"feedback_id"`
    Fittings   struct {
    } `json:"fittings"`
    Gallery             []string      `json:"gallery"`
    Gender              string        `json:"gender"`
    HasLooksForSku      bool          `json:"has_looks_for_sku"`
    ImageAttributes     interface{}   `json:"image_attributes"`
    InstallmentInfo     string        `json:"installment_info"`
    IsBuyTheLookAllowed bool          `json:"is_buy_the_look_allowed"`
    IsInStock           bool          `json:"is_in_stock"`
    IsLoyaltyApplicable bool          `json:"is_loyalty_applicable"`
    IsSellable          bool          `json:"is_sellable"`
    IsTryon             bool          `json:"is_tryon"`
    LongAttributes      []interface{} `json:"long_attributes"`
    ModelTitle          string        `json:"model_title"`
    Originality         []struct {
      Icon     string `json:"icon"`
      Subtitle string `json:"subtitle"`
      Title    string `json:"title"`
    } `json:"originality"`
    Photoreviews interface{} `json:"photoreviews"`
    Price        int64       `json:"price"`
    Prices       struct {
      LoyaltyBase struct {
        Discount struct {
          Amount             string `json:"amount"`
          AmountAccumulated  string `json:"amount_accumulated"`
          Percent            int    `json:"percent"`
          PercentAccumulated int    `json:"percent_accumulated"`
        } `json:"discount"`
        Price string `json:"price"`
      } `json:"loyalty_base"`
      Original struct {
        Price string `json:"price"`
      } `json:"original"`
    } `json:"prices"`
    ReplacementProducts interface{} `json:"replacement_products"`
    Reviews             struct {
      Items         []interface{} `json:"items"`
      Sort          string        `json:"sort"`
      SortDirection string        `json:"sort_direction"`
    } `json:"reviews"`
    Seller struct {
      BusinessModel   string `json:"business_model"`
      DeliveryProfile string `json:"delivery_profile"`
      ID              string `json:"id"`
      IsLamoda        bool   `json:"is_lamoda"`
      IsTryon         bool   `json:"is_tryon"`
      Title           string `json:"title"`
    } `json:"seller"`
    SeoTitle            string `json:"seo_title"`
    SizeRecommendations struct {
      Footty struct {
        CanRecommend    bool `json:"can_recommend"`
        HasMeasurements bool `json:"has_measurements"`
      } `json:"footty"`
    } `json:"size_recommendations"`
    SizeTableURL string `json:"size_table_url"`
    Sku          string `json:"sku"`
    Thumbnail    string `json:"thumbnail"`
    Title        string `json:"title"`
    Type         string `json:"type"`
  } `json:"product"`
  Breadcrumbs []struct {
    ID      int    `json:"id"`
    URL     string `json:"url"`
    Name    string `json:"name"`
    SeoTail string `json:"seo_tail"`
  } `json:"breadcrumbs"`
}

func (p *ParsedProduct) IsDiscountApplicable() bool {
  return p.Product.IsLoyaltyApplicable
}
