package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	m "github.com/beep/beepnet-spectrum-view/model"
)

func sampleHandler(c *gin.Context, db *sql.DB) {
	var resp []m.Sample

	var survey_id int64

	survey_id, err := strconv.ParseInt(c.Query("survey_id"), 0, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	log.Printf("Retrieve survey_id = %d", survey_id)
	rows, err :=
		db.Query(`SELECT power, freq, bandwidth
				  FROM sample
				  WHERE survey_id = $1
				  ORDER BY freq`, survey_id)
	if err != nil {
		log.Printf("Error querying sample: %q", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	for rows.Next() {
		var sample m.Sample
		rows.Scan(&sample.Power, &sample.Freq, &sample.Bandwidth)
		resp = append(resp, sample)
	}
	c.JSON(http.StatusOK, gin.H{"samples": resp})
}

func surveyHandler(c *gin.Context, db *sql.DB) {
	var resp []m.Survey

	rows, err := db.Query(`SELECT * FROM survey`)
	if err != nil {
		log.Printf("Error querying survey: %q", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	for rows.Next() {
		var survey m.Survey
		var locbuf []byte
		rows.Scan(&survey.Id, &survey.Label, &locbuf, &survey.Time)
		if err := m.ToPoint(string(locbuf), &survey.Location); err != nil {
			log.Printf("ERROR PARSING POINT: %s", err)
		}
		log.Printf("Survey: %s", &survey)
		resp = append(resp, survey)
	}

	c.JSON(http.StatusOK, gin.H{"surveys": resp})
}

type IncomingUpload struct {
	Survey  m.Survey         `json:"survey"`
	Samples []m.SampleVector `json:"samples"`
}

func uploadHandler(c *gin.Context, db *sql.DB) {
	var file multipart.File

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, "Bad request: %q", err)
		return
	}

	var incoming IncomingUpload
	buf := new(bytes.Buffer)
	buf.ReadFrom(file)

	if err := json.Unmarshal(buf.Bytes(), &incoming); err != nil {
		c.String(http.StatusBadRequest, "Invalid JSON: %q", err)
		return
	}

	insertDone := make(chan error)

	go func(db *sql.DB, inc *IncomingUpload, insertDone chan error) {
		var tx *sql.Tx
		tx, err := db.Begin()

		rollbackAndExit := func(err error) {
			if tx != nil {
				tx.Rollback()
			}
			log.Printf("DB Error: %q", err)
			insertDone <- err
		}

		if err != nil {
			rollbackAndExit(err)
			return
		}

		var surveyID int
		err = tx.QueryRow(`INSERT INTO survey
				(label, location, time)
				VALUES ($1, $2, $3)
				RETURNING id`,
			inc.Survey.Label, inc.Survey.Location.String(),
			inc.Survey.Time).Scan(&surveyID)

		if err != nil {
			rollbackAndExit(err)
			return
		}

		for _, sampleVec := range inc.Samples {
			sample := sampleVec.ToSample()
			if _, err := tx.Exec(`INSERT INTO sample
					(power, freq, bandwidth, survey_id)
					VALUES ($1, $2, $3, $4)`,
				sample.Power, sample.Freq,
				sample.Bandwidth, surveyID); err != nil {
				rollbackAndExit(err)
				return
			}
		}

		if err := tx.Commit(); err != nil {
			rollbackAndExit(err)
			return
		}

		log.Printf("Survey %q ingested with %d samples",
			inc.Survey.Label, len(inc.Samples))

		insertDone <- nil
	}(db, &incoming, insertDone)

	if insertErr := <-insertDone; insertErr != nil {
		c.String(http.StatusInternalServerError, "DB Error: %q", insertErr)
	} else {
		c.String(http.StatusOK, "OK")
	}
}

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error opening db: %q", err)
	}

	router := gin.Default()
	router.Use(gin.Logger())

	router.LoadHTMLGlob("templates/*.tmpl.html")
	router.Static("/static", "static")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl.html", gin.H{
			"title": "Spectrum Viewer",
		})
	})

	router.GET("/survey", func(c *gin.Context) {
		surveyHandler(c, db)
	})

	router.GET("/sample", func(c *gin.Context) {
		sampleHandler(c, db)
	})

	router.POST("/upload", func(c *gin.Context) {
		uploadHandler(c, db)
	})

	router.Run(":" + port)
}
