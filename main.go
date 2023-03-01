package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

func cleanup() {
	log.Println("🛬 Hangar server shutting down.")
}

func parseScopes(s string) []string {
	return strings.Split(s, ",")
}

func loadConfig(provider string, baseUrl string) oauth2.Config {
	scopes := parseScopes(os.Getenv(fmt.Sprintf("%s_DEFAULT_SCOPES", strings.ToUpper(provider))))
	return oauth2.Config{
		ClientID:     os.Getenv(fmt.Sprintf("%s_CLIENT_ID", strings.ToUpper(provider))),
		ClientSecret: os.Getenv(fmt.Sprintf("%s_CLIENT_SECRET", strings.ToUpper(provider))),
		Endpoint: oauth2.Endpoint{
			AuthURL:  os.Getenv(fmt.Sprintf("%s_AUTH_URL", strings.ToUpper(provider))),
			TokenURL: os.Getenv(fmt.Sprintf("%s_TOKEN_URL", strings.ToUpper(provider))),
		},
		RedirectURL: fmt.Sprintf("%s/auth/%s/callback", baseUrl, provider),
		Scopes:      scopes,
	}
}

func main() {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Fatal("Error loading .env file: ", err)
	}

	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	baseUrl := os.Getenv("BASE_URL")
	if baseUrl == "" {
		baseUrl = fmt.Sprintf("http://%s:%s", host, port)
	}

	configs := make(map[string]oauth2.Config)
	for _, e := range os.Environ() {
		parts := strings.Split(e, "=")
		key := parts[0]
		if strings.HasSuffix(key, "_CLIENT_ID") {
			provider := strings.ToLower(strings.TrimSuffix(key, "_CLIENT_ID"))
			conf := loadConfig(provider, baseUrl)
			log.Printf("Detected '%s' configuration.\n", provider)
			configs[provider] = conf
			http.HandleFunc(fmt.Sprintf("/auth/%s", provider), func(w http.ResponseWriter, r *http.Request) {
				authURL := conf.AuthCodeURL("state")
				http.Redirect(w, r, authURL, http.StatusFound)
			})
			http.HandleFunc(fmt.Sprintf("/auth/%s/callback", provider), func(w http.ResponseWriter, r *http.Request) {
				token, err := conf.Exchange(oauth2.NoContext, r.URL.Query().Get("code"))
				if err != nil {
					fmt.Fprintf(w, "Error: %s", err)
				}
				fmt.Fprintf(w, "Token: %+v", token)
			})
		}
	}

	// Register for SIGTERM (Ctrl-C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cleanup()
		os.Exit(1)
	}()

	addr := fmt.Sprintf("%s:%s", host, port)
	log.Printf("🛫 Hangar server starting at %s.\n", addr)
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}
