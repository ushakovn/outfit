package models

import "time"

const (
  StartMenu                 SessionMenu = "start_menu"
  StartSilentMenu           SessionMenu = "start_silent_menu"
  TrackingListMenu          SessionMenu = "tracking_list_menu"
  TrackingInsertMenu        SessionMenu = "tracking_insert_menu"
  TrackingInsertConfirmMenu SessionMenu = "tracking_insert_confirm_menu"
  TrackingInputUrlMenu      SessionMenu = "tracking_input_url_menu"
  TrackingInputSizesMenu    SessionMenu = "tracking_input_sizes_menu"
  TrackingInputDiscountMenu SessionMenu = "tracking_input_discount_menu"
  TrackingDeleteMenu        SessionMenu = "tracking_delete_menu"
  TrackingDeleteConfirmMenu SessionMenu = "tracking_delete_confirm_menu"
  // Deprecated: do not use.
  TrackingFieldsMenu SessionMenu = "tracking_fields_menu"
)

type SessionMenu = string

type Session struct {
  ChatId   int64          `bson:"chat_id" json:"chat_id"`
  Message  SessionMessage `bson:"message" json:"message"`
  Tracking *Tracking      `bson:"tracking" json:"tracking"`
}

type SessionMessage struct {
  Id        *int64      `bson:"id" json:"id"`
  Menu      SessionMenu `bson:"menu" json:"menu"`
  UpdatedAt time.Time   `bson:"updated_at" json:"updated_at"`
}
