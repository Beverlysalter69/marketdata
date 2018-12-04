package marketdata

import (
	"time"
	"os"
	"path"
	"strconv"
	"github.com/pkg/errors"
	"encoding/json"
	"io/ioutil"
	"fmt"
	"path/filepath"
	"sync"
	"context"

	"strings"
)

const (
	tickfilelayout = "2006-01-02"
)

type tickRequestParams struct {
	trades bool
	quotes bool
	date   time.Time
	symbol string
}

type errNothingToDownload struct {
}

func (*errNothingToDownload) Error() string {
	return "Nothing to download"
}

// Public request params
type TickUpdateParams struct {
	Symbol         string
	FromDate       time.Time
	ToDate         time.Time
	Quotes         bool
	Trades         bool
	UpdateWeekends bool
}

func (p *TickUpdateParams) checkErrors() error {
	if !p.Trades && !p.Quotes {
		return errors.New("Wrong parameters. Should be selected trades, quotes or both")
	}

	if p.FromDate.After(p.ToDate) {
		return errors.New("From date should be less than to date")
	}

	if p.Symbol == "" {
		return errors.New("Symbol not specified")
	}

	return nil

}

func (p *TickUpdateParams) modifyTimes() {
	p.ToDate = time.Date(p.ToDate.Year(), p.ToDate.Month(), p.ToDate.Day(), 0, 0, 0, 0, time.UTC)
	p.FromDate = time.Date(p.FromDate.Year(), p.FromDate.Month(), p.FromDate.Day(), 0, 0, 0, 0, time.UTC)
}

type CandlesUpdateParams struct {
	Symbol         string
	TimeFrame      string
	FromDate       time.Time
	ToDate         time.Time
	UpdateWeekends bool
}

func (p *CandlesUpdateParams) modifyTimes() {
	p.ToDate = time.Date(p.ToDate.Year(), p.ToDate.Month(), p.ToDate.Day(), 0, 0, 0, 0, time.UTC)
	p.FromDate = time.Date(p.FromDate.Year(), p.FromDate.Month(), p.FromDate.Day(), 0, 0, 0, 0, time.UTC)
}

func (p *CandlesUpdateParams) checkErrors() error {
	if p.FromDate.After(p.ToDate) {
		return errors.New("From date should be less than to date")
	}

	if p.Symbol == "" {
		return errors.New("Symbol not specified")
	}

	//Todo check timeframe here
	return nil
}

// Symbol meta information *************************************************

type JsonSymbolMeta struct {
	Symbol      string
	TimeFrame   string
	ListedDates []time.Time
}

func (j *JsonSymbolMeta) Load(loadPath string) error {
	jsonFile, err := os.Open(loadPath)

	if err != nil {
		return err
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, j)

	if err != nil {
		return err
	}

	return nil

}

func (j *JsonSymbolMeta) datesSet() map[time.Time]struct{} {
	dates := make(map[time.Time]struct{})
	for _, v := range j.ListedDates {
		dates[v] = struct{}{}
	}
	return dates
}

func (j *JsonSymbolMeta) getEmptyRanges(rng *DateRange) ([]*DateRange, error) {
	emptyDates, err := j.getEmptyDates(rng)
	if err != nil {
		return nil, err
	}

	if len(emptyDates) == 0 {
		return nil, nil //ToDo should I return error here
	}

	if len(emptyDates) == 1 {
		rng := DateRange{
			emptyDates[0],
			emptyDates[0],
		}

		return []*DateRange{&rng}, nil
	}

	start, end := emptyDates[0], emptyDates[1]
	var emptyRanges []*DateRange

	for i, v := range emptyDates {
		if i < 2 {
			continue
		}

		delta := int(v.Sub(end).Hours() / 24)
		if delta > 1 {
			rng := DateRange{start, end}
			emptyRanges = append(emptyRanges, &rng)
			start, end = v, v
			continue
		}
		end = v

	}

	rngF := DateRange{start, end}
	emptyRanges = append(emptyRanges, &rngF)

	return emptyRanges, nil
}

