package main

import (
  "fmt"
  //"time"
  "github.com/jinzhu/gorm"
  //"crypto/rand"
  //"crypto/tls"
  //"net"
  "github.com/sendgrid/sendgrid-go"
  "sync"
  "bytes"
  "encoding/binary"
  "golang.org/x/net/websocket"
  "net/http"
)

type Connections struct {
  s sync.Mutex
  m map[uint64]*websocket.Conn
}

func main() {
  db, err := gorm.Open("postgres", "host=localhost user=serv dbname=firstchat sslmode=disable password=Secret")
  if err != nil {
    panic("failed to connect database")
  }
  db.AutoMigrate(&User{}, &Chat{}, &Contact{}, &App{}, &Participant{}, &Message{}, &Text{}, &DHParam{}, &DHPubKey{})
  
  conns := Connections{m: make(map[uint64]*websocket.Conn)}

  var count int
  fmt.Println(count)
  db.Model(&User{}).Where("email = ?", "1").Count(&count)
  fmt.Println(count)
        
  http.Handle("/", wsHandler(&conns, db))
  err = http.ListenAndServeTLS(
    ":12341",
    "firstchat.mmkg.me.cert.pem",
    "firstchat.mmkg.me.key.pem", 
    nil)
  if err != nil {
    fmt.Println(err)
    panic("failed to listen")
  }
}

const (
  OK uint8 = 1 + iota
  NOT_OK
  APP_ID
  APP_ID_KEY
  EMAIL
  CHECK_CODE
  KEY
  GET_CONTACTS
  CONTACT
  ADD_FRIEND
  MESSAGE
  DH_PARAM
  DH_PUB_KEY
)

func wsHandler(conns *Connections, db *gorm.DB) websocket.Handler {
  return func(ws *websocket.Conn) {
    handleClient(ws, conns, db)
  }
}

