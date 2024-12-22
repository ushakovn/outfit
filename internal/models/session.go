package models

import "time"

const (
  StartMenu       SessionMenu = "start_menu"
  StartSilentMenu SessionMenu = "start_silent_menu"

  TrackingMyMenu                SessionMenu = "tracking_my_menu"
  TrackingListMenu              SessionMenu = "tracking_list_menu"
  TrackingSearchInputMenu       SessionMenu = "tracking_search_input_menu"
  TrackingSearchSilentInputMenu SessionMenu = "tracking_search_silent_input_menu"
  TrackingSearchShowMenu        SessionMenu = "tracking_search_show_menu"
  TrackingInsertMenu            SessionMenu = "tracking_insert_menu"
  TrackingInsertConfirmMenu     SessionMenu = "tracking_insert_confirm_menu"
  TrackingInputUrlMenu          SessionMenu = "tracking_input_url_menu"
  TrackingInputSizesMenu        SessionMenu = "tracking_input_sizes_menu"
  TrackingInputFlagMenu         SessionMenu = "tracking_input_flag_menu"
  TrackingCommentMenu           SessionMenu = "tracking_comment_menu"
  TrackingInputCommentMenu      SessionMenu = "tracking_input_comment_menu"
  TrackingFlagConfirmMenu       SessionMenu = "tracking_flag_confirm_menu"
  TrackingDeleteMenu            SessionMenu = "tracking_delete_menu"
  TrackingDeleteConfirmMenu     SessionMenu = "tracking_delete_confirm_menu"

  IssueInsertMenu        SessionMenu = "issue_insert_menu"
  IssueInputTypeMenu     SessionMenu = "issue_input_type_menu"
  IssueInputTextMenu     SessionMenu = "issue_input_text_menu"
  IssueInsertConfirmMenu SessionMenu = "issue_insert_confirm_menu"

  ShopListMenu SessionMenu = "shop_list_menu"
)

type SessionMenu = string

type ChatId = int64

type Session struct {
  ChatId    ChatId           `bson:"chat_id" json:"chat_id"`
  Message   SessionMessage   `bson:"message" json:"message"`
  Tracking  *Tracking        `bson:"tracking" json:"tracking"`
  Entities  *SessionEntities `bson:"entities" json:"entities"`
  UpdatedAt time.Time        `bson:"updated_at" json:"updated_at"`
}

type SessionEntities struct {
  Issue *Issue `bson:"issue" json:"issue"`
}

type SessionMessage struct {
  Id   *int        `bson:"id" json:"id"`
  Menu SessionMenu `bson:"menu" json:"menu"`
}
