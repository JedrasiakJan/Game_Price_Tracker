# Steam Price Tracker (Sniper Bot) 🎮🎯

![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-BCNF-336791?logo=postgresql)
![Status](https://img.shields.io/badge/Status-Active-brightgreen)
![License](https://img.shields.io/badge/License-MIT-yellow)

Precyzyjny, prywatny snajper rynkowy zintegrowany z Discordem, który monitoruje ceny gier we wszystkich wiodących sklepach cyfrowych (Steam, GOG, Epic Games Store, GreenManGaming itp.) za pośrednictwem API CheapShark.

W przeciwieństwie do masowych trackerów, ta aplikacja działa w sposób wysoce zoptymalizowany – po starcie pobiera aktualne dane rynkowe **wyłącznie dla tytułów, które sam dodałeś do bazy**, oszczędzając transfer rynkowy i eliminując spam.

---

## Spis Treści

📌 [Dlaczego to stworzyłem](#dlaczego-to-stworzyłem) &nbsp;•&nbsp;
[Dla kogo jest ten projekt](#dla-kogo-jest-ten-projekt) &nbsp;•&nbsp;
[Tech Stack](#tech-stack) &nbsp;•&nbsp;
[Szybki Start](#szybki-start) &nbsp;•&nbsp;
[Architektura Kanałów i Komendy](#architektura-kanałów-i-komendy) &nbsp;•&nbsp;
[Pipeline](#jak-działa-potok-przetwarzania) &nbsp;•&nbsp;
[Troubleshooting](#troubleshooting) &nbsp;•&nbsp;
[Decyzje architektoniczne](#decyzje-architektoniczne) &nbsp;•&nbsp;
[Roadmapa](#roadmapa)

---

## Dlaczego to stworzyłem

Chciałem kupować gry w najniższych możliwych cenach, ale nie chciało mi się za każdym razem ręcznie przeglądać kilku sklepów i porównywać ofert na Steam, GOG czy Epicu. Zamiast robić to samo w kółko, postanowiłem zbudować bota, który robi to za mnie – wystarczy raz podać ID gry i cenę docelową, a reszta dzieje się automatycznie w tle.

---

## Dla kogo jest ten projekt

Projekt został stworzony dla graczy i entuzjastów automatyzacji, którzy chcą:

- Kupować gry w absolutnie najniższych cenach, bez ciągłego manualnego sprawdzania.
- Mieć pełną kontrolę nad swoimi alertami bezpośrednio z poziomu dedykowanych kanałów tekstowych na własnym serwerze Discord.
- Przechowywać czystą i ustrukturyzowaną historię cenową gier we własnej lokalnej bazie danych.

---

## Tech Stack

- **Backend:** `Go (Golang)` – zapewniający błyskawiczne działanie goroutine skanujących rynek.
- **Baza Danych:** `PostgreSQL` – relacyjna baza danych zaprojektowana zgodnie z regułami BCNF, z pełną historią zmian cen gier (`game_price_history`).
- **Integracja z Discordem:** `discordgo` – oficjalna biblioteka do obsługi bota, eventów oraz tagowania użytkowników (`<@ID>`).
- **Serwer HTTP:** `Gin Gonic` – lekki framework obsługujący endpointy diagnostyczne (np. `/health`).
- **API Dostawcy:** `CheapShark API` – dostarcza unikalne ID gier, platformy oraz linki przekierowujące do ofert.
- **Środowisko:** `Docker` + `WSL` (Ubuntu) – konteneryzacja bazy danych PostgreSQL zapewniająca izolację i stabilność.

---

## Szybki Start

Poniższa sekcja to **jedyne miejsce**, którego potrzebujesz, żeby uruchomić projekt od zera. Wszystkie kolejne sekcje (troubleshooting, decyzje architektoniczne) są materiałem uzupełniającym.

### Sklonuj repozytorium

```bash
git clone <adres_repozytorium>
cd <nazwa_folderu_projektu>
```

### Skonfiguruj Discord Bota

1. Przejdź do [Discord Developer Portal](https://discord.com/developers/applications) i stwórz nową aplikację.
2. W sekcji **Bot** wygeneruj token dostępowy.
3. W sekcji **Privileged Gateway Intents** włącz: Presence Intent, Server Members Intent oraz Message Content Intent (kluczowe dla `!alert`, `!lista`).
4. W **OAuth2 → URL Generator** zaznacz `bot` + `applications.commands`, a w uprawnieniach bota: `Send Messages`, `Read Message History`, `View Channels`.
5. Zaproś bota na serwer i stwórz kanały: `sprawdz-gre`, `alerty`, `lista_alertow`.

### Ustaw zmienne środowiskowe

Skopiuj szablon zmiennych środowiskowych i utwórz własny plik `.env` (który jest bezpiecznie wykluczony z systemu Git przez `.gitignore`):

```bash
cp env.example .env
```

Otwórz plik `.env` i uzupełnij go swoimi danymi (tokenem Discorda oraz własnym, bezpiecznym hasłem do bazy):

```env
DISCORD_TOKEN=Twój_Prywatny_Token_Bota_Z_Discord_Developer_Portal

DB_USER=steam_admin
DB_PASSWORD=Twoje_Super_Bezpieczne_Haslo_123
DB_NAME=steam_api_prod
```

### Podnieś bazę danych

Nie musisz ręcznie instalować PostgreSQL ani wpisywać długich komend dockerowych. W katalogu głównym projektu, tam gdzie znajduje się plik `docker-compose.yml`, wpisz w terminalu WSL:

```bash
docker compose up -d
```

Docker automatycznie przeczyta dane z Twojego pliku `.env`, pobierze oficjalny obraz `postgres:15-alpine`, założy bazę danych oraz podepnie trwały wolumen, dzięki czemu nie stracisz danych po restarcie kontenera.

### Odpal aplikację

```bash
go run cmd/app/main.go
```

### Checklist przy kolejnym uruchomieniu

Przed każdym kolejnym uruchomieniem bota (np. po restarcie komputera) upewnij się, że Twoja baza danych w Dockerze żyje:

- Sprawdź status kontenerów: `docker ps -a`
- Jeśli baza ma status `Exited`, podnieś ją jedną komendą: `docker start steam_api_db`
- Uruchom bota: `go run cmd/app/main.go` (lub kliknij dwukrotnie w plik `.bat` na pulpicie)

### Automatyzacja na Windows

Stwórz plik `odpal_bota.bat` na pulpicie z zawartością:

```cmd
wsl ~ -e bash -c "cd ~/sciezka_do_twojego_projektu/steam_api && go run cmd/app/main.go"
```

Dwuklik = pełny skan cen bez otwierania terminala ręcznie.

---

## Architektura Kanałów i Komendy

### `#sprawdz-gre` — wyszukiwarka
- **Wpisanie nazwy gry** (np. `Cyberpunk`) → bot zwraca do 3 najlepszych dopasowań z ceną, sklepem i unikalnym **ID Gry** (potrzebnym do alertu).

### `#alerty` — zarządzanie alertami
- **`!alert <ID_GRY> <CENA>`** (np. `!alert 214911 14.50`) → mechanizm **Soft-Upsert**: aktualizuje istniejący alert lub tworzy nowy, bez duplikatów.

### `#lista_alertow` — centrum dowodzenia
- **`!lista`** → pokazuje aktywne alerty: nazwę gry, ID, cenę docelową i aktualną najniższą cenę.
- **`!usun <ID_GRY>`** → **Soft Delete**: ustawia `is_active = FALSE`, zachowując historię.

---

## Jak działa potok przetwarzania

1. Po starcie (`go run cmd/app/main.go`) bot loguje się do Discorda.
2. Równolegle odpala się asynchroniczna **Goroutine**.
3. Skaner pobiera z Postgresa listę ID gier z aktywnym alertem (`is_active = TRUE`).
4. Program odpytuje CheapShark API tylko o te gry, zapisuje historię cen i sprawdza warunki trafienia.
5. Gdy cena spadnie do progu — bot wysyła `SNIPER SHOT!` z tagiem, linkiem do koszyka i dezaktywuje alert.

---
<a name="troubleshooting"></a>
<details>
<summary>🛠️ Problemy i ich rozwiązania (Troubleshooting)</summary>

### 1. Crash bota (`panic: index out of range [1]`)
**Problem:** Sama komenda `!alert` bez argumentów powodowała próbę dostępu do `parts[1]` na tablicy jednoelementowej.
**Rozwiązanie:**
```go
if len(parts) != 3 {
    s.ChannelMessageSend(m.ChannelID, "❌ Niepoprawny format! ...")
    return
}
```

### 2. Nil pointer dereference przy `db.Begin()`
**Problem:** Konstruktor bota nie przyjmował instancji bazy danych.
**Rozwiązanie:** Dependency Injection — jawne przekazanie `*sql.DB` do `NewDiscordBot`.

### 3. Duplikaty alertów i spam powiadomień
**Problem:** Ślepe `INSERT` pozwalały dodać tę samą grę wielokrotnie.
**Rozwiązanie:** Soft-Upsert — najpierw `UPDATE` z `is_active = TRUE`, dopiero gdy `RowsAffected == 0` wykonywany jest `INSERT`.

</details>

<a name="decyzje-architektoniczne"></a>
<details>
<summary>Decyzje architektoniczne</summary>

### On-Demand zamiast `time.Ticker`
Świadoma rezygnacja z pętli 24/7 na rzecz jednorazowego przebiegu przy starcie:
- Pełna kontrola nad cyklem testowym i debugowaniem.
- Oszczędność API Rate Limitingu podczas developmentu lokalnego.

### Konteneryzacja bazy (Docker + WSL)
- Izolacja danych produkcyjnych od Windows.
- Zgodność z 3NF/BCNF dla `game_price_history`.
- Szybki reset/backup jedną komendą.

### Discord 
- `#sprawdz-gre` → wyszukiwanie bez zapisu.
- `#alerty` → zapis/aktualizacja (Upsert).
- `#lista_alertow` → odczyt i Soft Delete.

</details>

## Roadmapa

Projekt jest aktywnie rozwijany. Poniżej lista funkcji, które planuję dodać w kolejnych iteracjach:

- [ ] Opcjonalny tryb `time.Ticker` — automatyczne skanowanie cen w tle (24/7) zamiast wyłącznie On-Demand
- [ ] Wsparcie dla webhooków Epic Games Store — natychmiastowe powiadomienia zamiast pollingu API
- [ ] Komenda `!historia <ID_GRY>` — wykres zmian ceny gry w czasie
- [ ] Panel webowy (dashboard) jako alternatywa dla komend Discordowych
- [ ] Powiadomienia e-mail jako backup dla alertów Discordowych
- [ ] Testy jednostkowe (`go test`) dla logiki Soft-Upsert i parsowania komend
- [ ] Wsparcie dla wielu serwerów Discord jednocześnie (multi-guild)

> Masz pomysł na kolejną funkcję? Otwórz Issue w repozytorium 🙌
