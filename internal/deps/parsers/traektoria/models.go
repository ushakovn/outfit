package traektoria

type ParsedPage struct {
  Status string `json:"status"`
  Data   struct {
    HEADER struct {
      SEOBLOCK struct {
        MetaDesc        string `json:"meta_desc"`
        MetaKeywords    string `json:"meta_keywords"`
        MetaTitle       string `json:"meta_title"`
        Redirect        string `json:"redirect"`
        H1              string `json:"h1"`
        Page            string `json:"page"`
        SeoBottomText   string `json:"seo_bottom_text"`
        SeoBrandText    string `json:"seo_brand_text"`
        SeoPopularQuery string `json:"seo_popular_query"`
        SeoPopQueryBot  string `json:"seo_pop_query_bot"`
      } `json:"SEO_BLOCK"`
    } `json:"HEADER"`
    MAIN struct {
      Block   string `json:"block"`
      Sort    int    `json:"sort"`
      Site    string `json:"site"`
      Url     string `json:"url"`
      Content struct {
        Breadcrumb []struct {
          Title string `json:"title"`
          Url   string `json:"url"`
        } `json:"breadcrumb"`
        SelectedSku struct {
          Id                int    `json:"id"`
          XmlId             string `json:"xml_id"`
          ColorTitle        string `json:"color_title"`
          SizeTitle         string `json:"size_title"`
          IsAvailable       bool   `json:"is_available"`
          QuantityText      string `json:"quantity_text"`
          InWishlist        bool   `json:"in_wishlist"`
          Name              string `json:"name"`
          Article           string `json:"article"`
          Quantity          int    `json:"quantity"`
          QuantityInCart    bool   `json:"quantity_in_cart"`
          InCart            bool   `json:"in_cart"`
          BasePrice         int    `json:"base_price"`
          BasePriceFormat   string `json:"base_price_format"`
          RetailPrice       int    `json:"retail_price"`
          RetailPriceFormat string `json:"retail_price_format"`
          IsUseDiscountCard bool   `json:"is_use_discount_card"`
          ShopsAvailable    []struct {
            ID           int    `json:"ID"`
            NAME         string `json:"NAME"`
            CITYNAME     string `json:"CITY_NAME"`
            ADDRESS      string `json:"ADDRESS"`
            COORDINATESX string `json:"COORDINATES_X"`
            COORDINATESY string `json:"COORDINATES_Y"`
            WORKTIME     string `json:"WORK_TIME"`
            PHONE        string `json:"PHONE"`
            METRO        string `json:"METRO"`
            XMLID        string `json:"XML_ID"`
            SORT         int    `json:"SORT"`
          } `json:"shops_available"`
          Discount              any    `json:"discount"`
          StoresAvailable       []any  `json:"stores_available"`
          NearestDispatchDate   string `json:"nearest_dispatch_date"`
          OfflineStoresDataHtml []any  `json:"offline_stores_data_html"`
          PhotoList             []struct {
            Title     string `json:"title"`
            Alt       string `json:"alt"`
            Url       string `json:"url"`
            UrlResize string `json:"url_resize"`
            UrlThumb  string `json:"url_thumb"`
          } `json:"photo_list"`
        } `json:"selected_sku"`
        Model struct {
          SkuList   []ParsedSku `json:"sku_list"`
          PhotoList []struct {
            Title     string `json:"title"`
            Alt       string `json:"alt"`
            Url       string `json:"url"`
            UrlResize string `json:"url_resize"`
            UrlThumb  string `json:"url_thumb"`
          } `json:"photo_list"`
          VideoIframe   string `json:"video_iframe"`
          VideoInternal []any  `json:"video_internal"`
          Badges        []any  `json:"badges"`
          Props         struct {
            ProductId int `json:"product_id"`
            Section   struct {
              Id              int    `json:"id"`
              Name            string `json:"name"`
              Url             string `json:"url"`
              GridSizeBtnText any    `json:"grid_size_btn_text"`
            } `json:"section"`
            IsAvailable            bool   `json:"is_available"`
            IsSoon                 bool   `json:"is_soon"`
            ThingType              string `json:"thing_type"`
            ThingTypeXmlId         string `json:"thing_type_xml_id"`
            Name                   string `json:"name"`
            Article                string `json:"article"`
            Season                 string `json:"season"`
            Gender                 string `json:"gender"`
            ModelName              string `json:"model_name"`
            Country                string `json:"country"`
            Composition            string `json:"composition"`
            IsShowGridSize         bool   `json:"is_show_grid_size"`
            GridSizeText           any    `json:"grid_size_text"`
            IsPeak                 bool   `json:"is_peak"`
            IsShowInexpensiveBlock bool   `json:"is_show_inexpensive_block"`
            BannerHtml             string `json:"banner_html"`
            IsPreorder             bool   `json:"is_preorder"`
            IsGoodRequest          bool   `json:"is_good_request"`
            IsOfflineSale          bool   `json:"is_offline_sale"`
            IsBanSale              bool   `json:"is_ban_sale"`
            IsWear                 bool   `json:"is_wear"`
          } `json:"props"`
          BasePriceMin   string `json:"base_price_min"`
          BasePriceMax   string `json:"base_price_max"`
          RetailPriceMin string `json:"retail_price_min"`
          RetailPriceMax string `json:"retail_price_max"`
          MetaItemprop   struct {
            Description string `json:"description"`
          } `json:"meta_itemprop"`
          Brand struct {
            Name        string `json:"name"`
            Url         string `json:"url"`
            Image       string `json:"image"`
            Description string `json:"description"`
          } `json:"brand"`
          BadgeSeason string `json:"badge_season"`
        } `json:"model"`
        Descriptions struct {
          About    string `json:"about"`
          Features string `json:"features"`
          AllLinks struct {
            Section struct {
              Title string `json:"title"`
              Url   string `json:"url"`
            } `json:"section"`
            Brand struct {
              Title string `json:"title"`
              Url   string `json:"url"`
            } `json:"brand"`
          } `json:"all_links"`
        } `json:"descriptions"`
        FilterOptions []struct {
          Id    int    `json:"id"`
          Name  string `json:"name"`
          Code  string `json:"code"`
          Value string `json:"value"`
        } `json:"filter_options"`
        GridSizeHtml string `json:"grid_size_html"`
        Reviews      struct {
          Count struct {
            Num    int    `json:"num"`
            Ending string `json:"ending"`
          } `json:"count"`
          Average     int   `json:"average"`
          AverageText int   `json:"average_text"`
          List        []any `json:"list"`
        } `json:"reviews"`
        BlogPostList []any `json:"blog_post_list"`
      } `json:"content"`
      Title    string `json:"title"`
      CodePath string `json:"code_path"`
    } `json:"MAIN"`
  } `json:"data"`
}

