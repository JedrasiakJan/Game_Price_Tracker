package client

import (
	"encoding/json"
	"io"
	"net/http"
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
