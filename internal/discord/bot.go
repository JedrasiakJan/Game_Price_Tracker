package discord

import (
	"database/sql"
	"fmt"
	"log"
	"steam_api/internal/client"
	"steam_api/internal/database"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type DiscordBot struct {
	Session     *discordgo.Session
	SharkClient *client.CheapSharkClient
	DB          *sql.DB
}

func NewDiscordBot(token string, sharkClient *client.CheapSharkClient, db *sql.DB) (*DiscordBot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent
	bot := &DiscordBot{
		Session:     dg,
		SharkClient: sharkClient,
		DB:          db,
	}
	dg.AddHandler(bot.messageCreate)
	return bot, nil
}

func (b *DiscordBot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	channel, err := s.Channel(m.ChannelID)
	if err != nil {
		log.Printf("Discord Error: Failed to get channel info: %v", err)
		return
	}
	switch channel.Name {
	case "sprawdz-gre":
		log.Printf("Discord: User requested game check: %s", m.Content)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("🔍 Szukam ofert dla: **%s**...", m.Content))
		results, err := b.SharkClient.SearchGameByTitle(m.Content)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "❌ Błąd podczas odpytywania API CheapShark.")
			return
		}
		if len(results) == 0 {
			s.ChannelMessageSend(m.ChannelID, "🤷‍♂️ Nie znalazłem żadnej gry o tej nazwie.")
			return
		}
		response := "🎮 **Znalezione dopasowania:**\n"
		for i := 0; i < 3 && i < len(results); i++ {
			response += fmt.Sprintf("• **%s** - Cena: `$%s` (Sklep: **%s**) | ID Gry: `%s`\n",
				results[i].Title, results[i].SalePrice, results[i].GetPlatformName(), results[i].GameID)
		}
		s.ChannelMessageSend(m.ChannelID, response)
	case "alerty":
		if strings.HasPrefix(m.Content, "!alert") {
			parts := strings.Fields(m.Content)
			if len(parts) != 3 {
				s.ChannelMessageSend(m.ChannelID, "❌ Niepoprawny format! Wpisz tutaj: `!alert <ID_GRY> <CENA>`\nPrzykład: `!alert 202350 15.50`")
				return
			}
			gameID := parts[1]
			targetPrice, err := strconv.ParseFloat(parts[2], 64)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "❌ Cena musi być liczbą, np. `19.99` lub `15`")
				return
			}
			err = database.CreateAlert(b.DB, m.Author.ID, gameID, "Gra o ID "+gameID, targetPrice)
			if err != nil {
				log.Printf("Discord Error: Failed to save alert: %v", err)
				s.ChannelMessageSend(m.ChannelID, "❌ Wystąpił błąd podczas zapisywania alertu w bazie.")
				return
			}
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("🎯 **Alert dodany pomyślnie!** Pilnuję gry o ID `%s`. Gdy cena spadnie do **$%0.2f**, dam znać tutaj! 🔥", gameID, targetPrice))
			return
		}
	case "lista_alertow":
		if m.Content == "!lista" {
			alerts, err := database.GetActiveAlertForUsers(b.DB, m.Author.ID)
			if err != nil {
				log.Printf("Discord Error: Failed to fetch user alerts: %v", err)
				s.ChannelMessageSend(m.ChannelID, "❌ Wystąpił błąd podczas pobierania Twoich alertów z bazy.")
				return
			}

			if len(alerts) == 0 {
				s.ChannelMessageSend(m.ChannelID, "ℹ️ Nie masz obecnie żadnych aktywnych alertów. Możesz je dodać na kanale `#alerty`!")
				return
			}

			response := "📋 **Twoje aktywne alerty cenowe:**\n"
			for _, a := range alerts {
				if a.CurrentPrice == 0 {
					response += fmt.Sprintf("• **%s** | ID Gry: `%s` | Twoja cena: `$%0.2f` *(Oczekiwanie na pierwszy skan... ⏳)*\n",
						a.GameTitle, a.GameID, a.TargetPrice)
				} else {
					response += fmt.Sprintf("• **%s** | ID Gry: `%s` | Twoja cena: `$%0.2f` *(Obecnie w sklepach: **$%0.2f**)*\n",
						a.GameTitle, a.GameID, a.TargetPrice, a.CurrentPrice)
				}
			}
			s.ChannelMessageSend(m.ChannelID, response)
			return
		}

		if strings.HasPrefix(m.Content, "!usun") {
			parts := strings.Fields(m.Content)
			if len(parts) != 2 {
				s.ChannelMessageSend(m.ChannelID, "❌ Niepoprawny format! Wpisz: `!usun <ID_GRY>`\nPrzykład: `!usun 214911`")
				return
			}
			gameID := parts[1]
			wasDeactivated, err := database.DeactivateAlertManually(b.DB, m.Author.ID, gameID)
			if err != nil {
				log.Printf("Discord Error: Failed to deactivate alert: %v", err)
				s.ChannelMessageSend(m.ChannelID, "❌ Wystąpił błąd bazy podczas usuwania alertu.")
				return
			}
			if !wasDeactivated {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("🤷‍♂️ Nie znalazłem żadnego aktywnego alertu dla gry o ID `%s`.", gameID))
				return
			}
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("🗑️ **Alert usunięty pomyślnie!** Przestałem obserwować grę o ID `%s`.", gameID))
			return
		}
	}
}

func (b *DiscordBot) ProcessDealsForAlerts(deals []client.CheapSharkDeal) {
	var alertChannelID string
	guilds := b.Session.State.Guilds
	for _, g := range guilds {
		channels, err := b.Session.GuildChannels(g.ID)
		if err != nil {
			continue
		}
		for _, ch := range channels {
			if ch.Name == "alerty" {
				alertChannelID = ch.ID
				break
			}
		}
		if alertChannelID != "" {
			break
		}
	}

	if alertChannelID == "" {
		log.Println("Discord Alert: Nie znaleziono kanału o nazwie 'alerty' na serwerze!")
		return
	}

	for _, deal := range deals {
		price, err := strconv.ParseFloat(deal.SalePrice, 64)
		if err != nil {
			continue
		}

		triggered, err := database.CheckAlertsInDB(b.DB, deal.GameID, price)
		if err != nil {
			log.Printf("Error checking alerts for game %s: %v", deal.GameID, err)
			continue
		}

		for _, ta := range triggered {
			msg := fmt.Sprintf("🚨 **SNIPER SHOT!** <@%s>, Twoje polowanie zakończone powodzeniem!\n"+
				"🎮 Gra: **%s**\n"+
				"• Twoja cena docelowa: **$%0.2f**\n"+
				"• **Aktualna cena w promocji: $%0.2f** 🔥\n"+
				"• Platforma: **%s**\n"+
				"🛒 **Kup tutaj:** https://www.cheapshark.com/redirect?dealID=%s\n\n"+
				"🎯 Łap okazję póki gorąca!",
				ta.DiscordUserID, deal.Title, ta.TargetPrice, ta.SalePrice, deal.GetPlatformName(), deal.DealID)

			b.Session.ChannelMessageSend(alertChannelID, msg)
			log.Printf("Discord: Sent alert for game %s to user %s", ta.GameTitle, ta.DiscordUserID)

			// Wyłączamy alert w bazie, żeby nie spamować przy kolejnym skanie
			err = database.DisableAlert(b.DB, ta.DiscordUserID, deal.GameID)
			if err != nil {
				log.Printf("Error disabling alert: %v", err)
			}
		}
	}
}
