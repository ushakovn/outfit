package models

import "time"

const (
  Unknown             SessionMenu = "unknown_menu"
  StartMenu           SessionMenu = "start_menu"
  StartSilentMenu     SessionMenu = "start_menu_silent"
  TrackingInsertMenu  SessionMenu = "tracking_insert_menu"
  TrackingFieldsMenu  SessionMenu = "tracking_fields_menu"
  TrackingConfirmMenu SessionMenu = "tracking_confirm_menu"
  TrackingListMenu    SessionMenu = "tracking_list_menu"
  TrackingDeleteMenu  SessionMenu = "tracking_delete_menu"
)

type SessionMenu string

type Session struct {
  Telegram Telegram       `bson:"telegram" json:"telegram"`
  Message  SessionMessage `bson:"message" json:"message"`
  Tracking *Tracking      `bson:"tracking" json:"tracking"`
}

type SessionMessage struct {
  Menu      SessionMenu `bson:"menu" json:"menu"`
  CreatedAt time.Time   `bson:"created_at" json:"created_at"`
}
