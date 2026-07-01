package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type CheapSharkClient struct {
	httpClient *http.Client
	baseURL    string
}

type CheapSharkDeal struct {
	GameID      string `json:"gameID"`
	Title       string `json:"title"`
	SalePrice   string `json:"salePrice"` /* cheapshark zwraca cene jako string !!! */
	NormalPrice string `json:"normalPrice"`
	StoreID     string `json:"storeID"`
	DealID      string `json:"dealID"`
}

type CheapSharkGameResponse struct {
	Info struct {
		Title string `json:"title"`
	} `json:"info"`
	Deals []struct {
		StoreID     string `json:"storeID"`
		Price       string `json:"price"`
		RetailPrice string `json:"retailPrice"`
		DealID      string `json:"dealID"`
	} `json:"deals"`
}

func NewCheapSharkClient() *CheapSharkClient {
	return &CheapSharkClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://www.cheapshark.com/api/1.0",
	}
}

func (c *CheapSharkClient) FetchDeals() ([]CheapSharkDeal, error) {
	fullurl := c.baseURL + "/deals"
	resp, err := c.httpClient.Get(fullurl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var deals []CheapSharkDeal
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &deals)
	if err != nil {
		return nil, err
	}
	return deals, nil
}

/* funkcja pomocnicza fetchdeals */

func (d CheapSharkDeal) GetPlatformName() string {
	switch d.StoreID {
	case "1":
		return "Steam"
	case "2":
		return "GamersGate"
	case "3":
		return "GreenManGaming"
	case "7":
		return "GOG"
	case "25":
		return "Epic_Games_Store"
	default:
		return "Other"
	}
}

// SearchGameByTitle szuka gier w API po nazwie wpisanej przez użytkownika
func (c *CheapSharkClient) SearchGameByTitle(title string) ([]CheapSharkDeal, error) {
	fullurl := fmt.Sprintf("%s/deals?title=%s", c.baseURL, url.QueryEscape(title))
	resp, err := c.httpClient.Get(fullurl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var results []CheapSharkDeal
	err = json.Unmarshal(body, &results)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (c *CheapSharkClient) FetchDealsForGame(gameID string) ([]CheapSharkDeal, error) {
	fullurl := fmt.Sprintf("%s/games?id=%s", c.baseURL, gameID)
	resp, err := c.httpClient.Get(fullurl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data CheapSharkGameResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	var deals []CheapSharkDeal
	for _, d := range data.Deals {
		deals = append(deals, CheapSharkDeal{
			GameID:      gameID,
			Title:       data.Info.Title,
			SalePrice:   d.Price,
			NormalPrice: d.RetailPrice,
			StoreID:     d.StoreID,
			DealID:      d.DealID,
		})
	}

	return deals, nil
}
