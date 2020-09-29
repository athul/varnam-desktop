package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/groupcache"
	"github.com/labstack/echo/v4"
	"github.com/subins2000/varnamd/libvarnam"
)

var errCacheSkipped = errors.New("cache skipped")

// Context which gets passed into the groupcache fill function
// Data will be set if the cache returns CacheSkipped
type varnamCacheContext struct {
	Data []byte
	context.Context
}

type standardResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	At      string `json:"at"`
}

func newStandardResponse() standardResponse {
	return standardResponse{Success: true, At: time.Now().UTC().String()}
}

type transliterationResponse struct {
	standardResponse
	Result []string `json:"result"`
	Input  string   `json:"input"`
}

type metaResponse struct {
	Result *libvarnam.CorpusDetails `json:"result"`
	standardResponse
}

type downloadResponse struct {
	Count int     `json:"count"`
	Words []*word `json:"words"`
	standardResponse
}

// Args to read.
type args struct {
	LangCode string `json:"lang"`
	Text     string `json:"text"`
}

//TrainArgs are the args for the Train Endpoint
type TrainArgs struct {
	Lang    string `json:"lang"`
	Pattern string `json:"pattern"`
	Word    string `json:"word"`
}

//DownloadLangArgs are the args for the language download endpoint
type DownloadLangArgs struct {
	Lang string `json:"lang"`
}

func handleStatus(c echo.Context) error {
	uptime := time.Since(startedAt)

	resp := struct {
		Version string `json:"version"`
		Uptime  string `json:"uptime"`
		standardResponse
	}{
		buildVersion + "-" + buildDate,
		uptime.String(),
		newStandardResponse(),
	}

	return c.JSON(http.StatusOK, resp)
}

func handleTransliteration(c echo.Context) error {
	var (
		langCode = c.Param("langCode")
		word     = c.Param("word")
		app      = c.Get("app").(*App)
	)

	words, err := app.cache.Get(langCode, word)
	if err != nil {
		w, err := transliterate(langCode, word)
		if err != nil {
			app.log.Printf("error in transliterationg, err: %s", err.Error())
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("error transliterating given string. message: %s", err.Error()))
		}

		words, _ = w.([]string)
		_ = app.cache.Set(langCode, word, words...)
	}

	return c.JSON(http.StatusOK, transliterationResponse{standardResponse: newStandardResponse(), Result: words, Input: word})
}

func handleReverseTransliteration(c echo.Context) error {
	var (
		langCode = c.Param("langCode")
		word     = c.Param("word")
		app      = c.Get("app").(*App)
	)

	result, err := app.cache.Get(langCode, word)
	if err != nil {
		res, err := reveseTransliterate(langCode, word)
		if err != nil {
			app.log.Printf("error in reverse transliterationg, err: %s", err.Error())
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("error transliterating given string. message: %s", err.Error()))
		}

		result = []string{res.(string)}
		_ = app.cache.Set(langCode, word, res.(string))
	}

	if len(result) <= 0 {
		app.log.Printf("no reverse transliteration found for lang: %s word: %s", langCode, word)
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("no transliteration found for lanugage: %s, word: %s", langCode, word))
	}

	response := struct {
		standardResponse
		Result string `json:"result"`
	}{
		newStandardResponse(),
		result[0],
	}

	return c.JSON(http.StatusOK, response)
}

func handleMetadata(c echo.Context) error {
	var (
		schemeIdentifier = c.Param("langCode")
		app              = c.Get("app").(*App)
	)

	data, err := getOrCreateHandler(schemeIdentifier, func(handle *libvarnam.Varnam) (data interface{}, err error) {
		details, err := handle.GetCorpusDetails()
		if err != nil {
			return nil, err
		}

		return &metaResponse{Result: details, standardResponse: newStandardResponse()}, nil
	})
	if err != nil {
		app.log.Printf("error in getting corpus details for: %s, err: %s", schemeIdentifier, err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("error getting metadata. message: %s", err.Error()))
	}

	return c.JSON(http.StatusOK, data)
}

