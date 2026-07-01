package main

import (
	"log"
	"net/http"
	"os"
	"steam_api/internal/client"
	"steam_api/internal/database"
	"steam_api/internal/discord"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("Starting Steam Api App")

	err := godotenv.Load()
	if err != nil {
		log.Println("Informacja: Nie znaleziono pliku .env, czytam bezpośrednio ze środowiska")
	}

	db := database.InitDB()
	defer db.Close()

	sharkClient := client.NewCheapSharkClient()

	discordToken := os.Getenv("DISCORD_TOKEN")
	if discordToken == "" {
		log.Fatalf("BŁĄD: Zmienna DISCORD_TOKEN jest pusta! Sprawdź plik .env")
	}

	bot, err := discord.NewDiscordBot(discordToken, sharkClient, db)
	if err != nil {
		log.Fatalf("Failed to initialize Discord Bot: %v", err)
	}

	err = bot.Session.Open()
	if err != nil {
		log.Fatalf("Failed to open Discord session: %v", err)
	}
	defer bot.Session.Close()
	log.Println("Discord Bot is now online and listening!")

	go func() {
		log.Println("Goroutine: Sprawdzam aktywne alerty w bazie danych...")

		gameIDs, err := database.GetActiveAlertGameIDs(db)
		if err != nil {
			log.Printf("Goroutine Error: Nie udało się pobrać ID gier z bazy: %v", err)
			return
		}

		if len(gameIDs) == 0 {
			log.Println("Goroutine: Brak aktywnych alertów do sprawdzenia przy starcie.")
			return
		}
		log.Printf("Goroutine: Znaleziono %d obserwowanych gier. Odpytuję CheapShark API...", len(gameIDs))

		var trackedDeals []client.CheapSharkDeal
		for _, id := range gameIDs {
			deals, err := sharkClient.FetchDealsForGame(id)
			if err != nil {
				log.Printf("Goroutine Error: Nie udało się pobrać ceny dla gry ID %s: %v", id, err)
				continue
			}
			trackedDeals = append(trackedDeals, deals...)
		}

		if len(trackedDeals) == 0 {
			log.Println("Goroutine: Nie znaleziono żadnych aktualnych ofert dla obserwowanych gier.")
			return
		}

		err = database.SaveDeals(db, trackedDeals)
		if err != nil {
			log.Printf("Goroutine Error: Nie udało się zapisać cen do bazy: %v", err)
			return
		}
		log.Println("Goroutine Success: Aktualne ceny obserwowanych gier zrzucone do bazy.")

		bot.ProcessDealsForAlerts(trackedDeals)
	}()

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "working"})
	})
	r.Run(":8080")
}
