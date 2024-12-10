package models

import "time"

type IssueType string

const (
  IssueTypeBug   IssueType = "bug"
  IssueTypeStory IssueType = "story"
)

type Issue struct {
  ChatId    int64     `bson:"chat_id" json:"chat_id"`
  Type      IssueType `bson:"type" json:"type"`
  Text      string    `bson:"text" json:"text"`
  CreatedAt time.Time `bson:"created_at" json:"created_at"`
}
