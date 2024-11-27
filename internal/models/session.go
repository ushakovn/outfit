package models

import "time"

const (
	Unknown                   SessionMenu = "unknown_menu"
	StartMenu                 SessionMenu = "start_menu"
	StartSilentMenu           SessionMenu = "start_menu_silent"
	TrackingInsertMenu        SessionMenu = "tracking_insert_menu"
	TrackingFieldsMenu        SessionMenu = "tracking_fields_menu"
	TrackingConfirmMenu       SessionMenu = "tracking_confirm_menu"
	TrackingListMenu          SessionMenu = "tracking_list_menu"
	TrackingSelectDeleteMenu  SessionMenu = "tracking_select_delete_menu"
	TrackingConfirmDeleteMenu SessionMenu = "tracking_delete_confirm_menu"
)

type SessionMenu string

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
