package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

var connectedUsers []User

type User struct {
	id string
	ws *websocket.Conn
}

type Msg struct {
	Type      string `json:"type,omitempty"`
	SDP       string `json:"sdp,omitempty"`
	To        string `json:"to,omitempty"`
	Candidate string `json:"candidate,omitempty"`
}

func main() {
	app := fiber.New()
	app.Get("/", func(context *fiber.Ctx) error {
		return context.SendFile("./index.html")
	})

	app.Static("/", "/")

	app.Use("/ws/:userId", func(context *fiber.Ctx) error {
		userId := context.Params("userId")
		if websocket.IsWebSocketUpgrade(context) {
			context.Locals("id", userId)
			return context.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws/:userId", websocket.New(func(ws *websocket.Conn) {

		connectedUsers = append(connectedUsers, User{
			id: ws.Locals("id").(string),
			ws: ws,
		})

		var msg Msg
		for {
			msg = Msg{}
			err := ws.ReadJSON(&msg)
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway) {
					for i := range connectedUsers {
						if connectedUsers[i].id == ws.Locals("id").(string) {
							connectedUsers[i] = connectedUsers[len(connectedUsers)-1]
							connectedUsers = connectedUsers[:len(connectedUsers)-1]
							break
						}
					}
					_ = ws.Close()
					return
				} else {
					fmt.Println("ERROR HAPPENED :/")
					continue
				}
			}

			switch msg.Type {
			case "offer":
				for _, user := range connectedUsers {
					if user.id == msg.To {
						_ = user.ws.WriteJSON(Msg{
							Type: "offer",
							To:   ws.Locals("id").(string),
							SDP:  msg.SDP,
						})
						break
					} else {
						fmt.Println("Offer same user :/")
					}
				}
			case "answer":
				for _, user := range connectedUsers {
					if user.id == msg.To {
						_ = user.ws.WriteJSON(Msg{
							Type: "answer",
							SDP:  msg.SDP,
						})
						break
					}
				}
			case "candidate":
				for _, user := range connectedUsers {
					if user.id != ws.Locals("id").(string) {
						_ = user.ws.WriteJSON(Msg{
							Type:      "candidate",
							Candidate: msg.Candidate,
						})
						break
					}
				}
			case "end":
				for _, user := range connectedUsers {
					if user.id == msg.To {
						_ = user.ws.WriteJSON(Msg{
							Type: "end",
						})
						break
					}
				}
			default:
				fmt.Println("INVALID MESSAGE TYPE")
				fmt.Println(err)
				continue
			}
		}
	}))

	log.Fatalln(app.Listen(":3000"))
}
