package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type Promotion struct {
	ID             string    `json:"id"`
	Price          float64   `json:"price"`
	ExpirationDate time.Time `json:"expiration_date"`
}

type PromotionOut struct {
	ID             string `json:"id"`
	Price          string `json:"price"`
	ExpirationDate string `json:"expiration_date"`
}

func convertPromotion(promotion Promotion) PromotionOut {
	promotionOut := PromotionOut{
		ID:             promotion.ID,
		Price:          fmt.Sprintf("%.2f", promotion.Price),
		ExpirationDate: promotion.ExpirationDate.Format(time.DateTime),
	}
	return promotionOut
}

// In-memory cache for storing promotions
var promotionsCache []Promotion

// Map to store the index of the promotion in the slice
var promotionIndexMap = make(map[string]int)

// Mutex to synchronize access to the cache
var mutex = &sync.Mutex{}

// Load the CSV file and parse the data into the in-memory cache
func uploadCSV(file *multipart.FileHeader) error {

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	r := csv.NewReader(src)
	r.Comma = ','
	r.LazyQuotes = true

	// Clean the cache and the indexes from the map
	mutex.Lock()
	promotionsCache = make([]Promotion, 0, cap(promotionsCache))
	for k := range promotionIndexMap {
		delete(promotionIndexMap, k)
	}
	mutex.Unlock()

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		price, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			return err
		}

		expirationDate, err := time.Parse("2006-01-02 15:04:05 -0700 MST", record[2])
		if err != nil {
			fmt.Println(err)
			return err
		}

		promotion := Promotion{
			ID:             record[0],
			Price:          price,
			ExpirationDate: expirationDate,
		}

		// Store the promotion in the cache and the index in the map
		mutex.Lock()
		promotionsCache = append(promotionsCache, promotion)
		promotionIndexMap[promotion.ID] = len(promotionsCache) - 1
		mutex.Unlock()
	}

	return nil
}

func uploadFile(c *gin.Context) {

	file, err := c.FormFile("file")

	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "file not found."})
		return
	}

	err = uploadCSV(file)
	if err != nil {
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"message": "data updated"})
}

// Get the promotion object by ID from the in-memory cache
func findPromotionByID(promotionID string) (Promotion, bool) {
	mutex.Lock()
	defer mutex.Unlock()

	index, ok := promotionIndexMap[promotionID]
	if !ok {
		return Promotion{}, false
	}

	return promotionsCache[index], true
}

func getPromotionsByID(c *gin.Context) {

	var id = c.Param("id")
	promotion, err := findPromotionByID(id)

	if !err {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "not found."})
		return
	}

	c.IndentedJSON(http.StatusOK, convertPromotion(promotion))
}

func main() {

	// Load configuration file
	viper.SetConfigFile("config.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	// Override configuration for production environment
	env := viper.GetString("APP_ENV")
	fmt.Println(env)
	if env == "production" {
		viper.SetConfigFile("config.prod.yaml")
		err := viper.MergeInConfig()
		if err != nil {
			panic(err)
		}
		gin.SetMode("release")
	}

	router := gin.Default()

	router.GET("/promotions/:id", getPromotionsByID)
	router.POST("promotions/upload", uploadFile)
	router.Run(":" + viper.GetString("port"))
}
