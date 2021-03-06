package marketdata

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"os"
	"time"
	"fmt"
	"path"
	"io/ioutil"
	"sort"
)

func getSymbolMetaMock() *JsonSymbolMeta {
	storedDates := []time.Time{
		timeOnTheFly(2010, 1, 1),
		timeOnTheFly(2010, 1, 2),
		timeOnTheFly(2010, 1, 3),
		timeOnTheFly(2010, 1, 4),
		timeOnTheFly(2010, 1, 5),
		timeOnTheFly(2010, 1, 1),
		timeOnTheFly(2010, 1, 1),
		timeOnTheFly(2010, 1, 1),
		timeOnTheFly(2010, 1, 1),
		timeOnTheFly(2010, 1, 2),
		timeOnTheFly(2010, 1, 30),
	}

	symbMeta := JsonSymbolMeta{
		"TEST",
		"D",
		storedDates,
		true,
	}

	symbMeta.save("test_data/daily_ranges/candles/day/.meta/TEST.json")

	return &symbMeta

}

func TestJsonSymbolMeta_getEmptyDates(t *testing.T) {
	jsonMeta := JsonSymbolMeta{}
	rng := DateRange{timeOnTheFly(2010, 1, 1), timeOnTheFly(2012, 1, 1)}

	emptyDates, err := jsonMeta.getEmptyDates(&rng)
	if err != nil {
		t.Fatal(err)
	}

	emptyDatesSet := make(map[time.Time]struct{})
	for _, d := range emptyDates {
		_, ok := emptyDatesSet[d]
		if ok {
			t.Fatal("Duplicate date", d)
		} else {
			emptyDatesSet[d] = struct{}{}
		}
	}

	jsonMeta = JsonSymbolMeta{}
	jsonMeta.HasWeekends = true

	listedDates := []time.Time{
		timeOnTheFly(2018, 11, 28),
		timeOnTheFly(2018, 11, 30),
		timeOnTheFly(2018, 11, 23),
		timeOnTheFly(2018, 11, 19),
	}

	jsonMeta.ListedDates = listedDates

	rng = DateRange{timeOnTheFly(2018, 11, 15), timeOnTheFly(2018, 12, 1)}

	emptyDates, err = jsonMeta.getEmptyDates(&rng)
	if err != nil {
		t.Fatal(err)
	}

	expecting := []time.Time{
		timeOnTheFly(2018, 11, 15),
		timeOnTheFly(2018, 11, 16),
		timeOnTheFly(2018, 11, 20),
		timeOnTheFly(2018, 11, 21),
		timeOnTheFly(2018, 11, 22),
		timeOnTheFly(2018, 11, 26),
		timeOnTheFly(2018, 11, 27),
		timeOnTheFly(2018, 11, 29),
	}

	for i, v := range emptyDates {
		assert.Equal(t, v, expecting[i])
	}

	jsonMeta.HasWeekends = false

	emptyDates, err = jsonMeta.getEmptyDates(&rng)
	if err != nil {
		t.Fatal(err)
	}

	expecting = []time.Time{
		timeOnTheFly(2018, 11, 15),
		timeOnTheFly(2018, 11, 16),
		timeOnTheFly(2018, 11, 17),
		timeOnTheFly(2018, 11, 18),
		timeOnTheFly(2018, 11, 20),
		timeOnTheFly(2018, 11, 21),
		timeOnTheFly(2018, 11, 22),
		timeOnTheFly(2018, 11, 24),
		timeOnTheFly(2018, 11, 25),
		timeOnTheFly(2018, 11, 26),
		timeOnTheFly(2018, 11, 27),
		timeOnTheFly(2018, 11, 29),
		timeOnTheFly(2018, 12, 1),
	}

	for i, v := range emptyDates {
		assert.Equal(t, v, expecting[i])
	}

}