func (j *JsonSymbolMeta) getEmptyDates(rng *DateRange) ([]time.Time, error) {
	last := rng.from

	var emptyDates []time.Time
	datesSet := j.datesSet()

	if rng.from.Equal(rng.to) {
		if _, ok := datesSet[rng.from]; ok {
			return emptyDates, nil
		}
		emptyDates = append(emptyDates, rng.from)
		return emptyDates, nil
	}

	for {
		if last.After(rng.to) {
			break
		}
		_, ok := datesSet[last]

		if !ok {
			emptyDates = append(emptyDates, last)
		}

		last = last.AddDate(0, 0, 1)

	}

	return emptyDates, nil

}

func (j *JsonSymbolMeta) firstDate() (time.Time, bool) {
	minTime := time.Now().AddDate(0, 0, 1)
	for _, k := range j.ListedDates {
		if k.Unix() < minTime.Unix() {
			minTime = k
		}

	}
	if minTime.After(time.Now()) {
		return minTime, false
	}
	return minTime, true
}

func (j *JsonSymbolMeta) lastDate() (time.Time, bool) {
	maxTime := time.Time{}

	for _, k := range j.ListedDates {
		if k.Unix() > maxTime.Unix() {
			maxTime = k
		}
	}
	if maxTime.IsZero() {
		return maxTime, false
	}
	return maxTime, true

}

func (j *JsonSymbolMeta) save(savePath string) error {
	dirName := filepath.Dir(savePath)
	err := createDirIfNotExists(dirName)
	if err != nil {
		return err
	}

	json_, err := json.Marshal(j)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(savePath, json_, 0644)

	return err
}

func loadMetaIfExists(metaPath string) *JsonSymbolMeta {
	jsonMeta := JsonSymbolMeta{}

	if fileExists(metaPath) {
		jsonMeta.Load(metaPath)

	}

	return &jsonMeta

}

// Storage code starts here **************************************************

type JsonStorage struct {
	updateWorkers int
	path          string
	provider      HistoryProvider
}

func (p *JsonStorage) createFolders() error {
	ticksFolder := path.Join(p.path, "ticks")
	candlesFolder := path.Join(p.path, "candles")

	err := createDirIfNotExists(candlesFolder)
	if err != nil {
		return err
	}

	err = createDirIfNotExists(ticksFolder)
	if err != nil {
		return err
	}

	return nil

}

func (p *JsonStorage) GetStoredCandles(symbol string, tf string, dRange DateRange) (*CandleArray, error) {
	return nil, nil
}

func (p *JsonStorage) GetStoredTicks(symbol string, dRange DateRange) (*TickArray, error) {
	return nil, nil
}

func (p *JsonStorage) UpdateSymbolCandles(params CandlesUpdateParams) error {
	err := params.checkErrors()
	params.modifyTimes()

	if err != nil {
		return err
	}
	dRange := DateRange{
		params.FromDate, params.ToDate,
	}
	switch params.TimeFrame {
	case "D":
		return p.updateDailyCandles(params.Symbol, &dRange)
	case "W":
		return p.updateWeeklyCandles(params.Symbol, &dRange)

	default:
		minutes, err := strconv.Atoi(params.TimeFrame)
		if err != nil {
			return errors.New("Can't recognize timeframe. Should be D, W, Tick or Intraday Minutes (1-60)")
		}

		if minutes < 1 || minutes > 60 {
			return errors.New("Intraday minutes should be in range 1-60")
		}

		return p.updateIntradayCandles(minutes, params.Symbol, &dRange)

	}

	return nil

}

func (p *JsonStorage) saveCandlesToFile(candles *CandleArray, savePath string) error {
	dirName := filepath.Dir(savePath)
	err := createDirIfNotExists(dirName)
	if err != nil {
		return err
	}

	json_, err := json.Marshal(candles)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(savePath, json_, 0644)
	if err != nil {
		fmt.Println(fmt.Sprintf("Can't write candles to file: %v", err))
	}

	return err
}

