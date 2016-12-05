package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/ngs/go-amazon-product-advertising-api/amazon"
)

var currentClient = 0

const requestThrottleError = "You are submitting requests too quickly. Please retry your requests at a slower rate."

// Amazon returns amazon client
func (app *App) Amazon() *amazon.Client {
	client := app.AmazonClients[currentClient]
	currentClient++
	app.Log.Printf("Using Amazon Client %d of %d", currentClient, len(app.AmazonClients))
	if currentClient == len(app.AmazonClients) {
		currentClient = 0
	}
	return client
}

func (app *App) setupAmazonClients() error {
	accessKeyIDs := strings.Split(os.Getenv("AWS_ACCESS_KEY_ID"), ":")
	secretAccessKeys := strings.Split(os.Getenv("AWS_SECRET_ACCESS_KEY"), ":")
	if len(accessKeyIDs) != len(secretAccessKeys) {
		return fmt.Errorf("Specified %d Access Key IDs, but Secret Access Keys was %d",
			len(accessKeyIDs), len(secretAccessKeys))
	}
	clients := []*amazon.Client{}
	associateTag := os.Getenv("AWS_ASSOCIATE_TAG")
	for i, key := range accessKeyIDs {
		secret := secretAccessKeys[i]
		client, err := amazon.New(key, secret, associateTag, amazon.RegionJapan)
		if err != nil {
			return err
		}
		clients = append(clients, client)
	}
	app.AmazonClients = clients
	return nil
}

func getAmazonItemCarousel(items []amazon.Item,
	buildActions func(
		item amazon.Item,
		imgURL string,
		label string,
		title string) []linebot.TemplateAction) *linebot.CarouselTemplate {
	var columns []*linebot.CarouselColumn
	for _, item := range items {
		if len(columns) == 5 {
			break
		}
		title := []rune(item.ItemAttributes.Title)
		if len(title) == 0 || len(item.DetailPageURL) == 0 {
			continue
		}
		if len(title) > 40 {
			title = title[0:40]
		}
		imgURL := item.LargeImage.URL
		if imgURL == "" {
			imgURL = noimgURL
		} else {
			imgURL = strings.Replace(imgURL, "http://ecx.images-amazon.com/", "https://images-na.ssl-images-amazon.com/", -1)
		}
		label := ""
		if len(item.ItemAttributes.Author) > 0 && len(item.ItemAttributes.Author[0]) > 0 {
			label = item.ItemAttributes.Author[0]
		}
		if label == "" && len(item.ItemAttributes.Artist) > 0 {
			label = item.ItemAttributes.Artist
		}
		if label == "" && len(label) == 0 && len(item.ItemAttributes.Creator.Name) > 0 {
			label = item.ItemAttributes.Creator.Name
		}
		if label == "" {
			label = item.ItemAttributes.Manufacturer
		}
		if item.OfferSummary.LowestNewPrice.FormattedPrice != "" {
			if label != "" {
				label = label + " - "
			}
			label = label + item.OfferSummary.LowestNewPrice.FormattedPrice
		}
		if label == "" {
			continue
		}
		strTitle := string(title[0:len(title)])
		actions := buildActions(item, imgURL, label, strTitle)
		column := linebot.NewCarouselColumn(
			imgURL,
			strTitle,
			label,
			actions...,
		)
		columns = append(columns, column)
	}
	return linebot.NewCarouselTemplate(columns...)
}

func (app *App) searchItems(keyword string) ([]amazon.Item, error) {
	param := amazon.ItemSearchParameters{
		Keywords:    keyword,
		SearchIndex: amazon.SearchIndexAll,
		ResponseGroups: []amazon.ItemSearchResponseGroup{
			amazon.ItemSearchResponseGroupLarge,
		},
	}
	res, err := app.Amazon().ItemSearch(param).Do()
	if err != nil {
		app.Log.Printf("Got error %v %v", err, param)
		return []amazon.Item{}, err
	}
	return res.Items.Item, nil
}

func (app *App) lookupItems(ids []string) ([]amazon.Item, error) {
	param := amazon.ItemLookupParameters{
		ItemIDs: ids,
		IDType:  amazon.IDTypeASIN,
		ResponseGroups: []amazon.ItemLookupResponseGroup{
			amazon.ItemLookupResponseGroupLarge,
		},
	}
	res, err := app.Amazon().ItemLookup(param).Do()
	if err != nil {
		return []amazon.Item{}, err
	}
	return res.Items.Item, nil
}
