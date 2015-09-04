package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	m "github.com/beep/beepnet-spectrum-view/model"
)

func longSampleHandler(c *gin.Context, db *sql.DB) {
	type LongSample struct {
		Power float64   `json:"power"`
		Time  time.Time `json:"time"`
	}

	var resp []LongSample

	freq, err := strconv.ParseInt(c.Query("f"), 0, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	dFactor, err := strconv.ParseInt(c.DefaultQuery("df", "0"), 0, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	nSamples, err := strconv.ParseInt(c.DefaultQuery("n", "0"), 0, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	mostRecent, err := strconv.ParseInt(c.DefaultQuery("recent", "0"), 0, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	var rows *sql.Rows
	if mostRecent != 0 {
		rows, err =
			db.Query(`
					SELECT smp.power, sv.time from sample smp, survey sv
					WHERE smp.survey_id = sv.id AND freq = $1 AND decfactor = $2
					ORDER BY time DESC
					LIMIT $1`, freq, dFactor, nSamples)
	} else {
		rows, err =
			db.Query(`
					SELECT smp.power, sv.time from sample smp, survey sv
					WHERE smp.survey_id = sv.id AND freq = $1 AND decfactor = $2
					ORDER BY time`, freq, dFactor)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	var sampCount int
	err = db.QueryRow(`
			SELECT count(*) FROM sample smp, survey sv
			WHERE smp.survey_id = sv.id AND freq = $1 AND decfactor = $2`,
		freq, dFactor).Scan(&sampCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	var sampMod int
	if nSamples > 0 {
		sampMod = int(math.Floor(float64(sampCount) / float64(nSamples)))
	}

	// Guard against not enough samples to fulfill nSamples
	if sampMod == 0 {
		sampMod = 1
	}

	log.Printf("Sample Modulus: %d", sampMod)

	rowsToSlice := func(rows *sql.Rows) []LongSample {
		var out []LongSample
		for n := 0; rows.Next(); n++ {
			if nSamples != 0 && (n%sampMod != 0) {
				continue
			}
			var sample LongSample
			rows.Scan(&sample.Power, &sample.Time)
			out = append(out, sample)
		}
		return out
	}

	resp = rowsToSlice(rows)

	c.Header("Cache-Control", "public, max-age=604800")
	c.JSON(http.StatusOK, gin.H{"samples": resp})
}

func sampleHandler(c *gin.Context, db *sql.DB) {
	var resp []m.Sample

	surveyId, err := strconv.ParseInt(c.Query("survey_id"), 0, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	// dFactor == decimation factor
	// e.g., a dFactor of 10 would decimate a survey of 10k samples to 1k samples
	// Default is 0, or no decimation (1 would have no effect either)
	dFactor, err := strconv.ParseInt(c.DefaultQuery("df", "0"), 0, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
	}

	log.Printf("Retrieve Survey ID(%d) DF(%d)", surveyId, dFactor)
	rows, err :=
		db.Query(`SELECT power, freq, bandwidth
				  FROM sample
				  WHERE survey_id = $1 AND decfactor = $2
				  ORDER BY freq`, surveyId, dFactor)
	if err != nil {
		log.Printf("Error querying sample: %q", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	rowsToSlice := func(rows *sql.Rows) []m.Sample {
		var out []m.Sample
		for rows.Next() {
			var sample m.Sample
			rows.Scan(&sample.Power, &sample.Freq, &sample.Bandwidth)
			out = append(out, sample)
		}
		return out
	}

	resp = rowsToSlice(rows)

	c.Header("Cache-Control", "public, max-age=604800")
	c.JSON(http.StatusOK, gin.H{"samples": resp})
}

func surveyHandler(c *gin.Context, db *sql.DB) {
	var resp []m.Survey

	id, err := strconv.ParseInt(c.DefaultQuery("id", "0"), 0, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
	}

	var rows *sql.Rows
	if id > 0 {
		rows, err = db.Query(`
				SELECT srv.id, srv.label, srv.location, srv.time, st.tags
				FROM survey srv LEFT JOIN survey_tags st ON srv.id = st.survey_id
				WHERE srv.id = $1
				ORDER BY time DESC`, id)
	} else {
		rows, err = db.Query(`
				SELECT srv.id, srv.label, srv.location, srv.time, st.tags
				FROM survey srv LEFT JOIN survey_tags st ON srv.id = st.survey_id
				ORDER BY time DESC`)
	}

	if err != nil {
		log.Printf("Error querying survey: %q", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	for rows.Next() {
		var survey m.Survey
		var locbuf []byte
		var tagsbuf []byte
		rows.Scan(&survey.Id, &survey.Label, &locbuf, &survey.Time, &tagsbuf)
		survey.Tags = m.PGStringToTags(string(tagsbuf))
		if err := m.StringToPoint(string(locbuf), &survey.Location); err != nil {
			log.Printf("ERROR PARSING POINT: %s", err)
		}
		resp = append(resp, survey)
	}

	c.JSON(http.StatusOK, gin.H{"surveys": resp})
}

func uploadHandler(c *gin.Context, db *sql.DB) {
	type IncomingUpload struct {
		Survey  m.Survey         `json:"survey"`
		Samples []m.SampleVector `json:"samples"`
	}

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

	incoming.Survey.RawData = string(buf.Bytes())
	if uploaderId, err := strconv.Atoi(c.Request.Header["X-Uploader-Id"][0]); err != nil {
		c.String(http.StatusInternalServerError, "Missing header: X-Uploader-Id")
		return
	} else {
		incoming.Survey.UploaderId = uploaderId
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
		surveyID, err = inc.Survey.WriteToDB(tx)
		if err != nil {
			rollbackAndExit(err)
			return
		}

		var buffer bytes.Buffer
		preamble := `INSERT INTO sample (power, freq, bandwidth, decfactor, survey_id) VALUES `
		batchSize := 500

		// Decimate the samples with all default decimation factors
		for _, df := range m.DefaultDecimationFactors() {
			log.Printf("DF = %d", df)
			// Decimation
			var dSamples []m.Sample
			rawLen := len(inc.Samples)
			nBins := int(math.Ceil(float64(rawLen) / float64(df)))
			for i := 0; i < int(nBins); i++ {
				endIndex := int(df) * (i + 1)
				if endIndex > rawLen {
					endIndex = rawLen
				}
				binSamples := inc.Samples[int(df)*i : endIndex]
				sum := 0.0
				for _, sample := range binSamples {
					sum += sample[0] // Power
				}
				dSamples = append(dSamples, m.Sample{
					sum / float64(len(binSamples)),
					uint64((binSamples[0][1] + binSamples[len(binSamples)-1][1]) / 2),
					uint32(binSamples[0][2]) +
						uint32(binSamples[len(binSamples)-1][1]-binSamples[0][1]),
				})
			}

			// Save to DB
			for i, sample := range dSamples {
				batchI := i % batchSize
				if batchI == 0 {
					// Beginning of batch
					buffer.Reset()
					buffer.WriteString(preamble)
				}

				buffer.WriteString(
					fmt.Sprintf("(%f, %d, %d, %d, %d)",
						sample.Power, sample.Freq, sample.Bandwidth, df, surveyID))

				if batchI == batchSize-1 || i == len(dSamples)-1 {
					// End of batch
					if _, err := tx.Exec(buffer.String()); err != nil {
						rollbackAndExit(err)
						return
					}
				} else {
					buffer.WriteRune(',')
				}
			}
		}

		if err := tx.Commit(); err != nil {
			rollbackAndExit(err)
			return
		}

		insertDone <- nil
	}(db, &incoming, insertDone)

	if insertErr := <-insertDone; insertErr != nil {
		c.String(http.StatusInternalServerError, "DB Error: %q", insertErr)
	} else {
		c.String(http.StatusOK, "OK")
	}
}

func TokenAuthMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authFailure := func(msg string) {
			c.String(401, msg)
			c.Abort()
		}

		token := c.Request.FormValue("token")

		if token == "" {
			authFailure("Missing required field: token")
			return
		}

		var uploaderId int
		if err := db.QueryRow(`
				SELECT id
				FROM uploader
				WHERE upload_key = $1`, token).Scan(&uploaderId); err != nil {
			authFailure("Invalid token")
			return
		}

		// uploader table has a before update trigger that updates
		// last_access column
		if _, err := db.Exec(`
				UPDATE uploader
				SET access_count = access_count + 1
				WHERE id = $1`, uploaderId); err != nil {
			c.String(500, "Internal server error (%s)", err)
			c.Abort()
			return
		}

		c.Request.Header.Set("X-Uploader-Id", strconv.Itoa(uploaderId))
		c.Next()
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
		decFactors := m.DefaultDecimationFactors()
		c.HTML(http.StatusOK, "index.tmpl.html", gin.H{
			"title":       "Spectrum Viewer",
			"dec_factors": decFactors,
		})
	})

	router.GET("/survey", func(c *gin.Context) {
		surveyHandler(c, db)
	})

	router.GET("/sample", func(c *gin.Context) {
		sampleHandler(c, db)
	})

	router.GET("/longsample", func(c *gin.Context) {
		longSampleHandler(c, db)
	})

	authorized := router.Group("/priv")
	authorized.Use(TokenAuthMiddleware(db))
	authorized.POST("/upload", func(c *gin.Context) {
		uploadHandler(c, db)
	})

	router.Run(":" + port)
}