func (*JsonStorage) readCandlesFromFile(path string) (*CandleArray, error) {
	if !fileExists(path) {
		return nil, nil //Todo what error?
	}

	jsonFile, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var candles CandleArray

	err = json.Unmarshal(byteValue, &candles)

	if err != nil {
		return nil, err
	}

	return &candles, err

}

func (p *JsonStorage) updateDailyCandles(s string, dRange *DateRange) error {

	downloadRange, err := p.findDailyRangeToDownload(dRange, s)
	if err != nil {
		switch err.(type) {
		case *errNothingToDownload:
			return nil
		default:
			return err
		}
	}

	candles, err1 := p.provider.GetCandles(s, "D", *downloadRange)
	if err1 != nil {
		return err1
	}

	savePath := path.Join(p.path, "candles/day", s+".json")
	err2 := p.saveCandlesToFile(&candles, savePath)

	if err2 != nil {
		return err2
	}

	newMeta := p.genNewDailySymbolMeta(s, downloadRange)
	metaPath := path.Join(p.path, "candles/day/.meta", s+".json")
	err3 := newMeta.save(metaPath)
	if err3 != nil {
		fmt.Println(err3)
		return err3
	}
	return nil

}

func (p *JsonStorage) findDailyRangeToDownload(dRange *DateRange, symbol string) (*DateRange, error) {
	metaPath := path.Join(p.path, "candles/day/.meta", symbol+".json")
	downloadRange := *dRange

	if !fileExists(metaPath) {
		return &downloadRange, nil
	}

	symbolMeta := JsonSymbolMeta{}
	err := symbolMeta.Load(metaPath)
	if err != nil {
		return nil, err
	}

	firstListed, ok1 := symbolMeta.firstDate()
	lastListed, ok2 := symbolMeta.lastDate()

	if !ok1 && !ok2 {
		return &downloadRange, nil
	}

	if firstListed.Unix() < dRange.from.Unix() && lastListed.Unix() > dRange.to.Unix() && ok1 && ok2 {
		// If we already have all candles in this date range just return without errors
		return nil, &errNothingToDownload{}
	}

	if dRange.from.Unix() > firstListed.Unix() && ok1 {
		downloadRange.from = firstListed
	}

	if dRange.to.Unix() < lastListed.Unix() && ok2 {
		downloadRange.to = lastListed
	}

	return &downloadRange, nil

}

func (p *JsonStorage) genNewDailySymbolMeta(symbol string, dateRange *DateRange) *JsonSymbolMeta {
	var listedDates []time.Time
	lastD := dateRange.from
	for {
		if lastD.After(dateRange.to) {
			break
		}
		listedDates = append(listedDates, lastD)
		lastD = lastD.AddDate(0, 0, 1)

	}

	symbolMeta := JsonSymbolMeta{
		symbol,
		"D",
		listedDates,
	}

	return &symbolMeta

}

func (p *JsonStorage) updateWeeklyCandles(s string, dRange *DateRange) error {
	return nil

}

func (p *JsonStorage) updateIntradayCandles(minutes int, s string, dRange *DateRange) error {
	return nil
}

