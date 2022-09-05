package main

import (
	"context"
	"github.com/davecgh/go-spew/spew"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
)

func main() {
	appToken := os.Getenv("SLACK_APP_TOKEN")
	if appToken == "" {
		log.Fatalf("SLACK_APP_TOKEN must be set.")
	}

	if !strings.HasPrefix(appToken, "xapp-") {
		log.Fatalf("SLACK_APP_TOKEN must have the prefix \"xapp-\".")
	}

	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if botToken == "" {
		log.Fatalf("SLACK_BOT_TOKEN must be set.")
	}

	if !strings.HasPrefix(botToken, "xoxb-") {
		log.Fatalf("SLACK_BOT_TOKEN must have the prefix \"xoxb-\".")
	}

	enableDebugMode := false
	if debugMode := os.Getenv("DEBUG_MODE"); len(debugMode) > 0 {
		enableDebugMode = true
	}

	// SIGINT handling
	botContext, botShutdown := context.WithCancel(context.Background())
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGINT)
	go func() {
		select {
		case <-sigChannel:
			log.Println("Received SIGINT, shutting down ...")
			botShutdown()
		}
	}()

	api := slack.New(
		botToken,
		slack.OptionDebug(enableDebugMode),
		slack.OptionLog(log.New(os.Stdout, "api: ", log.Lshortfile|log.LstdFlags)),
		slack.OptionAppLevelToken(appToken),
	)

	client := socketmode.New(
		api,
		socketmode.OptionDebug(enableDebugMode),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	var (
		slackEventHandlerReady  atomic.Value
		slackEventHandlerBooted atomic.Value
	)
	slackEventHandlerReady.Store(false)
	slackEventHandlerBooted.Store(false)

	go func() {
		client.Debugf("Starting Slack event handler ...")
		for evt := range client.Events {
			switch evt.Type {
			case socketmode.EventTypeConnecting:
			case socketmode.EventTypeConnectionError:
				// when the connection failed, turn the healthz to red
				slackEventHandlerReady.Store(false)
				log.Println("Connection failed. Retrying later...")
			case socketmode.EventTypeConnected:
				// when we reconnect to slack, turn the healthz check back to ready
				if booted := slackEventHandlerBooted.Load(); booted == true {
					slackEventHandlerReady.Store(true)
				}
				log.Println("Connected to Slack with Socket Mode.")
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					log.Printf("WARNING: Ignored event:\n %s\n", spew.Sdump(evt.Data))
					continue
				}

				log.Printf("[shodan][debug] received event: %s", spew.Sdump(eventsAPIEvent))
				client.Ack(*evt.Request)

				switch eventsAPIEvent.Type {
				case slackevents.CallbackEvent:
					innerEvent := eventsAPIEvent.InnerEvent
					switch ev := innerEvent.Data.(type) {
					case *slackevents.AppMentionEvent:
						log.Printf("[shodan][debug] received mention %s", ev.Channel)
						_, _, err := api.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))
						if err != nil {
							log.Printf("Failed posting message: %v", err)
						}
					}
				default:
					client.Debugf("unsupported Events API event received")
				}
			case socketmode.EventTypeInteractive:
				callback, ok := evt.Data.(slack.InteractionCallback)
				if !ok {
					continue
				}
				log.Printf("[shodan][debug]: Received interaction: %v", spew.Sdump(callback))

				var payload interface{}
				switch callback.Type {
				case slack.InteractionTypeBlockActions:
					// See https://api.slack.com/apis/connections/socket-implement#button
				case slack.InteractionTypeShortcut:
				case slack.InteractionTypeViewSubmission:
					// See https://api.slack.com/apis/connections/socket-implement#modal
				case slack.InteractionTypeDialogSubmission:
				default:
				}
				client.Ack(*evt.Request, payload)
			case socketmode.EventTypeSlashCommand:
				cmd, ok := evt.Data.(slack.SlashCommand)
				if !ok {
					continue
				}
				log.Printf("[shodan][debug]: Received slash command: %v", spew.Sdump(cmd))

				payload := map[string]interface{}{
					"blocks": []slack.Block{
						slack.NewSectionBlock(
							&slack.TextBlockObject{
								Type: slack.MarkdownType,
								Text: "foo",
							},
							nil,
							slack.NewAccessory(
								slack.NewButtonBlockElement(
									"",
									"somevalue",
									&slack.TextBlockObject{
										Type: slack.PlainTextType,
										Text: "bar",
									},
								),
							),
						),
					}}

				client.Ack(*evt.Request, payload)
			case socketmode.EventTypeHello:
				// we only receive hello after boot and connection to slack
				slackEventHandlerBooted.Store(true)
				slackEventHandlerReady.Store(true)
			case socketmode.EventTypeIncomingError:
			default:
				log.Printf("Unexpected event type received: %s\n", evt.Type)
			}
		}
	}()

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if ready := slackEventHandlerReady.Load(); ready == true {
			log.Println("/healthz OK")
			w.WriteHeader(http.StatusOK)
			return
		}
		log.Println("/healthz NOT_READY")
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	// /healthz probe
	go http.ListenAndServe(":8080", nil)

	// slack handler
	if err := client.RunContext(botContext); err != nil && err != context.Canceled {
		log.Fatalf("Error running slack handler: %v", err)
	}
}
