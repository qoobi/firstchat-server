package main

import(
  "time"
  _ "github.com/jinzhu/gorm/dialects/postgres"
)

type User struct {
  ID          uint64    `gorm:"primary_key"`
  Email       string
  Name        string
  Surname     string
  Picture     string
}

type Chat struct {
  ID          uint64    `gorm:"primary_key"`
  Admin       User
  AdminID     uint64
  Picture     string
}

type Contact struct {
  Who         User
  WhoID       uint64    `gorm:"primary_key"`

  Whom        User
  WhomID      uint64    `gorm:"primary_key"`
}

type App struct {
  ID          uint64  `gorm:"primary_key"`

  User        User
  UserID      uint64    `gorm:"primary_key"`

  Key         uint64
}

type Participant struct {
  Chat        Chat
  ChatID      uint64    `gorm:"primary_key"`

  User        User
  UserID      uint64    `gorm:"primary_key"`
}

type Message struct {
  ID          uint64    `gorm:"primary_key"`

  Chat        Chat
  ChatID      uint64

  From        User
  FromID      uint64

  timestamp   time.Time
  content     string
}

type Text struct {
  ID          uint64    `gorm:"primary_key"`

  From        User
  FromID      uint64

  To          User
  ToID        uint64

  Contents    string
}

type DHParam struct {
  First       User
  FirstID     uint64    `gorm:"primary_key"`

  Second      User
  SecondID    uint64    `gorm:"primary_key"`

  Param       []byte
}

type DHPubKey struct {
  First       User
  FirstID     uint64    `gorm:"primary_key"`

  Second      User
  SecondID    uint64    `gorm:"primary_key"`

  PubKey      []byte
}