type ParsedSku struct {
  Name      any          `json:"name"`
  Sizes     []ParsedSize `json:"sizes"`
  PhotoList []struct {
    Title     string `json:"title"`
    Alt       string `json:"alt"`
    Url       string `json:"url"`
    UrlResize string `json:"url_resize"`
    UrlThumb  string `json:"url_thumb"`
  } `json:"photo_list"`
}

type ParsedSize struct {
  Id                int64  `json:"id"`
  XmlId             string `json:"xml_id"`
  ColorTitle        string `json:"color_title"`
  SizeTitle         string `json:"size_title"`
  IsAvailable       bool   `json:"is_available"`
  QuantityText      string `json:"quantity_text"`
  InWishlist        bool   `json:"in_wishlist"`
  Name              string `json:"name"`
  Article           string `json:"article"`
  Quantity          int64  `json:"quantity"`
  QuantityInCart    int64  `json:"quantity_in_cart"`
  InCart            bool   `json:"in_cart"`
  BasePrice         int64  `json:"base_price"`
  BasePriceFormat   string `json:"base_price_format"`
  RetailPrice       int64  `json:"retail_price"`
  RetailPriceFormat string `json:"retail_price_format"`
  IsUseDiscountCard bool   `json:"is_use_discount_card"`
  ShopsAvailable    []struct {
    ID           int64  `json:"ID"`
    NAME         string `json:"NAME"`
    CITYNAME     string `json:"CITY_NAME"`
    ADDRESS      string `json:"ADDRESS"`
    COORDINATESX string `json:"COORDINATES_X"`
    COORDINATESY string `json:"COORDINATES_Y"`
    WORKTIME     string `json:"WORK_TIME"`
    PHONE        string `json:"PHONE"`
    METRO        string `json:"METRO"`
    XMLID        string `json:"XML_ID"`
    SORT         int    `json:"SORT"`
  } `json:"shops_available"`
  Discount *struct {
    Percent           int    `json:"percent"`
    PercentFormat     string `json:"percent_format"`
    NewPriceFormat    string `json:"new_price_format"`
    OldPriceFormat    string `json:"old_price_format"`
    ProfitPrice       int    `json:"profit_price"`
    ProfitPriceFormat string `json:"profit_price_format"`
  } `json:"discount"`
  StoresAvailable       []any  `json:"stores_available"`
  NearestDispatchDate   string `json:"nearest_dispatch_date"`
  OfflineStoresDataHtml []any  `json:"offline_stores_data_html"`
}

type ParsedProduct struct {
  Page ParsedPage
  Sku  ParsedSku
}
