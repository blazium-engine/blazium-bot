package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gorilla/mux"

	"github.com/bwmarrin/discordgo"
	"github.com/servusdei2018/shards"

	"github.com/caarlos0/env"
)

type config struct {
	Token string `env:"BOT_TOKEN,required,notEmpty"`
}

var (
	cfg config
	Mgr *shards.Manager
)

func init() {

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("%+v\n", err)
	}

}



func main() {
	// Create a new router using Gorilla Mux
	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://blazium.app", http.StatusMovedPermanently)
	})

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Set the content type to application/json
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Define a health check response structure
		response := map[string]string{"status": "healthy"}

		// Encode the response as JSON and send it
		json.NewEncoder(w).Encode(response)
	})

	embedHandler := embedMiddleware(r)
	corsHandler := enableCORS(embedHandler)

	// Start the server
	fmt.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", corsHandler))

		// Create a new Discord session using the provided bot token.
		Mgr, err := shards.New("Bot " + cfg.Token)
		if err != nil {
			fmt.Println("[ERROR] Error creating manager,", err)
			return
		}
	
		// Register the messageCreate func as a callback for MessageCreate events.
		Mgr.AddHandler(messageCreate)
		// Register the onConnect func as a callback for Connect events.
		Mgr.AddHandler(onConnect)
	
		// In this example, we only care about receiving message events.
		Mgr.RegisterIntent(discordgo.IntentsGuildMessages)
	
		fmt.Println("[INFO] Starting shard manager...")
	
		// Open a websocket connection to Discord and begin listening.
		err = Mgr.Start()
		if err != nil {
			fmt.Println("[ERROR] Error starting manager,", err)
			return
		}
	
		// Wait here until CTRL-C or other term signal is received.
		fmt.Println("[SUCCESS] Bot is now running.  Press CTRL-C to exit.")
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
		<-sc
	
		// Cleanly close down the Discord session.
		fmt.Println("[INFO] Stopping shard manager...")
	
		Mgr.Shutdown()
	
		fmt.Println("[SUCCESS] Shard manager stopped. Bot is shut down.")
}



func enableCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Set CORS headers
        w.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins, you can restrict this to a specific domain
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

        // Handle preflight OPTIONS requests
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        // Call the next handler
        next.ServeHTTP(w, r)
    })
}

func embedMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get the User-Agent header and convert it to lowercase for case-insensitive comparison
        userAgent := strings.ToLower(r.Header.Get("User-Agent"))

        // Check if the User-Agent contains "discordbot" (case-insensitive)
        if strings.Contains(userAgent, "discordbot") {
            // Set appropriate headers for HTML content and caching
            w.Header().Set("Content-Type", "text/html; charset=utf-8")
            w.Header().Set("Cache-Control", "max-age=3600") // Cache the response for 1 hour

            // Write the Open Graph meta tags for Discord embeds
            w.Write([]byte(`
                <!DOCTYPE html>
                <html lang="en">
                <head>
                    <meta charset="UTF-8">
                    <meta name="viewport" content="width=device-width, initial-scale=1.0">
                    <meta property="og:title" content="Blazium Engine">
                    <meta property="og:description" content="Blazium Engine forked from Godot.">
                    <meta property="og:image" content="https://blazium.app/static/assets/logo.png">
                    <meta property="og:url" content="https://blazium.app">
                    <meta property="og:type" content="website">
                    <meta name="twitter:card" content="summary_large_image">
                    <meta property="og:site_name" content="Blazium Engine">
                    <title>Blazium Engine</title>
                </head>
                <body>
                    <h1>Welcome to Blazium Engine</h1>
                </body>
                </html>
            `))
            return
        }

        // If the User-Agent is not from Discord, pass the request to the next handler
        next.ServeHTTP(w, r)
    })
}

func onConnect(s *discordgo.Session, evt *discordgo.Connect) {
	fmt.Printf("[INFO] Shard #%v connected.\n", s.ShardID)
}

// This function will be called (due to AddHandler above) every time a
// new  message is created on any channel that the authenticated bot has
// access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself.
	// This isn't required in this specific example but it's a good
	// practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	// If the message is "ping" reply with "Pong!"
	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}

	// If the message is "pong" reply with "Ping!"
	if m.Content == "pong" {
		s.ChannelMessageSend(m.ChannelID, "Ping!")
	}

	// If the message is "restart" restart the shard manager and rescale
	// if necessary, all with zero down-time.
	if m.Content == "restart" {
		s.ChannelMessageSend(m.ChannelID, "[INFO] Restarting shard manager...")
		fmt.Println("[INFO] Restarting shard manager...")
		if err := Mgr.Restart(); err != nil {
			fmt.Println("[ERROR] Error restarting manager,", err)
		} else {
			s.ChannelMessageSend(m.ChannelID, "[SUCCESS] Manager successfully restarted.")
			fmt.Println("[SUCCESS] Manager successfully restarted.")
		}
	}
}