func TestJsonSymbolMeta_getEmptyRanges(t *testing.T) {
	//TODO not passed

	reqRange := DateRange{
		timeOnTheFly(2009, 12, 15),
		timeOnTheFly(2010, 1, 25),
	}

	expectedRanges := []*DateRange{
		{timeOnTheFly(2009, 12, 15), timeOnTheFly(2009, 12, 31)},
		{timeOnTheFly(2010, 1, 6), timeOnTheFly(2010, 1, 14)},
		{timeOnTheFly(2010, 1, 17), timeOnTheFly(2010, 1, 17)},
		{timeOnTheFly(2010, 1, 20), timeOnTheFly(2010, 1, 21)},
		{timeOnTheFly(2010, 1, 23), timeOnTheFly(2010, 1, 25)},
	}

	symbMeta := getSymbolMetaMock()

	empty, err := symbMeta.getEmptyRanges(&reqRange)
	if err != nil {
		fmt.Println(err)
	}
	for i, a := range empty {
		assert.Equal(t, *expectedRanges[i], *a)
	}

}

func TestJsonSymbolMeta_lastDate(t *testing.T) {
	symbMeta := getSymbolMetaMock()
	expected := timeOnTheFly(2010, 1, 30)

	actual, ok := symbMeta.lastDate()

	if !ok {
		t.Fatal("Result is not ok. Max date is zero")
	}

	assert.Equal(t, expected, actual)

}

func TestJsonSymbolMeta_firstDate(t *testing.T) {
	symbMeta := getSymbolMetaMock()
	expected := timeOnTheFly(2010, 1, 1)

	actual, ok := symbMeta.firstDate()

	if !ok {
		t.Fatal("Result is not ok. Date is more than today")
	}

	assert.Equal(t, expected, actual)

}

func TestJsonSymbolMeta_Save(t *testing.T) {
	pth := "./test_data/symbol_meta/test_save.json"
	defer os.Remove(pth)
	meta := getSymbolMetaMock()
	fmt.Println(meta)
	err := meta.save(pth)
	if err != nil {
		t.Fatal(err)
	}

	loadedMeta := JsonSymbolMeta{}
	loadedMeta.Load(pth)

	assert.True(t, len(loadedMeta.ListedDates) > 2)

	expectingDate := timeOnTheFly(2010, 01, 01)

	assert.Equal(t, loadedMeta.ListedDates[0], expectingDate)

}

func TestJsonStorage_findDailyRangeToDownload(t *testing.T) {
	at := mockActiveTick()
	testDir := "./test_data/daily_ranges"
	testSymbol := "TEST"
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	storage := JsonStorage{
		5,
		testDir,
		at,
		loc,
		true,
	}

	getSymbolMetaMock()
	range1 := DateRange{
		timeOnTheFly(2010, 1, 30),
		timeOnTheFly(2010, 5, 30),
	}

	actual1, err := storage.findDailyRangeToDownload(&range1, testSymbol)
	if err != nil {
		t.Fatal(err)
	}

	expected1 := DateRange{
		timeOnTheFly(2010, 1, 1),
		timeOnTheFly(2010, 5, 30),
	}

	assert.Equal(t, expected1, *actual1)

	range2 := DateRange{
		timeOnTheFly(2009, 1, 30),
		timeOnTheFly(2009, 5, 30),
	}

	actual2, err := storage.findDailyRangeToDownload(&range2, testSymbol)
	if err != nil {
		t.Fatal(err)
	}
	expected2 := DateRange{
		timeOnTheFly(2009, 1, 30),
		timeOnTheFly(2010, 1, 30),
	}

	assert.Equal(t, expected2, *actual2)

	range3 := DateRange{
		timeOnTheFly(2010, 1, 15),
		timeOnTheFly(2010, 1, 20),
	}

	_, err = storage.findDailyRangeToDownload(&range3, testSymbol)
	if err == nil {
		t.Fatal("should be error: ErrNothingToDownload")
	} else {
		switch err.(type) {
		case *ErrNothingToDownload:
			fmt.Println("Got expected error. OK!")
			return
		default:
			t.Fatal("should be error: ErrNothingToDownload")
		}
	}

}

func TestJsonStorage_ensureFolder(t *testing.T) {
	defer os.RemoveAll("./test_storage")
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	s := JsonStorage{
		3,
		"./test_storage",
		mockActiveTick(),
		loc,
		true,
	}

	err = s.createFolders()

	assert.Nil(t, err)

}

func TestJsonStorage_readCandlesFromFile(t *testing.T) {
	pth := "./test_data/candles.json"
	storage := JsonStorage{}
	candles, err := storage.readCandlesFromFile(pth)

	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, len(candles) > 1)

}