func handleDownload(c echo.Context) error {
	var (
		langCode = c.Param("langCode")
		start, _ = strconv.Atoi(c.Param("downloadStart"))

		app = c.Get("app").(*App)
	)

	if start < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid parameter")
	}

	fillCache := func(ctx context.Context, key string, dest groupcache.Sink) error {
		// cache miss, fetch from DB
		// key is in the form <schemeIdentifier>+<downloadStart>
		parts := strings.Split(key, "+")
		schemeID := parts[0]
		downloadStart, _ := strconv.Atoi(parts[1])

		words, err := getWords(schemeID, downloadStart)
		if err != nil {
			return err
		}

		response := downloadResponse{Count: len(words), Words: words, standardResponse: newStandardResponse()}

		b, err := json.Marshal(response)
		if err != nil {
			return err
		}

		// gzipping the response so that it can be served directly
		var gb bytes.Buffer
		gWriter := gzip.NewWriter(&gb)

		defer func() { _ = gWriter.Close() }()

		_, _ = gWriter.Write(b)
		_ = gWriter.Flush()

		if len(words) < downloadPageSize {
			varnamCtx, _ := ctx.(*varnamCacheContext)
			varnamCtx.Data = gb.Bytes()

			return errCacheSkipped
		}

		_ = dest.SetBytes(gb.Bytes())

		return nil
	}

	once.Do(func() {
		// Making the groups for groupcache
		// There will be one group for each language
		for _, scheme := range schemeDetails {
			group := groupcache.GetGroup(scheme.Identifier)
			if group == nil {
				// 100MB max size for cache
				group = groupcache.NewGroup(scheme.Identifier, 100<<20, groupcache.GetterFunc(fillCache))
			}
			cacheGroups[scheme.Identifier] = group
		}
	})

	cacheGroup := cacheGroups[langCode]
	ctx := varnamCacheContext{}

	var data []byte
	if err := cacheGroup.Get(&ctx, fmt.Sprintf("%s+%d", langCode, start), groupcache.AllocatingByteSliceSink(&data)); err != nil {
		if err == errCacheSkipped {
			c.Response().Header().Set("Content-Encoding", "gzip")
			return c.Blob(http.StatusOK, "application/json; charset=utf-8", ctx.Data)
		}

		app.log.Printf("error in fetching deta from cache: %s, err: %s", langCode, err.Error())

		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("error getting metadata. message: %s", err.Error()))
	}

	c.Response().Header().Set("Content-Encoding", "gzip")

	return c.Blob(http.StatusOK, "application/json; charset=utf-8", data)
}

func handleLanguages(c echo.Context) error {
	return c.JSON(http.StatusOK, schemeDetails)
}

func handleLearn(c echo.Context) error {
	var (
		a args

		app = c.Get("app").(*App)
	)

	c.Request().Header.Set("Content-Type", "application/json")

	if err := c.Bind(&a); err != nil {
		app.log.Printf("error in binding request details for learn, err: %s", err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("error getting metadata. message: %s", err.Error()))
	}

	ch, ok := learnChannels[a.LangCode]
	if !ok {
		app.log.Printf("unknown language requested to learn: %s", a.LangCode)
		return echo.NewHTTPError(http.StatusBadRequest, "unable to find language")
	}

	go func(word string) { ch <- word }(a.Text)

	return c.JSON(http.StatusOK, "success")
}

func toggleDownloadEnabledStatus(langCode string, status bool) (interface{}, error) {
	if err := varnamdConfig.setDownloadStatus(langCode, status); err != nil {
		return nil, err
	}

	return newStandardResponse(), nil
}

func handleEnableDownload(c echo.Context) error {
	var (
		langCode = c.Param("langCode")

		app = c.Get("app").(*App)
	)

	data, err := toggleDownloadEnabledStatus(langCode, true)
	if err != nil {
		app.log.Printf("failed to toggle download enable, err: %s", err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("error getting metadata. message: %s", err.Error()))
	}

	return c.JSON(http.StatusOK, data)
}

func handleDisableDownload(c echo.Context) error {
	var (
		langCode = c.Param("langCode")
		app      = c.Get("app").(*App)
	)

	data, err := toggleDownloadEnabledStatus(langCode, false)
	if err != nil {
		app.log.Printf("failed to disable download, err: %s", err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("error getting metadata. message: %s", err.Error()))
	}

	return c.JSON(http.StatusOK, data)
}

// handleIndex is the root handler that renders the Javascript frontend.
func handleIndex(c echo.Context) error {
	app, _ := c.Get("app").(*App)

	b, err := app.fs.Read("/index.html")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("Content-Type", "text/html")

	return c.String(http.StatusOK, string(b))
}
func handleTrain(c echo.Context) error {
	var targs TrainArgs

	c.Request().Header.Set("Content-Type", "application/json")

	if err := c.Bind(&targs); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("error getting metadata. message: %s", err.Error()))
	}
	handle, err := libvarnam.Init(targs.Lang)
	if err != nil {
		return fmt.Errorf("failed to get data")
	}
	if err := handle.Train(targs.Pattern, targs.Word); err != nil {
		return fmt.Errorf("failed to Train %s. %s", targs.Word, err.Error())
	}
	return c.JSON(200, "Word Trained")

}

func handleDownloadLanguage(c echo.Context) error {
	var (
		args DownloadLangArgs

		app = c.Get("app").(*App)
	)

	if err := c.Bind(&args); err != nil {
		app.log.Printf("error in binding request details for learn, err: %s", err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("error getting metadata. message: %s", err.Error()))
	}

	fileURL := fmt.Sprintf("%s/languages/%s/download", varnamdConfig.upstream, args.Lang)
	filePath := libvarnam.GetSchemeFileDirectory() + "/" + args.Lang + ".vst"

	app.log.Printf("%s : %s", fileURL, filePath)

	resp, err := http.Get(fileURL)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	defer resp.Body.Close()

	out, err := os.Create(filePath)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)

	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.String(http.StatusOK, "success")
}

func handleGetUpstreamURL(c echo.Context) error {
	return c.String(http.StatusOK, varnamdConfig.upstream)
}