func handleClient(ws *websocket.Conn, conns *Connections, db *gorm.DB) {
  fmt.Println("Accepted")
  var userId uint64
  var appId uint64
  var code string = "123456"
  var email string
  var err error
  for {
    var msg []byte
    err = websocket.Message.Receive(ws, &msg)
    if err != nil {
      fmt.Println("Can't receive")
      break
    }
    buf := bytes.NewBuffer(msg)
    var msgType uint8
    binary.Read(buf, binary.LittleEndian, &msgType)
    fmt.Println(msgType)
    switch msgType {
    case APP_ID_KEY:
      var appKey uint64
      binary.Read(buf, binary.LittleEndian, &appId)
      binary.Read(buf, binary.LittleEndian, &appKey)
      fmt.Println(appId)
      fmt.Println(appKey)
      var count int
      db.Table("apps").Where("id = ? and key = ?", appId, appKey).Count(&count)
      if count == 1 {
        db.Table("apps").Select("user_id").Where("id = ? and key = ?", appId, appKey).Find(&userId)
        sendMessage(ws, OK, nil)
      } else {
        sendMessage(ws, NOT_OK, nil)
      }
    case APP_ID:
      binary.Read(buf, binary.LittleEndian, &appId)
      fmt.Println("appId")
      fmt.Println(appId)
    case EMAIL:
      email, _ = buf.ReadString(0)
      //sendEmailViaSendgrid(email, code)
      fmt.Println("email sent to ", email)
      sendMessage(ws, OK, nil)
    case CHECK_CODE:
      codeToCheck, _ := buf.ReadString(0)
      if codeToCheck == code {
        //fmt.Println(email)
        user := User{Email: email}
        //fmt.Println(user)
        var count int
        //fmt.Println(count)
        //fmt.Println(db)
        db.Model(&User{}).Where("email = ?", email).Count(&count)
        //fmt.Println(count)
        if count == 0 {
          db.Create(&user)
        } else {
          db.Model(&User{}).Where("email = ?", email).Find(&user)
        }
        fmt.Println("user", user)
        userId = user.ID
        fmt.Println(conns)
        conns.s.Lock()
        fmt.Println("1")
        conns.m[userId] = ws
        fmt.Println("2")
        conns.s.Unlock()
        fmt.Println("3")
        sendMessage(ws, OK, nil)
        /*if db.NewRecord(user) {
          //db.Create(&user)
          fmt.Println("exists")
        } else {
          fmt.Println(user)
        }*/
      } else {
        sendMessage(ws, NOT_OK, nil)
      }
    case GET_CONTACTS:
      var contacts []Contact
      fmt.Println("contacts0", userId)
      db.Model(&Contact{}).Where("who_id = ?", userId).Find(&contacts)
      fmt.Println("contacts", contacts)
      for _, contactData := range contacts {
        contactId := contactData.WhomID
        var contact User
        db.Model(&User{}).Where("id = ?", contactId).Find(&contact)
        //fmt.Println(contact.ID, contact.Email)
        buf := new(bytes.Buffer)
        binary.Write(buf, binary.LittleEndian, contact.ID)
        buf.WriteString(contact.Email)
        sendMessage(ws, CONTACT, buf.Bytes())
      }
    case ADD_FRIEND:
      var contact User
      email, _ = buf.ReadString(0)
      var count uint
      db.Model(&User{}).Where("email = ?", email).Count(&count)
      if count != 0 {
        db.Model(&User{}).Where("email = ?", email).Find(&contact)
        var who User
        db.Model(&User{}).Where("id = ?", userId).Find(&who)
        db.Create(&Contact{Who: who, Whom: contact})
        buf := new(bytes.Buffer)
        binary.Write(buf, binary.LittleEndian, contact.ID)
        buf.WriteString(contact.Email)
        sendMessage(ws, CONTACT, buf.Bytes())
      }
    case MESSAGE:
      var uid uint64
      binary.Read(buf, binary.LittleEndian, &uid)
      var from User
      db.Model(&User{}).Where("id = ?", userId).Find(&from)
      var to User
      db.Model(&User{}).Where("id = ?", uid).Find(&to)
      contents, _ := buf.ReadString(0)
      text := Text{From: from, To: to, Contents: contents}
      conns.s.Lock()
      tows := conns.m[uid]
      conns.s.Unlock()
      if tows == nil {
        db.Create(&text)
      } else {
        buf := new(bytes.Buffer)
        binary.Write(buf, binary.LittleEndian, userId)
        buf.WriteString(contents)
        sendMessage(tows, MESSAGE, buf.Bytes())
      }
    case DH_PARAM:
      var uid uint64
      binary.Read(buf, binary.LittleEndian, &uid)
      data := buf.Bytes()[8:]
      /*data := make([]byte, buf.size() - 8)
      buf.Read(pub_key)*/
      var from User
      db.Model(&User{}).Where("id = ?", userId).Find(&from)
      var to User
      db.Model(&User{}).Where("id = ?", uid).Find(&to)
      dh := DHParam{First: from, Second: to, Param: data}
      conns.s.Lock()
      tows := conns.m[uid]
      conns.s.Unlock()
      if tows == nil {
        db.Create(&dh)
      } else {
        buf := new(bytes.Buffer)
        binary.Write(buf, binary.LittleEndian, userId)
        buf.Write(data)
        //buf.WriteString(pub_key)
        sendMessage(tows, DH_PARAM, buf.Bytes())
      }
    case DH_PUB_KEY:
      var uid uint64
      binary.Read(buf, binary.LittleEndian, &uid)
      data := buf.Bytes()[8:]
      /*data := make([]byte, buf.size() - 8)
      buf.Read(pub_key)*/
      var from User
      db.Model(&User{}).Where("id = ?", userId).Find(&from)
      var to User
      db.Model(&User{}).Where("id = ?", uid).Find(&to)
      dh := DHPubKey{First: from, Second: to, PubKey: data}
      conns.s.Lock()
      tows := conns.m[uid]
      conns.s.Unlock()
      if tows == nil {
        db.Create(&dh)
      } else {
        buf := new(bytes.Buffer)
        binary.Write(buf, binary.LittleEndian, userId)
        buf.Write(data)
        //buf.WriteString(pub_key)
        sendMessage(tows, DH_PUB_KEY, buf.Bytes())
      }
    }
    fmt.Println("msg", msg)
    //fmt.Println(msg)
  }
}

func sendMessage(ws *websocket.Conn, msgType uint8, contents []byte) {
  buf := new(bytes.Buffer)
  binary.Write(buf, binary.LittleEndian, msgType)
  buf.Write(contents)
  err := websocket.Message.Send(ws, buf.Bytes())
  if err != nil {
    fmt.Println("send err", err)
  }
  fmt.Println("sent", buf.Bytes())
}

func sendEmailViaSendgrid(email string, code string) {
  sendgridKey := "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"
  sg := sendgrid.NewSendGridClientWithApiKey(sendgridKey)
  message := sendgrid.NewMail()
  message.AddTo(email)
  message.SetSubject("Firstchat code")
  message.SetText(code)
  message.SetFrom("firstchat@mmkg.me")
  if r := sg.Send(message); r == nil {
    fmt.Println("Email sent!")
  } else {
    fmt.Println(r)
  }
}