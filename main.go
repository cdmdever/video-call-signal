package main

import (
	"crypto/tls"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

type User struct {
	id     string
	userId string
	ws     *websocket.Conn
}

type Msg struct {
	Type      string `json:"type,omitempty"`
	SDP       string `json:"sdp,omitempty"`
	To        string `json:"to,omitempty"`
	Candidate string `json:"candidate,omitempty"`
}

func main() {
	connectedUsers := make([]User, 0)

	app := fiber.New()
	app.Get("/", func(context *fiber.Ctx) error {
		clientIP := context.IP()
		return context.SendString(clientIP)
	})

	app.Use("/ws/:userId", func(context *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(context) {
			context.Locals("id", uuid.New().String())
			return context.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws/:userId", websocket.New(func(ws *websocket.Conn) {
		userId := ws.Params("userId")

		connectedUsers = append(connectedUsers, User{
			id:     ws.Locals("id").(string),
			userId: userId,
			ws:     ws,
		})

		var msg Msg
		for {
			msg = Msg{}
			err := ws.ReadJSON(&msg)
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway) {
					for i := range connectedUsers {
						if connectedUsers[i].userId == userId {
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
					if user.userId == msg.To {
						_ = user.ws.WriteJSON(Msg{
							Type: "offer",
							To:   userId,
							SDP:  msg.SDP,
						})
					}
				}
			case "answer":
				for _, user := range connectedUsers {
					if user.userId == msg.To {
						_ = user.ws.WriteJSON(Msg{
							Type: "answer",
							SDP:  msg.SDP,
						})
					}
				}
			case "candidate":
				for _, user := range connectedUsers {
					if user.userId != userId {
						_ = user.ws.WriteJSON(Msg{
							Type:      "candidate",
							Candidate: msg.Candidate,
						})
					}
				}
			case "end":
				for _, user := range connectedUsers {
					if user.userId == msg.To {
						_ = user.ws.WriteJSON(Msg{
							Type: "end",
						})
					}
				}
			default:
				fmt.Println("INVALID MESSAGE TYPE")
				fmt.Println(err)
				continue
			}
		}
	}))

	htmlApp := fiber.New()
	htmlApp.Get("/", func(context *fiber.Ctx) error {
		return context.SendFile("fluidSim.html")
	})

	// TLS Configuration
	certFile := "fullchain.pem" // Path to your certificate file
	keyFile := "privkey.pem"    // Path to your private key file

	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = make([]tls.Certificate, 1)

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("failed to load TLS certificates: %s", err)
	}
	tlsConfig.Certificates[0] = cert

	// Start WebSocket server (port 8089) in a goroutine
	go func() {
		log.Println("WebSocket server started on http://localhost:8089")
		if err := app.ListenTLS(":8089", certFile, keyFile); err != nil {
			log.Fatalf("Failed to start WebSocket server: %s", err)
		}
	}()

	// Start HTML server (port 443) with TLS (HTTPS)
	log.Println("HTML server started on https://rusted.app:443")
	log.Fatalln(htmlApp.ListenTLS(":443", certFile, keyFile))
}
