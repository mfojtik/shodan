package main

import (
	"context"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mfojtik/shodan/pkg/config"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
)

var cfg *config.Environment

func init() {
	var err error
	cfg, err = config.Read()
	if err != nil {
		panic(err)
	}
}

func setupShutdownSignalHandling(shutdown context.CancelFunc) {
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGINT)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sigChannel:
		log.Println("Received SIGINT, shutting down ...")
		shutdown()
	}
}

func main() {
	api := slack.New(
		cfg.Slack.BotToken,
		slack.OptionDebug(cfg.Debug),
		slack.OptionLog(log.New(os.Stdout, "api: ", log.Lshortfile|log.LstdFlags)),
		slack.OptionAppLevelToken(cfg.Slack.AppToken),
	)
	client := socketmode.New(
		api,
		socketmode.OptionDebug(cfg.Debug),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	botContext, shutdown := context.WithCancel(context.Background())
	go setupShutdownSignalHandling(shutdown)

	// these are set in slack handler, but read in /healthz endpoint
	var (
		slackEventHandlerReady  atomic.Value
		slackEventHandlerBooted atomic.Value
	)
	slackEventHandlerReady.Store(false)
	slackEventHandlerBooted.Store(false)

	go func() {
		client.Debugf("Waiting for slack events ...")
		for evt := range client.Events {
			switch evt.Type {
			case socketmode.EventTypeConnecting:
			case socketmode.EventTypeConnectionError:
				// when the connection failed, turn the healthz to red
				slackEventHandlerReady.Store(false)
				client.Debugf("Connection failed. Retrying later...")
			case socketmode.EventTypeConnected:
				// when we reconnect to slack, turn the healthz check back to ready
				if booted := slackEventHandlerBooted.Load(); booted == true {
					slackEventHandlerReady.Store(true)
				}
				client.Debugf("Connected to Slack with Socket Mode.")
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					continue
				}

				// we ignore some events for now (like links shared events)
				switch slackevents.EventsAPIType(eventsAPIEvent.InnerEvent.Type) {
				case slackevents.LinkShared:
					continue
				}
				client.Debugf("[shodan][debug] received event: %s", spew.Sdump(eventsAPIEvent))
				client.Ack(*evt.Request)

				switch eventsAPIEvent.Type {
				case slackevents.CallbackEvent:
					innerEvent := eventsAPIEvent.InnerEvent
					switch ev := innerEvent.Data.(type) {
					case *slackevents.AppMentionEvent:
						log.Printf("[shodan][debug] received mention %s", ev.Channel)
						user, err := client.GetUserInfo(ev.User)
						if err != nil {
							log.Printf("Failed to get user %q info: %v", ev.User, err)
						}
						if _, _, err := api.PostMessage(ev.Channel, slack.MsgOptionText(fmt.Sprintf("Yes, hello @%s.", user.Profile.DisplayName), false)); err != nil {
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
				// this usually happens on shutdown, nothing to handle here.
			default:
				client.Debugf("WARNING: received unhandled event type: %q", evt.Type)
			}
		}
	}()

	// run healthz endpoint.
	// fly.io use this to measure the health of this app, if it fails, it restarts it.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if ready := slackEventHandlerReady.Load(); ready == true {
			log.Println("/healthz OK")
			w.WriteHeader(http.StatusOK)
			return
		}
		log.Println("/healthz NOT_READY")
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	go http.ListenAndServe(":8080", nil)

	// run the main slack handler
	if err := client.RunContext(botContext); err != nil && err != context.Canceled {
		log.Fatalf("error running slack handler: %v", err)
	}
}
