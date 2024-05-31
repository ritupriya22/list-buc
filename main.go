package main
 
import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
	"encoding/json"
    "os"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	ginSwagger "github.com/swaggo/gin-swagger"
    swaggerFiles "github.com/swaggo/files"
	"go.uber.org/zap"
    _ "github.com/ritupriya22/list-buc/docs"
)
var logger, _ = zap.NewProduction()

type ListBinParams struct {
    User   string `json:"user_id"`
}
type ErrorResponse struct {
	Code             int    `json:"status_code"`
	ErrorDescription string `json:"error_description"`
}
type Resp struct {
	Response struct {
		Status       string                   `json:"request_status"`
		Data         []map[string]interface{} `json:"data"`
	} `json:"response"`
}
// Swagger handler setup remains unchanged

// Swagger annotation for the listBins operation
// @Summary List bins for a user
// @Description List bins for a user with the given parameters
// @Tags bins
// @Accept  json
// @Produce  json
// @Param params body ListBinParams true "ListBinParams"
// @Success 200 {object} Resp
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /listBins [get] 
func main() {
	// Initialize the Gin engine
	r := gin.Default()
    gin.SetMode(gin.ReleaseMode)
    os.Setenv("GIN_MODE", "release")
	// Define the route for updating the bucket data
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/listBins", func(c *gin.Context) {
		// Extract inputs from the request body
		var requestBody ListBinParams
		if err := c.BindJSON(&requestBody); err!= nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Code: http.StatusBadRequest, ErrorDescription:"Incorrect User Id given for listing bins"})
			return
		}
 
		// Database connection details
		host := "noobaa-db-pg.openshift-storage.svc.cluster.local"
		port := 5432 // Default PostgreSQL port
		user := "noobaa"
		password := "1OzXjadKG5h0zQ=="
		dbName := "nbcore"
 
		// Connect to the database
		psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbName)
		db, err := sql.Open("postgres", psqlInfo)
		if err!= nil {
			log.Fatal(err)
		}
		defer db.Close()
 
		// Check the connection
		err = db.Ping()
		if err!= nil {
			log.Fatal(err)
		}

		// Prepare the SQL statement to retrieve the _id
		sqlStatement := `
		SELECT json_agg(json_build_object('name', data->>'name', 'objects_count', (data->'storage_stats')->>'objects_count'::text, 'objects_size',(data->'storage_stats')->>'objects_size'::text, 'last_update', data->>'last_update'::text)) AS aggregated_data FROM public.buckets WHERE data->>'userId' = $1;`
 
		var result []byte
		err = db.QueryRow(sqlStatement, requestBody.User).Scan(&result)
		if err!= nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Code: http.StatusInternalServerError, ErrorDescription:"Unable to list bins for the user"})
            zap.L().Info("Failed to query", zap.Error(err), zap.Time("timestamp", time.Now()), zap.String("user_id", requestBody.User))
            logger.Error("Failed to query", zap.Error(err), zap.String("user_id", requestBody.User))
			//log.Fatalf("Failed to query: %v", err)
		}

		var jsonData []map[string]interface{}
		err = json.Unmarshal(result, &jsonData)
		if err!= nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Code: http.StatusInternalServerError, ErrorDescription:"Unable to list bins for the user"})
            zap.L().Info("Failed to unmarshal JSON", zap.Error(err), zap.Time("timestamp", time.Now()), zap.String("user_id", requestBody.User))
            logger.Error("Failed to unmarshal JSON", zap.Error(err), zap.String("user_id", requestBody.User))
			//log.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		for i, item := range jsonData {
			// Extract the last_update value as a string
			var lastUpdateInt int64
			lastUpdateStr, exists := item["last_update"]
			if!exists || lastUpdateStr == "" {
				log.Printf("No last_update found in item #%d\n", i)
				continue
			}
		
			if lastUpdateStr, ok := item["last_update"].(string); ok {
				lastUpdateInt, err = strconv.ParseInt(lastUpdateStr, 10, 64)
				if err!= nil {
					fmt.Println("Error parsing Unix timestamp:", err)
					return
				}
			}
		
			// Convert the Unix timestamp to a time.Time object in UTC
			lastUpdateTime := time.Unix(lastUpdateInt/1000, 0)
		
			// Format the time as desired
			formattedTime := lastUpdateTime.Format(time.RFC3339)
		
			// Update the item with the formatted time
			item["last_update"] = formattedTime
		
			// Optionally, update the original slice with the modified item
			jsonData[i] = item
		}
	
	c.JSON(http.StatusOK, Resp{Response: struct {
		Status string "json:\"request_status\""
		Data []map[string]interface{} "json:\"data\""
	}{Status: "Bins Succcesfully Listed for the User", Data: jsonData}})
	zap.L().Info("Bins Succcesfully Listed for the User", zap.Time("start", time.Now()), zap.Int("status", http.StatusOK), zap.String("user_id", requestBody.User))
	//c.JSON(http.StatusOK, jsonData)
    })
	// Start the server
	r.Run(":5000") // Listen and serve on 0.0.0.0:8080
}