func TestJsonStorage_saveCandlesToFile(t *testing.T) {
	defer os.Remove("./test_data/save_test.json")

	at := mockActiveTick()
	dRange := DateRange{timeOnTheFly(2010, 1, 1),
		timeOnTheFly(2012, 5, 3)}
	candles, err := at.GetCandles("SPY", "D", dRange)
	if err != nil {
		t.Fatal(err)
	}

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	storage := JsonStorage{
		3,
		"./test_data",
		at,
		loc,
		true,
	}

	err = storage.saveCandlesToFile(&candles, "./test_data/save_test.json")
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, fileExists("./test_data/save_test.json"))
}

func TestJsonStorage_saveAndLoadCandles(t *testing.T) {
	defer os.Remove("./test_data/TEST_read_write.json")

	at := mockActiveTick()
	dRange := DateRange{timeOnTheFly(2010, 1, 1),
		timeOnTheFly(2012, 5, 3)}
	candles, err := at.GetCandles("SPY", "D", dRange)
	if err != nil {
		t.Fatal(err)
	}

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	storage := JsonStorage{
		3,
		"./test_data",
		at,
		loc,
		true,
	}

	err = storage.saveCandlesToFile(&candles, "./test_data/TEST_read_write.json")
	if err != nil {
		t.Fatal(err)
	}

	loadedCandles, err := storage.readCandlesFromFile("./test_data/TEST_read_write.json")

	if err != nil {
		t.Fatal(err)
	}

	for i, v := range candles {
		assert.Equal(t, *v, *loadedCandles[i])
	}

}

func TestJsonStorage_updateDailyCandles(t *testing.T) {
	testDir := "./test_data/json_storage"
	os.RemoveAll(testDir)

	at := mockActiveTick()
	createDirIfNotExists(testDir)

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	storage := JsonStorage{
		3,
		testDir,
		at,
		loc,
		true,
	}

	//storage.createFolders()

	range1 := DateRange{timeOnTheFly(2010, 1, 1), timeOnTheFly(2011, 1, 1)}

	err = storage.updateDailyCandles("SPY", &range1)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, fileExists(path.Join(testDir, "candles/day", "SPY.json")))
	assert.True(t, fileExists(path.Join(testDir, "candles/day/.meta", "SPY.json")))

	range2 := DateRange{timeOnTheFly(2010, 5, 1), timeOnTheFly(2015, 1, 1)}

	err2 := storage.updateDailyCandles("SPY", &range2)
	if err2 != nil {
		t.Fatal(err2)
	}

	candles, err := storage.readCandlesFromFile(path.Join(testDir, "candles/day", "SPY.json"))
	if err != nil {
		t.Fatal(err)
	}

	realRange := DateRange{timeOnTheFly(2010, 1, 1), timeOnTheFly(2015, 1, 1)}

	datasourceCandles, err := at.GetCandles("SPY", "D", realRange)

	if err != nil {
		t.Fatal(err)
	}

	for i, v := range datasourceCandles {
		assert.Equal(t, *candles[i], *v)
	}
}