func (p *JsonStorage) UpdateSymbolTicks(params TickUpdateParams) error {
	err := params.checkErrors()
	params.modifyTimes()
	if err != nil {
		return err
	}

	folderName := p.generateTicksFolderName(params.Quotes, params.Trades)
	metaPath := path.Join(p.path, "ticks", folderName, ".meta", params.Symbol+".json")

	jsonMeta := loadMetaIfExists(metaPath)
	dRange := DateRange{params.FromDate, params.ToDate}

	emptyDates, err := jsonMeta.getEmptyDates(&dRange)

	if err != nil {
		return err
	}

	if emptyDates == nil {
		return errors.Wrapf(&errNothingToDownload{}, "UpdateSymbolTicks() Symbol: %v dRange: %v", params.Symbol, &dRange)
	}

	wg := &sync.WaitGroup{}

	datesChan := make(chan tickRequestParams, 2)
	errorsChan := make(chan error)
	successChan := make(chan struct{}, 1)
	ctx, finish := context.WithCancel(context.Background())

	defer func() {
		close(errorsChan)
		close(successChan)

	}()

	var retError error

	//Workers pool
	for i := 0; i < p.updateWorkers; i++ {
		go func() {
			wg.Add(1)
			p.tickUpdateWorker(ctx, datesChan, errorsChan, successChan)
			fmt.Println("Done")
			wg.Done()
		}()
	}

	// Requests producer
	go func() {
		defer close(datesChan)
		for _, d := range emptyDates {

			params := tickRequestParams{
				params.Trades,
				params.Quotes,
				d,
				params.Symbol,
			}

			datesChan <- params
		}

	}()

	counter := 0

LoadingLoop:
	for {

		select {
		case err, ok := <-errorsChan:
			if !ok {
				continue LoadingLoop
			}
			errorsChan = nil
			datesChan = nil
			finish()
			return err
		case <-successChan:
			counter++
			fmt.Println(counter, len(emptyDates))
			if counter == len(emptyDates) {
				finish()
				break LoadingLoop
			}

		default:
			if counter == len(emptyDates) {
				finish()
				break LoadingLoop
			}

		}
	}

	wg.Wait()

	storageFolder := path.Join(p.path, "ticks", folderName, params.Symbol)
	listedDates, err := p.getStoredTickDates(storageFolder)
	if err != nil {
		return err
	}
	jsonMeta.ListedDates = listedDates
	jsonMeta.save(metaPath)

	return retError
}

func (p *JsonStorage) tickUpdateWorker(ctx context.Context, params chan tickRequestParams, errorsChan chan<- error,
	successChan chan<- struct{}) {
LOOP:
	for {
		select {
		case <-ctx.Done():
			return
		case par, ok := <-params:

			if !ok {
				return
			}

			d := par.date
			r := DateRange{}
			r.from = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
			r.to = time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 59, time.UTC)

			folderName := p.generateTicksFolderName(par.quotes, par.trades)

			savePath := path.Join(p.path, "ticks", folderName, par.symbol, par.date.Format(tickfilelayout)+".json")

			ticks, err := p.provider.GetTicks(par.symbol, r, par.quotes, par.trades)
			if err != nil {
				switch errors.Cause(err).(type) {
				case *ErrEmptyResponse:
					err = p.saveTicksToFile(&ticks, savePath)
					if err != nil {
						errorsChan <- err
						return
					}
					successChan <- struct{}{}
					continue LOOP

				default:
					errorsChan <- err
					return

				}
			}

			err = p.saveTicksToFile(&ticks, savePath)

			if err != nil {
				errorsChan <- err
				return
			}

			successChan <- struct{}{}

		}

	}
}

func (*JsonStorage) generateTicksFolderName(quotes bool, trades bool) string {
	folderName := ""
	if quotes {
		folderName += "quotes"
	}
	if trades {
		if folderName != "" {
			folderName += "_trades"
		} else {
			folderName += "trades"
		}
	}

	return folderName
}

func (p *JsonStorage) saveTicksToFile(ticks *TickArray, savePath string) error {
	dirName := filepath.Dir(savePath)
	err := createDirIfNotExists(dirName)
	if err != nil {
		return err
	}

	json_, err := json.Marshal(ticks)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(savePath, json_, 0644)
	if err != nil {
		fmt.Println(fmt.Sprintf("Can't write ticks to file: %v", err))
	}

	return err
}

func (*JsonStorage) readTicksFromFile(path string) (*TickArray, error) {
	if !fileExists(path) {
		return nil, nil //Todo what error?
	}

	jsonFile, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var ticks TickArray

	err = json.Unmarshal(byteValue, &ticks)

	if err != nil {
		return nil, err
	}

	return &ticks, err

}

func (p *JsonStorage) getStoredTickDates(pth string) ([]time.Time, error) {
	var listed []time.Time
	files, err := ioutil.ReadDir(pth)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		filename := strings.Split(f.Name(), ".")[0]
		t, err := time.Parse(tickfilelayout, filename)
		if err != nil {
			// Todo log???
			continue
		}
		listed = append(listed, t)
	}

	return listed, nil

}