func TestJsonStorage_updateTicks(t *testing.T) {
	testDir := "./test_data/json_storage"
	os.RemoveAll(testDir)

	at := mockActiveTick()
	createDirIfNotExists(testDir)

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	startTime := TimeOfDay{}
	endTime := TimeOfDay{11, 10, 0}

	storage := JsonStorage{
		5,
		testDir,
		at,
		loc,
		true,
	}

	start := timeOnTheFly(2018, 10, 1)
	end := timeOnTheFly(2018, 10, 15)

	params := TickUpdateParams{
		"GJH",
		start,
		end,
		startTime,
		endTime,
		true,
		true,
	}

	err = storage.UpdateSymbolTicks(params)

	if err != nil {
		t.Fatal(err)
	}

	folderSymbol := path.Join(testDir, "ticks/quotes_trades/GJH")

	// Map files mod times. Request again bigger range. Files that already stored shouldn't be requested again

	modTimes := make(map[string]time.Time)

	pathMeta := path.Join(testDir, "ticks/quotes_trades/.meta", "GJH.json")
	fi, err := os.Stat(pathMeta)
	if err != nil {
		t.Fatal(err)
	}
	metaModTime := fi.ModTime()

	files, err := ioutil.ReadDir(folderSymbol)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		modTimes[f.Name()] = f.ModTime()
	}

	start = timeOnTheFly(2018, 9, 25)
	end = timeOnTheFly(2018, 10, 18)

	params = TickUpdateParams{
		"GJH",
		start,
		end,
		startTime,
		endTime,
		true,
		true,
	}

	err = storage.UpdateSymbolTicks(params)

	files, err = ioutil.ReadDir(folderSymbol)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		modTimeExpected, ok := modTimes[f.Name()]

		if !ok {
			continue
		}

		assert.Equal(t, f.ModTime(), modTimeExpected)
	}

	fi, err = os.Stat(pathMeta)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, fi.ModTime().After(metaModTime)) // Check if update time of metafile changed

	// Check if we have some quotes that ActiveTick gives us

	for {
		if start.After(end) {
			break
		}

		if start.Weekday() == 6 || start.Weekday() == 0 {
			start = start.AddDate(0, 0, 1)
			continue
		}

		dateFileName := path.Join(folderSymbol, start.Format(tickfilelayout)+".json")
		storedTicks, err := storage.readTicksFromFile(dateFileName)
		if err != nil {
			t.Fatal(err)
		}

		rng := DateRange{
			time.Date(start.Year(), start.Month(), start.Day(), startTime.Hour, startTime.Minute, startTime.Second, 0, time.UTC),
			time.Date(start.Year(), start.Month(), start.Day(), endTime.Hour, endTime.Minute, endTime.Second, 0, time.UTC),
		}
		atTicks, err := at.GetTicks("GJH", rng, true, true)
		if err != nil {
			t.Fatal(err)

		}

		for i, v := range *storedTicks {
			assert.Equal(t, v.Datetime, atTicks[i].Datetime)
		}

		start = start.AddDate(0, 0, 1)

	}

}

func TestJsonStorage_getLoadedTickDates(t *testing.T) {
	storage := JsonStorage{}
	dates, err := storage.getStoredTickDates("C:\\Users\\alex1\\go\\src\\alex\\marketdata\\test_data\\json_storage\\ticks\\GJH")
	fmt.Println(err)
	for _, d := range dates {
		fmt.Println(d.Format("2006-01-02"))
	}
}

func TestJsonStorage_GetStoredTicks(t *testing.T) {
	testDir := "./test_data/json_storage"
	os.RemoveAll(testDir)

	at := mockActiveTick()
	createDirIfNotExists(testDir)

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	startTime := TimeOfDay{}
	endTime := TimeOfDay{11, 10, 0}

	storage := JsonStorage{
		2,
		testDir,
		at,
		loc,
		true,
	}

	start := timeOnTheFly(2018, 10, 1)
	end := timeOnTheFly(2018, 10, 15)

	params := TickUpdateParams{
		"GJH",
		start,
		end,
		startTime,
		endTime,
		true,
		true,
	}

	err = storage.UpdateSymbolTicks(params)

	if err != nil {
		t.Fatal(err)
	}

	ticks, err := storage.GetStoredTicks("GJH", DateRange{start, end}, true, true)

	if err != nil {
		t.Fatal(err)
	}

	datesFound := 0
	for i, t := range ticks {
		if i == 0 {
			continue
		}

		if t.Datetime.Weekday() != ticks[i-1].Datetime.Weekday() {
			datesFound++
		}
	}

	sorted := sort.SliceIsSorted(ticks, func(i, j int) bool {
		return ticks[i].Datetime.Unix() < ticks[j].Datetime.Unix()
	})

	assert.True(t, sorted)
	assert.Equal(t, 10, datesFound)

	d := start
	for {
		rng := DateRange{
			From: time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC),
			To:   time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, time.UTC),
		}
		if d.Weekday() == 6 || d.Weekday() == 0 {
			d = d.AddDate(0, 0, 1)
			if d.After(end) {
				break
			}
			continue
		}

		ticks, err := storage.GetStoredTicks("GJH", rng, true, true)

		if err != nil {
			t.Fatal(err)
		}

		for _, t_ := range ticks {
			assert.Equal(t, t_.Datetime.Weekday(), d.Weekday())
		}

		d = d.AddDate(0, 0, 1)
		if d.After(end) {
			break
		}

	}
}
